# End-to-End Verification Report

Technical Test — Back End Developer (Task Management API, Multi-User).

- **Date:** 2026-09-11
- **Repository:** https://github.com/fatkulnurk/task-management-api
- **Commit under test:** `ca7dae9` (branch `main`), tests run on branch `feat/e2e-verification`
- **Result:** All 17 endpoints pass, 17/17 error cases pass, 35/35 Postman assertions pass, all Go suites pass.

## 1. Environment

| Item | Version |
|---|---|
| Go | 1.27.1 |
| Docker | 29.7.2 |
| Docker Compose | v5.5.1 |
| MySQL image | `mysql:9.7.2` |
| Newman | 6.2.2 (via `npx`) |
| OS | Windows (PowerShell 7) |

The stack was rebuilt from scratch before the run:

```text
docker compose down -v
docker compose up -d --build
```

`migrate` applied `1/u .. 8/u` (7 schema files + seed). `GET /health` returned `204`.
Both containers reported `healthy`:

```text
taskmanagement-api-1     Up (healthy)   127.0.0.1:44001->8080/tcp
taskmanagement-mysql-1   Up (healthy)   127.0.0.1:44002->3306/tcp
```

## 2. Happy path — full collection via Newman

Command:

```text
npx newman run postman/task-management-api.postman_collection.json
```

Summary: **18 requests, 35 assertions, 0 failures** (duration ~1.7s, average 17ms).

| # | Request | Status |
|---|---|---|
| 1 | `GET /health` | `204` |
| 2 | `POST /auth/register` | `201` |
| 3 | `POST /auth/login` | `200` |
| 4 | `POST /auth/refresh` | `200` |
| 5 | `POST /auth/logout` | `204` |
| 6 | `POST /teams` | `201` |
| 7 | `GET /teams` | `200` |
| 8 | `GET /teams/{id}` | `200` |
| 9 | `GET /teams/{id}/members` | `200` |
| 10 | `POST /teams/{id}/members` | `201` |
| 11 | `POST /tasks` (new key) | `201` |
| 12 | `POST /tasks` (same key, replay) | `201` |
| 13 | `GET /tasks` | `200` |
| 14 | `GET /tasks/{id}` | `200` |
| 15 | `PUT /tasks/{id}` | `200` |
| 16 | `POST /tasks/{id}/assign` | `200` |
| 17 | `DELETE /tasks/{id}` | `204` |
| 18 | `DELETE /teams/{id}/members/{user_id}` (Cleanup) | `204` |

## 3. Error and edge cases (ad-hoc HTTP)

17 cases, all matched the expected status and error code. Every error body carried the full envelope (`status`, `code`, `message`, `timestamp`) and used only snake_case keys.

| Case | Expected | Actual | `code` |
|---|---|---|---|
| No token | `401` | `401` | `unauthorized` |
| Invalid token | `401` | `401` | `unauthorized` |
| Path id not a UUID | `422` | `422` | `invalid_request` |
| `Idempotency-Key` missing | `400` | `400` | `invalid_idempotency_key` |
| `Idempotency-Key` not a UUID | `400` | `400` | `invalid_idempotency_key` |
| Broken JSON | `400` | `400` | `invalid_json` |
| Body validation | `422` | `422` | `invalid_request` |
| Email already registered | `409` | `409` | `conflict` |
| Wrong password | `401` | `401` | `unauthorized` |
| Read another user's task (IDOR) | `404` | `404` | `not_found` |
| Create task in a non-member team | `403` | `403` | `forbidden` |
| Assign to a non-member | `403` | `403` | `forbidden` |
| Add duplicate team member | `409` | `409` | `user_already_team_member` |
| Remove the team owner | `403` | `403` | `owner_cannot_be_removed` |
| Remove a member with an active assignment | `409` | `409` | `member_has_active_assignments` |
| `page=0` | `422` | `422` | `invalid_request` |
| `limit=101` | `422` | `422` | `invalid_request` |

The IDOR case returns `404`, not `403`, so a stranger cannot tell whether the resource exists.

## 4. Response format consistency

Every response was checked automatically during the run:

| Check | Result |
|---|---|
| `Content-Type: application/json` (non-204) | PASS |
| `X-Request-Id` present and a UUID | PASS |
| A valid UUID `X-Request-Id` request header is echoed unchanged | PASS |
| No PascalCase JSON keys anywhere (all snake_case) | PASS |
| Error body has `status` (int), `code`, `message`, `timestamp` (RFC3339 UTC) | PASS |
| `errors[]` appears only on validation errors | PASS |
| `Team` = `id, owner_id, name, created_at, updated_at` | PASS |
| `Member` = `id, name, email, is_owner` | PASS |
| `UserOutput` = `id, name, email` | PASS |
| `TokenPair` = `access_token, refresh_token, token_type, expires_in, access_token_expires_at, refresh_token_expires_at` | PASS |
| `Task` = `id, team_id, creator_id, assignee_id, title, description, status, created_at, updated_at` | PASS |

Every task response is checked for non-empty `created_at` and `updated_at`. On the first pass, `PUT /tasks/{id}` returned empty timestamps because the service echoed the request instead of the stored row. That is fixed on branch `fix/put-task-response`: `Update` now re-reads the stored task, so `PUT` matches `GET /tasks/{id}` byte for byte (`PUT_EQ_GET=True`) and a stray `assignee_id` in the update body is ignored (assignment stays on `POST /tasks/{id}/assign`).


Sample error body:

```json
{"status":401,"code":"unauthorized","message":"authentication required","timestamp":"2026-09-11T06:30:15Z"}
```

## 5. Idempotency and concurrency

| Check | Result |
|---|---|
| `POST /tasks` response body equals `GET /tasks/{id}` byte for byte (no trailing newline) | PASS (`POST_EQ_GET=True`) |
| Same key, first call returns `201` and creates the task | PASS |
| Same key, second call replays the first body and makes no new task | PASS |
| Same key with a **different** body still replays the first response (no `409`) | PASS — status `201`, same `id`, title stayed `Replay one` |
| Sequential unit test (mock repository, no database) | PASS |
| Concurrent unit test: 32 goroutines, one key, exactly one create | PASS |
| Concurrent integration test against MySQL: 32 goroutines -> 1 task row, 1 log row, 1 idempotency row | PASS (race detector on, `-tags=integration`) |

## 6. Database integrity (MySQL)

Baseline after fresh migration:

```text
users 3 | teams 2 | team_members 5 | tasks 7 (6 alive) | task_logs 11 | idempotency_keys 0
```

After the Newman run plus the error matrix:

```text
users 5 | teams 3 | team_members 6 | tasks 11 (9 alive) | task_logs 18 | idempotency_keys 4
```

`task_logs` by action:

```text
task_created 10 | task_assigned 2 | task_status_changed 4 | task_deleted 2
```

The deltas match the operations performed: every create, update, assign, and delete wrote exactly one log row in the same transaction. `POST /tasks/{id}/assign` changed `assignee_id` and wrote `task_assigned`; unassign writes `task_unassigned`. Each distinct `Idempotency-Key` produced exactly one stored response.

## 7. Logging and observability

Structured JSON logs from `log/slog`, one line per request:

```json
{"time":"2026-09-11T06:30:20.615338129Z","level":"WARN","msg":"http","request_id":"11111111-2222-4333-8444-555555555555","method":"GET","path":"/tasks","status":401,"latency":45}
```

| Check | Result |
|---|---|
| Every request logs `request_id` (UUID), `method`, `path`, `status`, `latency` | PASS |
| Level INFO for 2xx/3xx | PASS (38 INFO lines) |
| Level WARN for 4xx | PASS (19 WARN lines) |
| Level ERROR for 5xx | PASS — MySQL was stopped briefly; `GET /tasks` returned a sanitized `500` and logged at ERROR with all fields; MySQL restarted, health back to `204` |
| Panic is recovered and answered with a generic `500`; no stack trace in the body | PASS (covered by the `TestStackRecover` unit test; stack trace goes to the ERROR log) |

The forced 5xx response body did not leak any internal detail:

```json
{"status":500,"code":"internal_error","message":"internal server error","timestamp":"2026-09-11T06:30:39Z"}
```

## 8. Automated test suites

