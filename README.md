# Task Management API

This is a REST API for tasks. Many users can use it. Each user has their own tasks.

The API is written in Go 1.27. It uses chi for routes, MySQL for data, JWT for login, bcrypt for passwords, and `log/slog` for logs.

## Settings

Copy `.env.example` to `.env`. The file `.env` loads by itself when you run the app. Then apply `migrations/001..008` in order. Apply the `.up` files first. Do not apply any `.down` file. Then run `go run ./cmd/api`.

| Variable | Default | Notes |
|---|---|---|
| `APP_ADDR` | `:8080` | The address the API listens on |
| `DATABASE_URL` | — | The MySQL DSN, for example `user:password@tcp(127.0.0.1:3306)/taskmanagement?parseTime=true` |
| `JWT_SECRET` | — | Required. Use at least 32 characters. The app will not start without it |
| `DB_MAX_OPEN_CONNS` | `25` | A whole number greater than zero |
| `DB_MAX_IDLE_CONNS` | `10` | A whole number. It cannot be more than the max open |
| `DB_CONN_MAX_LIFETIME` | `30m` | A Go duration |
| `DB_CONN_MAX_IDLE_TIME` | `5m` | A Go duration |
| `DB_PING_TIMEOUT` | `5s` | A Go duration |

You do not need `.env` if the settings are already in the environment. Docker Compose does this. The app never changes a setting that is already set.

## How the code is organized

The code has three parts: `auth`, `teams`, and `tasks`. They do not use each other. Each part has four folders:

```text
internal/modules/<module>/
├── module.go
├── domain/
├── repository/
├── service/
└── adapter/api/
```

- `domain` — the data and the errors.
- `repository` — it talks to MySQL.
- `service` — the rules and who can do what.
- `adapter/api` — it reads and writes HTTP.

`cmd/api/main.go` starts the app and connects all the parts.

Shared code lives in `internal/platform`. The token contract is in `internal/application/token`, and the JWT code is in `internal/platform/token/jwt`. The HTTP helpers are in `internal/platform/http`. They keep the error format the same for all errors. They also collect all validation errors in one response.

## Run with Docker

The API image is built in two steps with alpine. It is about 26MB. The database is MySQL 9.7.2.

```text
docker compose up -d --build
docker compose down
docker compose down -v
```

- The API uses host port `44001`. MySQL uses host port `44002`.
- On the first start, MySQL applies `migrations/001..008` (the schema and the test data). It reads them from `/docker-entrypoint-initdb.d`.
- `down` stops the stack. `down -v` also deletes the data. The init scripts run only when the data folder is empty. So use `down -v` after you change a migration or the test data.
- The settings are in `.env.example` (`MYSQL_*`, `API_PORT`, `MYSQL_PORT`, `JWT_SECRET`).

Before the first start, copy the settings file. Compose reads it by itself:

```text
cp .env.example .env
```

You can also set a value in the shell. The shell value wins over `.env`:

```text
$env:JWT_SECRET="change-me-to-a-long-secret"; docker compose up -d
```

Or point Compose at the file by hand:

```text
docker compose --env-file .env up -d
```

## Test data

The seed migration makes three users. All of them use the password `password`:

- `alice@fatkulnurk.com`
- `bob@fatkulnurk.com`
- `carol@fatkulnurk.com`

It also makes 2 teams, 5 team memberships, 7 tasks (1 is soft-deleted), and 11 task logs.

## Endpoints

All endpoints need `Authorization: Bearer <access_token>`, except `GET /health` and the `auth` routes.

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Checks that the app is alive. Returns `204` |
| `POST` | `/auth/register` | Makes a new user |
| `POST` | `/auth/login` | Logs in and returns an access token and a refresh token |
| `POST` | `/auth/refresh` | Gives a new refresh token |
| `POST` | `/auth/logout` | Deletes the refresh token you send |
| `POST` | `/teams` | Makes a new team |
| `GET` | `/teams` | Lists your teams |
| `GET` | `/teams/{id}` | Shows one team |
| `GET` | `/teams/{id}/members` | Lists the members of a team |
| `POST` | `/teams/{id}/members` | Adds a user by `user_id` or `email` |
| `DELETE` | `/teams/{id}/members/{user_id}` | Removes a member who is not the owner |
| `POST` | `/tasks` | Makes a new task |
| `GET` | `/tasks` | Lists tasks |
| `GET` | `/tasks/{id}` | Shows one task |
| `PUT` | `/tasks/{id}` | Changes a task |
| `DELETE` | `/tasks/{id}` | Soft-deletes a task |
| `POST` | `/tasks/{id}/assign` | Assigns or unassigns a task |

