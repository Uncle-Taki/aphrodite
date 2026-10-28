# Test coverage

The coverage gate measures **combined Go statement coverage across all packages**.
It includes command entrypoints, domain/use-case tests, HTTP handlers, adapters,
and integration tests. It does not exclude generated Go code or untested packages.
It requires every statement to execute; a rounded `100.0%` is not sufficient.
This is not a claim of complete branch coverage or JavaScript coverage of Strapi.

Run the complete suite, including live PostgreSQL, Redis, and Strapi:

```bash
python3 scripts/test_coverage.py --integration --race
```

Requirements: Go 1.26, Python 3, `redis-server`, and Docker with Compose. The first
integration run builds the project's Strapi image and may download dependencies.

The runner creates a unique Compose project, a separate Go database and Strapi
database, a temporary Redis instance, and generated Strapi read-only/full-access
tokens. Ports bind only to localhost. It does not load the application's `.env`,
use its database credentials, or modify its running containers. Test containers,
volumes, Redis, and token files are removed afterward.

The live tests cover:

- Redis set/get/delete/ping and queue payload storage.
- Go startup, schema migration, health checks, registration, duplicate-user
  rejection, login, authenticated posts/comments, updates, and deletion against
  PostgreSQL through the HTTP API.
- Article creation/update/deletion through the Go API and real Strapi.
- Strapi published reads in `fa`, invalid-token rejection, full-access writes,
  read-only write rejection, and document lifecycle behavior.

For a faster run with Redis but without Docker services:

```bash
python3 scripts/test_coverage.py --race
```

This skips the two live PostgreSQL/Strapi tests. Ordinary `go test ./...` also
skips Redis integration tests unless `aphrodite_TEST_REDIS_ADDR` is set.

Reports default to `/tmp/aphrodite-coverage/coverage.out` and
`/tmp/aphrodite-coverage/coverage.html`. Choose another directory with `--output`.
The runner exits unsuccessfully if tests fail, a dependency fails to start, or any
Go statements remain uncovered.
