# Aphrodite

Aphrodite is a simple blog backend. It exposes a small, opinionated HTTP+JSON API so a front-end app (web, mobile, or SSR) can register users, publish posts, and thread comments.

- **Base URL (local dev):** `http://localhost:8080`
- **API version prefix:** `/v1`
- **Auth:** stateless JWT bearer token issued by `POST /v1/users/login`
- **Content type:** `application/json` on every request and response
- **Time format:** RFC 3339 UTC (e.g. `2026-07-03T14:22:11Z`)
- **IDs:** UUID v4 strings for every resource

Interactive API docs (Swagger UI) are served at [`/swagger/index.html`](http://localhost:8080/swagger/index.html) once the server is running.

---

## Table of contents

- [Getting started](#getting-started)
- [Authentication](#authentication)
- [Resource summary](#resource-summary)
- [Users](#users)
- [Posts](#posts)
- [Comments](#comments)
- [Pagination](#pagination)
- [Error format](#error-format)
- [Data model reference](#data-model-reference)
- [End-to-end example](#end-to-end-example)
- [Generating / updating Swagger docs](#generating--updating-swagger-docs)

---

## Getting started

The back-end is a single Go service backed by Postgres and Redis. If you're only integrating a front-end, ask your back-end teammate for the base URL — otherwise:

```bash
cp .env.example .env      # then fill in required values
docker compose -f build/docker-compose.yaml up --build
curl -s http://localhost:8080/healthz
```

If Postgres reports `password authentication failed` after changing `.env`, reset the local database volume and start again:

```bash
docker compose -f build/docker-compose.yaml down -v
docker compose -f build/docker-compose.yaml up --build
```

`/healthz` returns `200` when both Postgres and Redis are reachable, `503` otherwise. Use it as your readiness probe from the front-end deployment pipeline (not from the user's browser).

For hot-reload development with Air:

```bash
docker compose -f build/docker-compose.dev.yaml up --build
```

---

## Authentication

Aphrodite uses a **bearer token** model.

```
┌──────────────┐    POST /v1/users/register    ┌──────────────┐
│              │ ────────────────────────────► │              │
│  Front-end   │    POST /v1/users/login       │  Aphrodite   │
│              │ ◄──── { token, user } ─────── │              │
│              │                                │              │
│              │    GET /v1/users/me            │              │
│              │    Authorization: Bearer …     │              │
│              │ ────────────────────────────► │              │
└──────────────┘                                └──────────────┘
```

1. **Register** with `POST /v1/users/register`. Returns the created user (no token — you still have to log in).
2. **Login** with `POST /v1/users/login` using either username *or* email in `identifier`. Returns `{ token, user }`.
3. **Store the token** in memory or `httpOnly` cookie. Do not `localStorage` it in production.
4. **On every authenticated request**, send `Authorization: Bearer <token>`.
5. **Log out** by dropping the token client-side. Tokens are stateless; the server has nothing to invalidate.

> Treat the JWT as opaque from clients. Decode it only in trusted server-side code that owns the signing secret.

### Roles

Every user carries a `role`:

| Role | Can do |
|---|---|
| `user` | Register, login, view/update own profile, change password, create/list/read/update/delete own posts, comment, update/delete own comments |
| `admin` | Everything a `user` can, plus: view/update any profile, promote users to admin, update/delete any post, update/delete any comment |

Roles default to `user`. Public registration may create an `admin` only when the request includes the configured `SUPER_ADMIN_KEY` as `super_admin_key`; after bootstrap, existing admins can promote users through `PUT /v1/users/{id}`.

---

## Resource summary

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/v1/users/register` | — | Create a new user |
| POST | `/v1/users/login` | — | Exchange credentials for a bearer token |
| GET | `/v1/users/me` | Bearer | Current user's profile |
| PUT | `/v1/users/me` | Bearer | Update current user's profile |
| PUT | `/v1/users/me/password` | Bearer | Change current user's password |
| GET | `/v1/users` | Bearer (admin) | List users (paginated, newest first) |
| GET | `/v1/users/{id}` | Bearer (admin) | Any user's profile |
| PUT | `/v1/users/{id}` | Bearer (admin) | Update any user's profile |
| GET | `/v1/posts` | — | List posts (paginated, newest first) |
| GET | `/v1/posts/{id}` | — | Fetch one post |
| POST | `/v1/posts` | Bearer | Publish a post (caller = author) |
| PUT | `/v1/posts/{id}` | Bearer (author or admin) | Replace a post's title/content |
| DELETE | `/v1/posts/{id}` | Bearer (author or admin) | Delete a post |
| GET | `/v1/posts/{id}/comments` | — | List comments on a post (paginated, oldest first) |
| POST | `/v1/posts/{id}/comments` | Bearer | Add a comment to a post |
| PUT | `/v1/comments/{id}` | Bearer (author or admin) | Replace a comment's content |
| DELETE | `/v1/comments/{id}` | Bearer (author or admin) | Delete a comment |
| GET | `/healthz` | — | Health probe |

---

## Users

### Register — `POST /v1/users/register`

Public. Creates a new account.

```json
// Request
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "correct-horse-battery-staple",
  "phone_number": "+15551234567",
  "role": "user"
}
```

`phone_number` and `role` are optional. `role` defaults to `"user"`. To create a bootstrap admin, send `"role": "admin"` with `"super_admin_key": "<SUPER_ADMIN_KEY>"`; otherwise admin registration returns `403`.

**201 Created** returns the created user (no token — call `login` next).

```json
{
  "id": "b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f",
  "username": "alice",
  "email": "alice@example.com",
  "phone_number": "+15551234567",
  "role": "user",
  "created_at": "2026-07-03T14:22:11Z",
  "updated_at": "2026-07-03T14:22:11Z"
}
```

**Errors**

| Status | Meaning |
|---|---|
| 400 | Missing/invalid field (username blank, malformed email, invalid role) |
| 409 | Username or email already taken |

### Login — `POST /v1/users/login`

Public. Exchanges credentials for a bearer token.

```json
{
  "identifier": "alice",
  "password": "correct-horse-battery-staple"
}
```

`identifier` accepts username **or** email.

**200 OK**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJiM2Y2YTNhOC02ZGYwLTRlOWUtYTljMC1kNGExYzFlNThhOWYiLCJyb2xlIjoidXNlciIsImlhdCI6MTc4MzA4ODUzMSwiZXhwIjoxNzgzMTc0OTMxfQ.FJnOGxSQqEsQF9CWIFxjrTC1vlhqIpxHFyDFEoJM1oQ",
  "user": {
    "id": "b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f",
    "username": "alice",
    "email": "alice@example.com",
    "phone_number": "+15551234567",
    "role": "user",
    "created_at": "2026-07-03T14:22:11Z",
    "updated_at": "2026-07-03T14:22:11Z"
  }
}
```

Use `Authorization: Bearer <token>` on every subsequent authenticated request.

**Errors**

| Status | Meaning |
|---|---|
| 400 | Missing identifier or password |
| 401 | Invalid credentials (**do not** distinguish "no such user" vs "wrong password" in your UI) |

### Get current user — `GET /v1/users/me`

Authenticated. Returns the caller's own profile — use this to hydrate the app after a page reload if you cached the token but not the user object.

**200 OK** → `UserResponse` (same shape as register response).

### Update current user — `PUT /v1/users/me`

Authenticated. Updates the caller's `username`, `email`, and optional `phone_number`; role cannot be changed through this endpoint.

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "phone_number": "+15551234567"
}
```

**200 OK** → `UserResponse`.

### Change password — `PUT /v1/users/me/password`

Authenticated. Verifies `current_password` before writing the new password hash.

```json
{
  "current_password": "correct-horse-battery-staple",
  "new_password": "new-correct-horse-battery-staple"
}
```

**204 No Content** on success.

### List users — `GET /v1/users`

**Admin only.** Newest first. Paginated.

```
GET /v1/users?limit=20&page=1
```

| Query param | Type | Default | Max | Notes |
|---|---|---|---|---|
| `limit` | int | `USER_DEFAULT_LIMIT` | `USER_MAX_LIMIT` | Values ≤ 0 fall back to default |
| `page` | int | 1 | — | Values ≤ 0 fall back to page 1 |

**200 OK**

```json
{
  "users": [
    {
      "id": "b3f6…",
      "username": "alice",
      "email": "alice@example.com",
      "role": "user",
      "created_at": "2026-07-03T14:22:11Z",
      "updated_at": "2026-07-03T14:22:11Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "page": 1
}
```

### Get user by ID — `GET /v1/users/{id}`

**Admin only.** Look up any user by UUID.

**Errors**

| Status | Meaning |
|---|---|
| 401 | Missing / invalid token |
| 403 | Caller is not `admin` |
| 404 | No such user |

### Update user by ID — `PUT /v1/users/{id}`

**Admin only.** Updates any user's `username`, `email`, optional `phone_number`, and optional `role`.

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "phone_number": "+15551234567",
  "role": "admin"
}
```

**200 OK** → `UserResponse`.

---

## Posts

### List posts — `GET /v1/posts`

Public. Newest first. Paginated.

```
GET /v1/posts?limit=20&page=1
```

| Query param | Type | Default | Max | Notes |
|---|---|---|---|---|
| `limit` | int | `POST_DEFAULT_LIMIT` | `POST_MAX_LIMIT` | Values ≤ 0 fall back to default |
| `page` | int | 1 | — | Values ≤ 0 fall back to page 1 |

**200 OK**

```json
{
  "posts": [
    {
      "id": "38f2…",
      "author_id": "b3f6…",
      "title": "Hello, world",
      "content": "This is my first post.",
      "created_at": "2026-07-03T14:25:03Z",
      "updated_at": "2026-07-03T14:25:03Z"
    }
  ],
  "total": 137,
  "limit": 20,
  "page": 1
}
```

`total` is the full count matching the query — use it to render "Page 2 of N" UI. The server always echoes the `limit` and `page` it actually applied after clamping.

### Get one post — `GET /v1/posts/{id}`

Public. Returns a single `PostResponse` or `404`.

### Create a post — `POST /v1/posts`

Authenticated. The caller becomes the author.

```json
{
  "title": "Hello, world",
  "content": "This is my first post."
}
```

Constraints: title 1–`POST_TITLE_MAX_LENGTH` chars, content 1–`POST_CONTENT_MAX_LENGTH` chars, both trimmed of leading/trailing whitespace.

**201 Created** → `PostResponse`.

### Update a post — `PUT /v1/posts/{id}`

Authenticated. **Only the original author or an admin** may update. Replaces both `title` and `content`.

```json
{
  "title": "Updated title",
  "content": "Updated post content."
}
```

Uses the same validation as create. **200 OK** → `PostResponse`.

### Delete a post — `DELETE /v1/posts/{id}`

Authenticated. **Only the original author or an admin** may delete.

**204 No Content** on success.

**Errors**

| Status | Meaning |
|---|---|
| 401 | Missing / invalid token |
| 403 | Caller is neither author nor admin |
| 404 | No such post |

---

## Comments

Comments are nested under posts for listing and creation, and flat for deletion.

### List comments on a post — `GET /v1/posts/{id}/comments`

Public. **Oldest first** (chronological reading order).

| Query param | Type | Default | Max |
|---|---|---|---|
| `limit` | int | `COMMENT_DEFAULT_LIMIT` | `COMMENT_MAX_LIMIT` |
| `page` | int | 1 | — |

**200 OK**

```json
{
  "comments": [
    {
      "id": "a01b…",
      "post_id": "38f2…",
      "author_id": "b3f6…",
      "content": "Great post!",
      "created_at": "2026-07-03T14:26:00Z",
      "updated_at": "2026-07-03T14:26:00Z"
    }
  ],
  "total": 4,
  "limit": 50,
  "page": 1
}
```

### Add a comment — `POST /v1/posts/{id}/comments`

Authenticated. The caller becomes the author.

```json
{ "content": "Great post!" }
```

Constraint: content 1–`COMMENT_CONTENT_MAX_LENGTH` chars, trimmed.

**201 Created** → `CommentResponse`.

### Update a comment — `PUT /v1/comments/{id}`

Authenticated. **Only the author or an admin** may update. Replaces `content`.

```json
{ "content": "Updated comment." }
```

Uses the same validation as add. **200 OK** → `CommentResponse`.

### Delete a comment — `DELETE /v1/comments/{id}`

Authenticated. **Only the author or an admin** may delete. **204 No Content** on success. Same 401 / 403 / 404 semantics as post deletion.

---

## Pagination

Every list endpoint uses the same shape:

- **Request:** `?limit=<int>&page=<int>` query params, both optional.
- **Response:** an object with an items array (`users`, `posts`, or `comments`), `total` (full match count), `limit`, and `page` (echo of what the server actually applied after clamping).

Client-side "next page" is just `page += 1`. There's no cursor pagination yet.

---

## Error format

Every 4xx and 5xx response has the shape:

```json
{ "error": "human-readable message" }
```

Status codes are consistent across all resources:

| Code | Meaning |
|---|---|
| 400 | Bad request — malformed body, invalid UUID, failed validation |
| 401 | Missing or invalid `Authorization` bearer token, or wrong credentials |
| 403 | Authenticated, but not permitted (e.g. deleting someone else's post as a non-admin) |
| 404 | Resource does not exist |
| 409 | Uniqueness conflict (user registration or profile update) |
| 500 | Unexpected server error — safe to surface a generic "something went wrong" toast |

**A note on 401 vs 403:** the front-end should clear the stored token on **401** (it's stale or bogus), but keep it on **403** (the user is logged in, just not authorized for that action). A 403 should surface as a permission error, not a logout.

---

## Data model reference

The Swagger schema at `/swagger/index.html` is generated from the same Go structs, so the field lists there are authoritative. For quick reference:

### `UserResponse`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | |
| `username` | string | Unique |
| `email` | string | Unique, lowercased server-side |
| `phone_number` | string \| null | Optional |
| `role` | `"user"` \| `"admin"` | |
| `created_at` | RFC 3339 timestamp | |
| `updated_at` | RFC 3339 timestamp | |

### `RegisterRequest`

| Field | Type | Notes |
|---|---|---|
| `username` | string | Required |
| `email` | string | Required |
| `password` | string | Required |
| `phone_number` | string \| null | Optional |
| `role` | `"user"` \| `"admin"` | Optional, defaults to `"user"` |
| `super_admin_key` | string | Required only when registering an admin |

### `PostResponse`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | |
| `author_id` | UUID | Points to `UserResponse.id` |
| `title` | string | 1–`POST_TITLE_MAX_LENGTH` chars |
| `content` | string | 1–`POST_CONTENT_MAX_LENGTH` chars |
| `created_at` | RFC 3339 timestamp | |
| `updated_at` | RFC 3339 timestamp | |

### `CommentResponse`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | |
| `post_id` | UUID | Points to `PostResponse.id` |
| `author_id` | UUID | Points to `UserResponse.id` |
| `content` | string | 1–`COMMENT_CONTENT_MAX_LENGTH` chars |
| `created_at` | RFC 3339 timestamp | |
| `updated_at` | RFC 3339 timestamp | |

---

## End-to-end example

Using `curl` — translate to your HTTP client of choice.

```bash
BASE=http://localhost:8080

# 1. Register
curl -s -X POST "$BASE/v1/users/register" \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com","password":"hunter22"}'

# 2. Login → save the token
TOKEN=$(curl -s -X POST "$BASE/v1/users/login" \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"alice","password":"hunter22"}' \
  | jq -r .token)

# 3. Who am I?
curl -s "$BASE/v1/users/me" -H "Authorization: Bearer $TOKEN"

# 4. Publish a post → capture its ID
POST_ID=$(curl -s -X POST "$BASE/v1/posts" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Hello","content":"first post"}' \
  | jq -r .id)

# 5. Comment on it
curl -s -X POST "$BASE/v1/posts/$POST_ID/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"content":"nice"}'

# 6. Read the thread
curl -s "$BASE/v1/posts/$POST_ID"
curl -s "$BASE/v1/posts/$POST_ID/comments"

# 7. Clean up
curl -s -X DELETE "$BASE/v1/posts/$POST_ID" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Generating / updating Swagger docs

Handlers carry [swaggo](https://github.com/swaggo/swag) annotations. To regenerate the JSON/YAML spec and the `docs/docs.go` file consumed by Swagger UI:

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.4

# -g is resolved relative to the FIRST -d entry, so pass it as a bare filename.
swag init -g main.go -d ./cmd/api,./internal -o ./docs
```

You'll see a benign warning `no Go files in .../internal` — swag still recurses into the sub-packages that hold the annotations.

After regenerating, the updated docs are served automatically at [`/swagger/index.html`](http://localhost:8080/swagger/index.html).
