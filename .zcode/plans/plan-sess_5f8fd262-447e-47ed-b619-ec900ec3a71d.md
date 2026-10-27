Raise Aphrodite test coverage from 47% to >80%

# Current state and target math

Project-wide coverage today: **47.1%** (`go test ./... -coverpkg=./...`). Target: **>80%**.

Coverage is computed over ALL packages, so every statement counts. The untested mass falls into five buckets, ordered by value:

1. **Pure domain/model code** (~500 lines, zero deps): user/post/comment domain, editorial/domain, shared/media (imgproxy signing), shared/httpx/authctx, user/infra/crypto (bcrypt) — table-driven goldmine.
2. **Usecase gaps**: user 56.9% (get_profile, authenticate happy/error paths, session Authenticate/Logout, change_password branches), post & comment 19.7% (only update_test.go exists — create/delete/get/list entirely untested; the in-package fakes already implement the full ports).
3. **Transport/http x3** (user ~653, post ~295, comment ~285 lines, all 0%): handlers take concrete `*usecase.Xxx`, but usecase constructors accept interfaces — build real usecases over the existing in-memory repo fakes, then drive gin routers with httptest, copying the harness from `internal/content/transport/http/handler_test.go`.
4. **infra/postgres x3** (~540 lines, 0%): repos take concrete `*gorm.DB`, but the sqlite driver is already a direct dependency with three existing `sqlite.Open(":memory:")` harnesses to copy — run real CRUD/pagination/not-found tests against in-memory sqlite.
5. **Hard leftovers** (~170 lines): shared/jobs (asynq+redis), shared/objectstore network methods, cmd/worker main (no seam), cmd/api main/run/defaultRunDeps wrappers, plus 2 sqlite-unreachable unique-violation branches in the user repo.

Getting buckets 1–4 fully tested projects to roughly 88–92% overall, comfortably above 80% even ignoring bucket 5. No new dependencies, no testcontainers, no mocks libraries — stdlib testing + httptest + in-memory sqlite, matching repo conventions.

# Test strategy per package (following aphrodite-go-testing)

All tests are focused table-driven or scenario tests next to the code, in-memory fakes for use cases, boundary/error/authorization coverage. No changes to production code except two small seam extractions (below).

## Phase 1 — Pure domain & utility packages (biggest cheap win)

- `internal/user/domain` (`user_test.go`, `session_test.go`): NewUser validation table (empty username/email/role, invalid role, phone trim), Update, ChangePassword, Role.Valid, looksLikeEmail, HashSessionToken.
- `internal/post/domain` (`post_test.go`): NewPost (nil author, title/content trim + max length, empty), Update, normalizeContent branches.
- `internal/comment/domain` (`comment_test.go`): NewComment (nil post/author, content rules), Update.
- `internal/editorial/domain` (`model_test.go`): Post.Validate (slug/language/blocks, JSON payload validity, duplicate positions) + CanTransition full state machine as a transition table.
- `internal/shared/media` (`imgproxy_test.go`): URL signing (known key/salt → expected signature), unsigned mode, invalid hex fallback in decodeHex, non-positive width, itoa.
- `internal/shared/httpx/authctx` (`caller_test.go`): Set/From round-trip, missing-key, wrong-type.
- `internal/user/infra/crypto` (`bcrypt_test.go`): NewBcryptHasher cost clamping, Hash/Verify round-trip, wrong password → ErrInvalidCredential (use bcrypt.MinCost for speed).

## Phase 2 — Usecase gap-filling (fakes already exist in-package)

- `internal/user/usecase`:
  - `get_profile_test.go`: GetUserProfile happy + not-found.
  - `authenticate_test.go` (extend): happy path, empty input, unknown identifier (username and email lookup), disabled user, wrong password, repo error, token issue failure.
  - `session_test.go` (extend): CMSSessionManager.Authenticate (empty token, expired/revoked, disabled user, non-editor role, touch), Logout (revoke, nil session), secureToken.
  - `change_password` cases in update_test.go (extend): empty password, user not found, hash failure, update failure.
- `internal/post/usecase`: new `create_test.go`, `delete_test.go`, `get_test.go`, `list_test.go` — success, validation limits (title/content max), authorization (author vs other user vs admin), not-found, list default/max clamping and pagination. Reuse `updatePostRepo` fake (rename to a shared `postRepo` fake in a `fakes_test.go` if needed).
- `internal/comment/usecase`: same treatment — `add_test.go`, `delete_test.go`, `list_test.go` reusing `updateCommentRepo`.

