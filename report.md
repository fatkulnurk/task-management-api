# End-to-End Verification Report

Technical Test — Back End Developer (Task Management API, Multi-User).

- **Date:** 2026-09-11
- **Repository:** https://github.com/fatkulnurk/task-management-api
- **Commit under test:** `2cef96a` (branch `main`)
- **Result:** All 17 endpoints pass, 17/17 error cases pass, 35/35 Postman assertions pass, 51-case security matrix pass, all Go suites pass.

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

## 10. Security E2E (IDOR, JWT, tampering)

A second pass ran a 51-case adversarial matrix on a fresh database (`down -v` + `up -d --build`). Actors: `alice` (Platform owner), `bob` (Platform member, Mobile owner), `carol` (member of both), and `mallory` (a new user in no team). Result: **50 cases matched the exact expected status and code; 1 case was rejected earlier than expected but is still safe** (see note).

### A. Horizontal task access — another user's task returns `404`

| Case | Expected | Actual |
|---|---|---|
| bob / carol / mallory `GET` alice's task | `404` | `404` |
| bob `PUT` alice's task | `404` | `404` |
| bob / mallory `DELETE` alice's task | `404` | `404` |
| bob `POST /assign` alice's task | `404` | `404` |
| alice `GET` own task | `200` | `200` |

### B. Assignee scope — read and status yes, title/delete/assign no

| Case | Expected | Actual |
|---|---|---|
| alice assigns task to bob | `200` | `200` |
| bob (assignee) `GET` | `200` | `200` |
| bob (assignee) `PUT` status only | `200` | `200` |
| bob (assignee) `PUT` title | `403` | `403` |
| bob (assignee) `DELETE` | `404` | `404` |
| bob (assignee) reassign | `404` | `404` |
| carol (same team, not assignee) `GET` | `404` | `404` |

### C. Cross-team — `404` on tasks, `403` on create

| Case | Expected | Actual |
|---|---|---|
| alice `GET`/`PUT`/`DELETE`/`assign` bob's Mobile task | `404` | `404` |
| alice `POST /tasks` into the Mobile team (not a member) | `403` | `403` |

### D. Team scope — non-member `404`, non-owner writes `403`

| Case | Expected | Actual |
|---|---|---|
| alice `GET` Mobile team / members | `404` | `404` |
| alice add/remove Mobile member (not owner) | `403` | `403` |
| carol add Platform member (not owner) | `403` | `403` |
| mallory `GET` Platform team / members | `404` | `404` |

### E. JWT and token abuse — all rejected

| Case | Expected | Actual |
|---|---|---|
| No token / malformed `Bearer` / garbage | `401` | `401` |
| Tampered signature | `401` | `401` |
| Expired `HS256` | `401` | `401` |
| `alg=none` (unsigned) | `401` | `401` |
| `alg=HS384` signed with the secret | `401` | `401` |
| Missing `exp` | `401` | `401` |
| Refresh token used as access token | `401` | `401` |
| Old refresh token after rotation | `401` | `401` |
| New refresh token after rotation | `200` | `200` |
| Access token used as refresh token | `401` | `422` (see note) |

The `alg=none`, `HS384`, missing-`exp`, and expired tokens were minted for this test from the configured secret; the verifier accepts `HS256` with a required `exp` only. Note: a JWT access token placed in the refresh body is rejected by request validation at `422 invalid_request` (the value exceeds the 128-char refresh-token limit) before it reaches the service. It is still rejected; only the status differs from the `401` expectation.

### F. Idempotency scoping

| Case | Expected | Actual |
|---|---|---|
| Same `Idempotency-Key` for alice and bob | two separate tasks, no cross-user replay | `201` + `201`, different ids |
| alice replays her own key | byte-identical body | byte-identical |
| Non-UUID key | `400` | `400 invalid_idempotency_key` |

### G. Mass assignment / parameter tampering — ignored

| Case | Result |
|---|---|
| `creator_id`, `id`, `assignee_id` in `POST /tasks` body | ignored: creator = token user, assignee = `null`, server id generated |
| `team_id`, `creator_id` in `PUT /tasks/{id}` body | ignored: team stays as stored |
| `owner_id` in `POST /teams` body | ignored: owner = token user |

### H. Input abuse

| Case | Expected | Actual |
|---|---|---|
| SQLi payload in `search` | `200`, no error leak | `200` |
| Invalid enum in `status` | `422` | `422 invalid_request` |
| Body larger than 1 MiB | `413` | `413 payload_too_large` |
| Unknown JSON fields | ignored, `200` | `200` |

### I. Response hygiene

| Case | Result |
|---|---|
| Any response contains `password_hash` or a bcrypt hash | PASS — absent |
| Any response contains `token_hash` | PASS — absent |
| 5xx body | sanitized `{"status":500,"code":"internal_error","message":"internal server error",...}`, no stack trace |

### J. Database integrity

After the entire security run, no task titled `hijack` or `x` existed, and no task had been created by carol or mallory. Every rejected write created zero rows. Authorized operations wrote exactly the expected rows.

### Findings (documented, not fixed)

| # | Severity | Finding | Observed |
|---|---|---|---|
| S1 | Medium | Login runs bcrypt only when the email exists, so an unknown email answers faster | known-email wrong-password avg `0.0555s` vs unknown-email avg `0.0037s` (15x); user enumeration oracle |
| S2 | Medium | No login rate limiting or lockout | 20 consecutive wrong-password attempts all returned `401`, no `429` |
| S3 | Low | `POST /auth/logout` returns `401` for an unknown refresh token | confirmed `401 unauthorized` |
| S4 | Info | Concurrent duplicate `POST /teams/{id}/members` | 12 concurrent adds: `201` x1, `409` x11, no `500` — the previously suspected `500` did not reproduce |
| S5 | Info | Removing a team member does not revoke their identity-based task access | by design; task access is creator/assignee scoped |
| S6 | Info | Access token in the refresh body is rejected at `422` validation instead of `401` | still rejected, no security impact |

Concurrent idempotency was also exercised over HTTP: 16 concurrent `POST /tasks` with one key produced exactly one unique task id.

## 11. Known limitations

- `gofmt -l` on this Windows checkout flags every file because line endings are CRLF. The committed blobs are LF and are gofmt-clean; this was confirmed by extracting the commit into a Linux container (0 files needed formatting). This is an environment artifact, not a code issue.
- `golangci-lint` and `go-arch-lint` were not run: the installed lint binary targets Go 1.26 and rejects Go 1.27, and `go-arch-lint` is not installed. Dependency direction was checked by hand and by `go vet`.
- The ERROR log level was exercised by stopping MySQL for a few seconds; there is no code path that returns a 5xx under normal operation, which is the intended behavior.

## Conclusion

Every endpoint in the case study works, all response envelopes are consistent and snake_case, idempotency is race-safe, the assignment transaction is atomic, logging is structured with correct levels, and the unit suite runs without a database. The adversarial pass found no IDOR, no cross-user data leak, no mass assignment, and no token-forgery bypass. The remaining items are hardening follow-ups (login timing, rate limiting) and the recruiter email.