The team endpoints are extra. They exist because every task belongs to a team. The `POST /tasks/{id}/assign` endpoint needs a team. It assigns a task to a user in the same team.

### `POST /tasks`

You must send an `Idempotency-Key` header. The value must be a UUID.

If you send the same request again with the same key, you get the same answer. No new task is made. This works for 24 hours. If the body is different, you get `409`.

```text
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
Content-Type: application/json

{"team_id":"team-uuid","title":"Prepare report","description":"Weekly report","status":"todo"}
```

### `GET /tasks`

Query parameters: `status` (`todo` | `in_progress` | `done`), `search` (a part of the title), `page` (starts at `1`, default `1`), `limit` (`1`–`100`, default `20`), and `team_id` (optional).

```text
GET /tasks?page=1&limit=20&status=todo&search=report&team_id=team-uuid
```

### `POST /tasks/{id}/assign`

```json
{"assignee_id":"user-uuid"}
```

An empty `assignee_id` means unassign. Only the creator can assign. The target user must be in the same team.

**Case study requirement (original):**
● Tambahkan endpoint POST /tasks/:id/assign untuk assign task ke user lain dalam tim yang sama. Proses ini harus berjalan dalam satu database transaction: update assignee, catat riwayat perubahan ke tabel task_logs, dan kirim notifikasi (boleh mock/log saja).

This project meets that rule. The endpoint does three things in one database transaction:

1. It updates the `assignee`.
2. It writes the change to `task_logs`.
3. It sends a notification. The notification is a mock: it only writes a log line.

If one step fails, all the steps roll back. The database never keeps a half-done change.

## Errors and logs

All errors use the same JSON format:

```json
{"status":409,"code":"idempotency_key_reused","message":"unable to create task","timestamp":"2026-09-10T10:00:00Z"}
```

Validation errors also have an `errors` array. Each item has `field`, `code`, and `message`. The status codes are: `400` for bad input, `401` for no valid token, `403` for not allowed, `404` for not found or hidden, `409` for a conflict, `422` for a validation error, and `500` for an internal error.

Every response has an `X-Request-Id` header. The value is a UUID. If you send a valid UUID in the `X-Request-Id` header, the API uses it. If not, the API makes a new one. The API writes one JSON log line for each request. The line has `request_id`, `method`, `path`, `status`, and `latency`. The log level is INFO for 2xx and 3xx, WARN for 4xx, and ERROR for 5xx. If the code panics, the API logs the stack trace at ERROR. The response never shows the stack trace.

## Tests

```text
go test ./...
go vet ./...
go build ./...
gofmt -w internal cmd
```

You can also run `make test`, `make test-verbose`, `make test-cover`, and `scripts\test.bat`. `make test-cover` writes `coverage.out` and shows the coverage for each function. The coverage covers the business modules in `internal/modules/*/{adapter,repository,service}`.

### Race detector

`go test -race` needs cgo and a C compiler. Not every computer has them. Run it in a container instead:

```text
make test-race
scripts\test-race.bat
```

### Concurrency integration test

This test checks the idempotency rule against a real MySQL server. It starts 32 goroutines. All of them use the same `Idempotency-Key`. The test checks that the database has exactly one task, one task log, and one idempotency record. The test uses the `integration` build tag. So `go test ./...` does not need a database.

Start MySQL first, then run:

```text
docker compose up -d mysql
make test-integration
scripts\test-integration.bat
```

Both race targets use the `taskmanagement-gomod` volume to cache modules. The integration test makes a new schema for each run and drops it at the end.