## Phase 3 — Transport/http handler tests x3 (copy content harness pattern)

For each of user, post, comment transport packages, one `handler_test.go` (+ `middleware_test.go` for user):
- Construct real usecases over in-memory repo fakes (from Phase 2), or thin local fakes where a usecase is awkward — handlers take concrete `*usecase.Xxx` pointers, and all usecase constructors accept interface ports, so real construction works without a DB.
- Drive `gin.SetMode(gin.TestMode)` routers via httptest; cover every handler happy path + every `mapXxxError` branch + authctx-missing 401s + invalid UUID 400s + JSON bind failures.
- user transport additionally: `middleware.go` (AuthMiddleware with fake TokenIssuer: valid/invalid/expired/malformed Bearer, RequireAdmin) and `cms.go` (RegisterCMS wiring, CMSAuthMiddlewareSecure cookie flows: missing cookie, invalid session, disabled user; RequireCMSRole allowed/denied; Login sets cookie; Logout revokes; Me).
- Include route-wiring assertions (Register/ routes respond 405 vs 404 as appropriate) where cheap.

## Phase 4 — infra/postgres repo tests x3 (sqlite in-memory)

Copy the `sqlite.Open(":memory:")` + AutoMigrate pattern from `internal/shared/postgres/transaction_test.go`:
- `internal/user/infra/postgres/repository_test.go`: Create + FindByID/FindByUsername/FindByEmail round-trips, findOne not-found mapping, List pagination + ordering, Update (success, RowsAffected=0 → ErrNotFound), unique-violation mapping branch tested via the existing `IsUniqueViolation` helper only (the PgError branch itself is postgres-only; document the gap in a comment, don't chase it).
- `session_test.go`: session Create/FindActiveByHash (active, revoked, expired), Touch, Revoke.
- `internal/post/infra/postgres/repository_test.go`: CRUD + pagination + not-found.
- `internal/comment/infra/postgres/repository_test.go`: CRUD + ListByPost (ASC ordering, pagination, Count) — create parent user/post rows first to satisfy FKs.
- `row_test.go` mapping tests where mappers have branches (fromDomain/toDomain).

## Phase 5 — cmd/api gap-filling + small seams

- Extend `cmd/api/main_test.go`: a scenario with `config.C.Strapi.BaseURL` set + stubbed `waitForStrapi` to execute the content-handler wiring block (lines ~141–159); redis-connect failure and JWT-setup failure paths already injectable via runDeps.
- `internal/user/infra/token`: already 77.6% — leave.

## Explicitly OUT of scope (documented, not chased)

- `internal/shared/jobs`, `internal/shared/objectstore` network paths, `cmd/worker` main: require redis/minio or a production-code seam refactor with no consumer benefit; jobs.Queue has no production consumer, cmd/worker calls Run(nil). ~170 statements total; skipping still lands ~88%.
- `cmd/api` `main`/`run`/`defaultRunDeps` wrappers.
- The two PgError-only branches in the user repo (sqlite can't produce 23505).

# Production code changes

Only test-enabling seams if a handler test genuinely can't construct a dependency — expected: none, since all usecase constructors take interfaces. If one arises during implementation (e.g. CMSSessionManager's `now`/token hook), prefer unexported constructor hooks already present (session_test.go overrides these) over signature changes.

# Verification

Per phase: `go test ./<pkg> -cover -count=1` targeting the package, then:
1. `go test ./... -coverpkg=./... -coverprofile=cover.out` → `go tool cover -func=cover.out | tail -1` must report **>80%**.
2. `gofmt -l` clean on all new files; `go vet ./...` clean.
3. `.agents/.plugin/aphrodite-engineering/scripts/check_architecture.py` still green (tests must not break dependency direction — fakes live in `_test.go` packages).
4. `go build ./cmd/api ./cmd/cli` still OK.

If the total lands between 78–80% after phases 1–4, the fallback lever is cmd/api strapi-wait coverage and the token package's remaining gaps — cheaper than touching the excluded packages.

# Execution order

Phases 1→2→3→4→5 (domain first: fastest, and domain tests anchor the usecase tests' expectations). Each phase leaves the suite green before the next starts. Estimated ~20 new/extended test files.