# Task Management API

This is a REST API for tasks. Many users can use it. Each user has their own tasks.

The API is written in Go 1.27. It uses chi for routes, MySQL for data, JWT for login, bcrypt for passwords, and `log/slog` for logs.

## Settings

Copy `.env.example` to `.env`. Compose and the app read it. It is the one place for every setting.

| Variable | Example | Notes |
|---|---|---|
| `JWT_SECRET` | — | Required. Use at least 32 characters. The app will not start without it. Make one with `make jwt-secret` or `scripts\generate-jwt-secret.ps1` |
| `DATABASE_URL` | `taskmanagement:taskmanagement@tcp(mysql:3306)/taskmanagement?parseTime=true` | The MySQL DSN. Inside Compose the host is `mysql` |
| `MYSQL_ROOT_PASSWORD` | `rootpassword` | The MySQL root password |
| `MYSQL_DATABASE` | `taskmanagement` | The database name |
| `MYSQL_USER` | `taskmanagement` | The MySQL user |
| `MYSQL_PASSWORD` | `taskmanagement` | The MySQL password |
| `API_PORT` | `44001` | The host port for the API, bound to `127.0.0.1` |
| `MYSQL_PORT` | `44002` | The host port for MySQL, bound to `127.0.0.1` |

The app reads `APP_ADDR` too. It defaults to `:8080`, which matches the container. The `DB_*` pool settings (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`, `DB_CONN_MAX_IDLE_TIME`, `DB_PING_TIMEOUT`) are optional and have defaults. You do not have to set them.

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

The parts follow one direction:

```text
adapter/api  ->  service  ->  domain  <-  repository
```

- `domain` knows only the data and the errors. It has no SQL and no HTTP.
- `service` knows the rules. It uses the `domain` interfaces.
- `repository` knows MySQL. `adapter/api` knows HTTP.
- A module does not import another module. `teams` and `tasks` meet only through the database and the shared packages.
- `cmd/api/main.go` is the only place that builds the parts and joins them.

## Run with Docker

The API image is built in two steps with alpine. It is about 26MB. The database is MySQL 9.7.2.

Start the stack:

```text
docker compose up -d --build
```

Stop the stack:

```text
docker compose down
```

- The API uses host port `44001`. MySQL uses host port `44002`. Both bind to `127.0.0.1`, so they are not open on the network.
- A `migrate` service applies `migrations/001..008` (the schema and the test data) before the API starts. It uses `migrate/migrate` and keeps its place in a `schema_migrations` table.
- Each migration runs one time only. When you add a new file, the next `up` applies only the new file. You keep your data.
- `down` stops the stack. `down -v` also deletes the data. Use `down -v` only to reset everything from zero.
- The app reads every setting from `.env` through `env_file`. The file is required.
- `JWT_SECRET` has no default. The app exits when it is empty.

Before the first start, copy the settings file and set the secret (on PowerShell use `Copy-Item .env.example .env`):

```text
cp .env.example .env
# Put a value in JWT_SECRET. Make one with `make jwt-secret`.
docker compose up -d --build
```

## Quick start

The folder `postman/` has a collection for this API. The file is `postman/task-management-api.postman_collection.json`. It has 5 folders and 18 requests in total.

Import it:

1. Open Postman.
2. Click Import.
3. Choose `postman/task-management-api.postman_collection.json`.

Then run the requests from top to bottom: Health, Auth, Teams, Tasks, Cleanup.

The collection keeps its own variables. It does the link work for you:

| Variable | Set by | Meaning |
|---|---|---|
| `base_url` | the collection | `http://localhost:44001` |
| `access_token` | `POST /auth/login`, `POST /auth/refresh` | the bearer token for the later requests |
| `refresh_token` | `POST /auth/login`, `POST /auth/refresh` | the token for `refresh` and `logout` |
| `team_id` | `POST /teams` | the new team, used by the team and task requests |
| `task_id` | `POST /tasks` | the new task, used by the task requests |
| `user_id` | the collection | the seed user bob, used to add a member and to assign |
| `replay_idempotency_key` | the collection | the fixed key for the replay request |

Start with `POST /auth/login`. It uses a seed user: `alice@fatkulnurk.com` with password `password`. After that, the Teams and Tasks requests have a token.

### Run with Newman

You can run the whole collection from the command line:

```text
npx newman run postman/task-management-api.postman_collection.json
```

This makes 18 requests and 36 checks. All of them must pass. The API must be running first. The collection uses `http://localhost:44001`.

Two requests in the Tasks folder show idempotency:

- `POST /tasks` uses a new key each run, so it makes a new task.
- `POST /tasks (same key - replay)` uses a fixed key. The first run makes a task; later runs return that first task and no duplicate.

Run `Cleanup` last. It removes the member, and a member with an active task assignment cannot be removed.

## Test data

The seed migration makes three users. All of them use the password `password`:

- `alice@fatkulnurk.com`
- `bob@fatkulnurk.com`
- `carol@fatkulnurk.com`

It also makes 2 teams, 5 team memberships, 7 tasks (1 is soft-deleted), and 11 task logs.

## Endpoints

All endpoints need `Authorization: Bearer <access_token>`, except `GET /health` and the `auth` routes.

| Method | Path | Status | Description |
|---|---|---|---|
| `GET` | `/health` | `204` | Checks that the app is alive |
| `POST` | `/auth/register` | `201` | Makes a new user |
| `POST` | `/auth/login` | `200` | Logs in and returns an access token and a refresh token |
| `POST` | `/auth/refresh` | `200` | Gives a new token pair |
| `POST` | `/auth/logout` | `204` | Deletes the refresh token you send |
| `POST` | `/teams` | `201` | Makes a new team |
| `GET` | `/teams` | `200` | Lists your teams |
| `GET` | `/teams/{id}` | `200` | Shows one team |
| `GET` | `/teams/{id}/members` | `200` | Lists the members of a team |
| `POST` | `/teams/{id}/members` | `201` | Adds a user by `user_id` or `email` |
| `DELETE` | `/teams/{id}/members/{user_id}` | `204` | Removes a member who is not the owner |
| `POST` | `/tasks` | `201` | Makes a new task |
| `GET` | `/tasks` | `200` | Lists tasks |
| `GET` | `/tasks/{id}` | `200` | Shows one task |
| `PUT` | `/tasks/{id}` | `200` | Changes a task |
| `DELETE` | `/tasks/{id}` | `204` | Soft-deletes a task |
| `POST` | `/tasks/{id}/assign` | `200` | Assigns or unassigns a task |

The team endpoints are extra. They exist because every task belongs to a team. The `POST /tasks/{id}/assign` endpoint needs a team. It assigns a task to a user in the same team.

### Auth requests

```text
POST /auth/register   {"name":"Dewi","email":"dewi@example.com","password":"password123"}
POST /auth/login      {"email":"alice@fatkulnurk.com","password":"password"}
POST /auth/refresh    {"refresh_token":"<REFRESH_TOKEN>"}
POST /auth/logout     {"refresh_token":"<REFRESH_TOKEN>"}
```

The password must have 8 characters or more. The email must be unique. `login` and `refresh` give a token pair.

### `POST /tasks`

You must send an `Idempotency-Key` header. The value must be a UUID.

If you send the same request again with the same key, you get the same answer and no new task is made, even when the body is different. This works for 24 hours.

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

## Who can do what

You need a token for every `/tasks` and `/teams` route. `GET /health` and the `auth` routes do not need one.

Tasks:

- The creator can read, change, delete, and assign the task.
- The assignee can read the task and change only the `status`.
- Other users cannot see the task.

Teams:

- The owner can add and remove members.
- A member can read the team and its members.

If you are not part of a task or a team, the API answers `404`. It does not answer `403`. This way a stranger cannot learn that the resource exists. The API answers `403` only when you can see the resource but the action is not yours.

## Security notes

- The password is stored with bcrypt. The API never stores the plain password.
- The access token is a JWT. It uses `HS256` and always has an `exp`. The secret must be 32 characters or more.
- The refresh token is a random string. The database stores only its hash. Each `refresh` gives a new refresh token and drops the old one. `logout` drops it too.
- An idempotency key is private to one user. The stored response is counted per user and key.

## Errors and logs

All errors use the same JSON format:

```json
{"status":404,"code":"not_found","message":"task not found","timestamp":"2026-09-10T10:00:00Z"}
```

Validation errors also have an `errors` array. Each item has `field`, `code`, and `message`. The status codes are: `400` for bad input, `401` for no valid token, `403` for not allowed, `404` for not found or hidden, `409` for a conflict, `422` for a validation error, and `500` for an internal error.

Some errors carry a specific `code`: `user_already_team_member` (`409`), `owner_cannot_be_removed` (`403`), and `member_has_active_assignments` (`409`).

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