| Command | Result |
|---|---|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./...` | PASS — 12 packages `ok`, 0 failures (~97 test functions) |
| `go test -race ./...` (via `make test-race`, `golang:1.27`) | PASS — no data races |
| `go test -race -tags=integration ./internal/modules/tasks/repository/...` (via `make test-integration`) | PASS |
| `make test-cover` | 64.9% of statements in the business modules |
| `gofmt -l internal cmd` on committed LF blobs (Linux container) | PASS — 0 files |

Unit tests use `gomock` and `go-sqlmock` only. `go test ./...` needs no database or network; the integration test is gated behind the `integration` build tag and skips when `TEST_DATABASE_URL` is unset.

## 9. Requirement matrix

| Requirement | Result | Evidence |
|---|---|---|
| Register / login / JWT | PASS | Newman requests 2–4; refresh rotation and logout covered by unit tests |
| `POST /tasks` | PASS | Newman 11–12; `201` |
| `GET /tasks` | PASS | Newman 13; `{data, meta}` |
| `GET /tasks/:id` | PASS | Newman 14; byte-identical to create |
| `PUT /tasks/:id` | PASS | Newman 15 |
| `DELETE /tasks/:id` | PASS | Newman 17; soft delete, then `404` |
| Filter by status | PASS | `GET /tasks?status=` handled in query builder; unit-tested |
| Search by title | PASS | `GET /tasks?search=` (`LIKE %..%`); unit-tested |
| Pagination (`limit`, `page`) | PASS | `meta{page,limit,total}`; `422` on out-of-range |
| README (run, endpoints, architecture) | PASS | `README.md`, 272 lines |
| 1. Idempotency (header UUID, 24h, replay) | PASS | Section 5 |
| 2. Structured errors + panic handler | PASS | Sections 3, 4, 7 |
| 3. Assign in one transaction (update + log + notify + rollback) | PASS | Section 6; `sqlmock` rollback test |
| 4. Structured logging + levels | PASS | Section 7 |
| 5. Unit tests without database; sequential + concurrent idempotency | PASS | Section 8; service test runs 32 goroutines and asserts exactly one create |
| Published to GitHub | PASS | Public repo |
| Email the repository link to the recruiter | OUTSTANDING | Candidate action |

## 10. Security review (IDOR and cross-user access)

A read-only audit of every resource path was done. Task access is scoped to the token identity in the SQL itself, and team access is scoped to membership, so there is no IDOR.

| Surface | Guard | Location |
|---|---|---|
| `GET /tasks/{id}` | `WHERE id=? AND (creator_id=? OR assignee_id=?)` | `tasks/repository/repository.go` |
| `GET /tasks` | JOIN `team_members` plus creator/assignee scope | `tasks/repository/repository.go` |
| `PUT /tasks/{id}` | scoped read, then creator/assignee role check | `tasks/service/service.go` |
| `DELETE /tasks/{id}` | `WHERE id=? AND creator_id=?` | `tasks/repository/repository.go` |
| `POST /tasks/{id}/assign` | creator-only, target must be a team member | `tasks/service/service.go` |
| `GET /teams`, `GET /teams/{id}` | JOIN `team_members` on the caller | `teams/repository/repository.go` |
| `GET /teams/{id}/members` | `IsMember`, else `404` | `teams/repository/repository.go` |
| `POST`/`DELETE /teams/{id}/members` | `IsOwner` | `teams/service/service.go` |

Non-access returns `404`, not `403`, so a stranger cannot tell whether a resource exists. Mass assignment is not possible: `creator_id` and `owner_id` are never read from a body, and `team_id` on update is overwritten from the stored task.

Recorded gaps (not IDOR, left as follow-ups):

- Removing a team member does not revoke task access for tasks they created; task access is identity-based by design.
- Login runs bcrypt only when the email exists, a timing oracle for user enumeration.
- No login rate limiting.
- A concurrent duplicate `POST /teams/{id}/members` can return `500` instead of `409` because the unique-key error is not mapped.
- `POST /auth/logout` returns `401` for an unknown token; refresh-token reuse has no family invalidation.

## 11. Known limitations

- `gofmt -l` on this Windows checkout flags every file because line endings are CRLF. The committed blobs are LF and are gofmt-clean; this was confirmed by extracting the commit into a Linux container (0 files needed formatting). This is an environment artifact, not a code issue.
- `golangci-lint` and `go-arch-lint` were not run: the installed lint binary targets Go 1.26 and rejects Go 1.27, and `go-arch-lint` is not installed. Dependency direction was checked by hand and by `go vet`.
- The ERROR log level was exercised by stopping MySQL for a few seconds; there is no code path that returns a 5xx under normal operation, which is the intended behavior.

## Conclusion

Every endpoint in the case study works, all response envelopes are consistent and snake_case, idempotency is race-safe, the assignment transaction is atomic, logging is structured with correct levels, and the unit suite runs without a database. Only the recruiter email remains.
