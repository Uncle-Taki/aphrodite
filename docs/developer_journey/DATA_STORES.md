# Data stores and ownership

This project uses several storage systems. They are deliberately split by
ownership: the Go BFF owns application/business data, while Strapi owns
editorial content and its CMS metadata.

## At a glance

| Store | Local service | Owner | What it stores | What it does not store |
|---|---|---|---|---|
| Application PostgreSQL | `postgres` | Go application | Users, posts, comments, and CMS sessions | Strapi editorial documents and uploaded media |
| Strapi PostgreSQL | `strapi-postgres` | Strapi | Strapi content records, localization, relations, admin/plugin data, and upload metadata | The actual media bytes when S3/MinIO upload is enabled |
| Redis | `redis` | Go application and worker | Strapi response cache and Asynq job queues | Durable source-of-truth records |
| MinIO / S3-compatible object storage | `minio` locally | Strapi upload plugin and Go media adapters | Original uploaded files and other binary objects | Relational metadata, content fields, or job state |
| imgproxy | `imgproxy` | Image delivery layer | No durable application data; reads objects and produces transformed images | Original files and media metadata |

Production may replace MinIO with AWS S3, Cloudflare R2, or another S3-compatible
provider. The Go adapter uses the MinIO SDK because these providers expose the
same S3 API.

## Application PostgreSQL and GORM

The Go service connects to the database configured by `POSTGRES_HOST`,
`POSTGRES_DB`, `POSTGRES_USER`, and `POSTGRES_PASSWORD`. GORM is the persistence
library used by Go repositories; it is not a separate database.

The Go-owned tables are:

- `users`: registered users, password hashes, roles, phone numbers, timestamps,
  and the `disabled` flag.
- `posts`: posts authored by Go users.
- `comments`: comments attached to Go posts and authors.
- `cms_sessions`: hashed, expiring CMS login sessions for admin/editor access.

The row mappings live under `internal/*/infra/postgres`. Repositories and
use-cases should treat these tables as the source of truth for authentication,
authorization, posts, comments, and sessions.

The API initializes these four tables with GORM `AutoMigrate` during startup.
There is no separate Aphrodite migration command. AutoMigrate creates and
updates the schema represented by the Go row types, including their declared
indexes and relationships; it does not reproduce arbitrary SQL or perform
destructive cleanup of legacy tables.

## Strapi PostgreSQL

Strapi connects to the separate `strapi-postgres` service using the
`STRAPI_DB_*` settings. This database is intentionally separate from the Go
application database.

Strapi owns:

- editorial content types such as articles, recipes, categories, products, and
  newsletters;
- localized entries and publication state;
- relations between editorial entries;
- Strapi users/admin/plugin tables; and
- upload-library metadata (file name, MIME type, dimensions, provider key, and
  related metadata).

The Go BFF does not query Strapi's database directly. It calls Strapi's HTTP API
through `internal/content/strapi`, passing the server-side `STRAPI_API_TOKEN`.
The public BFF routes (`/v1/articles/...`, `/v1/recipes/...`, and the other
content routes) therefore expose published, localized content without exposing
Strapi credentials or requiring clients to know Strapi's URL.

Strapi applies its own schema management when it starts. The API waits for the
Strapi health endpoint before serving requests, but Go migrations and GORM
AutoMigrate never target `strapi-postgres` tables.

## Redis

The Go Redis client connects to `REDIS_ADDR` and uses `REDIS_DB` and
`REDIS_PASSWORD`.

Redis has two roles:

1. **Content cache**: `internal/content/cache.go` caches successful Strapi slug
   lookups under keys beginning with `content:strapi:`. Entries are disposable
   and expire according to `STRAPI_CACHE_TTL`.
2. **Background jobs**: `internal/shared/jobs` uses Asynq on the same Redis
   service for queue metadata, task payloads, retries, and leases. Queue names
   and worker concurrency come from `WORKER_QUEUE` and `WORKER_CONCURRENCY`.

Redis data can be lost and rebuilt. It must not be used as the canonical store
for users, posts, comments, editorial content, or media files.

## MinIO / S3 object storage

The local Compose stack runs MinIO with the `minio_data` volume. The relevant
settings are `MEDIA_ENDPOINT`, `MEDIA_BUCKET`, `MEDIA_ACCESS_KEY`,
`MEDIA_SECRET_KEY`, `MEDIA_REGION`, and `MEDIA_FORCE_PATH_STYLE`.

Strapi's AWS S3 upload provider writes uploaded media bytes to the configured
bucket. Strapi PostgreSQL keeps the metadata and object key; MinIO keeps the
binary content. The Go `internal/shared/objectstore` package provides the same
S3-compatible operations for future BFF-owned media workflows: put, delete,
bucket creation, and presigned reads.

Backups must therefore cover both the Strapi database and the object-storage
bucket. Backing up only Strapi PostgreSQL does not restore the uploaded files.

## imgproxy

imgproxy is an image transformation and delivery service. The Go helper in
`internal/shared/media` builds URLs such as resize/crop requests that point to
an `s3://<bucket>/<key>` source. imgproxy reads the object from MinIO/S3,
generates the requested variant in memory, and returns it to the client.

imgproxy does not own the original file, Strapi metadata, or a durable variants
table. `IMGPROXY_KEY` and `IMGPROXY_SALT` enable URL signing; leaving them empty
is intended only for local development.

## Volumes and non-database state

- `postgres_data` contains the Go application PostgreSQL cluster.
- `strapi_postgres_data` contains the Strapi PostgreSQL cluster.
- `minio_data` contains uploaded object data.
- `strapi_node_modules` and `go_modules` are dependency caches, not business
  data.

Do not put secrets in these stores or commit `.env` files. Use `.env.example`
as the configuration reference, and keep database credentials, Strapi tokens,
object-storage keys, and imgproxy signing values in deployment secrets.
