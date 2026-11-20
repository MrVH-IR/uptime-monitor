# Uptime Monitor · دیدبان

A small HTTP uptime monitor built with Go, PostgreSQL, and React. A single Go process schedules checks, runs four concurrent workers, saves results, and serves a Persian dashboard and read-only API. No authentication is required.

## What it does

- Checks HTTP/HTTPS targets using GET or HEAD, with a timeout per monitor.
- Treats final HTTP status codes from 200 through 399 as successful.
- Records response time, HTTP status, network errors, and check timestamps.
- Saves each result and the next check time in one database transaction.
- Claims due monitors using PostgreSQL row locks and temporary leases.
- Displays status and paginated history; refreshes every 10 seconds.
- Returns at most 50 records per API page.

## Run locally

Requirements: Go matching `go.mod`, Node 26 through nvm, and PostgreSQL with the project migrations applied.

```bash
cp .env.example .env
```

Set `DATABASE_URL` to your local database connection. Example:

```ini
APP_ENV=development
DATABASE_URL=postgres://monitor_user:secret@localhost:5432/monitor_db?sslmode=disable
```

For a new, empty local database, apply the migrations in order:

```bash
set -a
source .env
set +a
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000001_create_monitors.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000002_create_check_results.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000003_add_monitor_lock.up.sql
```

Build the dashboard and start the service from the repository root:

```bash
source ~/.nvm/nvm.sh
cd frontend
nvm use 26
npm ci
npm run build
cd ..
go run ./cmd/monitor
```

Open <http://127.0.0.1:8080>. For frontend development, run `npm run dev` in `frontend` and open <http://127.0.0.1:5173>. Vite proxies `/api` to Go on port 8080.

## Run with Docker Compose

```bash
cp .env.example .env
# Set POSTGRES_PASSWORD in .env to a strong URL-safe password.
docker compose up --build -d
docker compose ps
docker compose logs -f app
```

Compose starts PostgreSQL 17, runs pending migrations, then starts the app. Dashboard assets are built inside the multi-stage Docker build. The runtime runs as a non-root user with a read-only filesystem. The app restarts after failure and has a database-backed health check.

Open <http://127.0.0.1:8080>. Only the app's loopback port is published; PostgreSQL stays on the Compose network. Docker uses `POSTGRES_PASSWORD`; the local Go process uses `DATABASE_URL`. These settings are independent.

Use letters, digits, hyphens, and underscores in `POSTGRES_PASSWORD` because it is interpolated into a connection URL. Changing that variable after database initialization does not change the database role password; rotate it explicitly in PostgreSQL.

Data persists in the `postgres17_data` volume. This is separate from the earlier development `postgres_data` volume and from any manually started PostgreSQL container. Existing data is not automatically imported. Back up and restore it explicitly if needed; never point a PostgreSQL 17 image at a volume initialized by a different major version.

Stop without removing data:

```bash
docker compose down
```

Do not use `docker compose down -v` unless you intend to delete the database.

### Migrations

The `migrate` service records applied filenames in `schema_migrations` and applies each new migration transactionally. Add new `*.up.sql` files; do not edit applied migrations. Run one deployment/migration process at a time. For databases previously migrated manually, record the already-applied filenames in `schema_migrations` before using this runner; it does not infer existing schema versions.

### Backup and restore

```bash
docker compose exec -T postgres pg_dump -U monitor_user -d monitor_db -Fc > monitor.dump
# Restore into an empty database using pg_restore; verify backups regularly.
```

## Add monitors

There is no write API. Manage monitors through the Go service or SQL:

```bash
docker compose exec -T postgres psql -U monitor_user -d monitor_db <<'SQL'
INSERT INTO monitors (name, url, http_method, interval_seconds, timeout_seconds)
VALUES ('Example', 'https://example.com', 'GET', 30, 10);
SQL
```

The next scheduler poll picks up active due monitors. Disable one with `UPDATE monitors SET is_active=false WHERE id=...`. Checks already running may still finish. Deleting a monitor also deletes its history through the foreign key cascade.

## API

All API routes are read-only. JSON field names follow the Go model names, for example `ID`, `MonitorID`, and `StartedAt`.

| Route | Result |
| --- | --- |
| `GET /api/monitors?page=1` | Monitors, newest ID first |
| `GET /api/results?page=1` | Latest result per monitor, newest check first |
| `GET /api/results?monitor_id=1&page=1` | One monitor's history, newest check first |
| `GET /healthz` | 200 if PostgreSQL responds, otherwise 503 |

`GET /api/results?monitor_ids=1,2,3` batches latest results for up to 50 monitor IDs, avoiding one dashboard request per monitor. It cannot be combined with `monitor_id`.

List responses remain JSON arrays. Pagination headers:

- `X-Page`: current page, default 1.
- `X-Page-Size`: fixed at 50.
- `X-Has-More`: whether another page exists.

Invalid page numbers or monitor IDs return 400. Pages beyond the end return an empty array. Database failures return 500 without exposing database details. Pagination uses offsets; new records can shift page boundaries between requests.

```bash
curl -i 'http://127.0.0.1:8080/api/results?monitor_id=1&page=2'
```

## Configuration

| Variable | Purpose |
| --- | --- |
| `APP_ENV` | Required environment label |
| `DATABASE_URL` | Required PostgreSQL URL for Go |
| `HTTP_ADDR` | HTTP bind address; defaults to `127.0.0.1:8080`, Docker uses `0.0.0.0:8080` |
| `POSTGRES_PASSWORD` | Required by Docker Compose |

A `.env` file is optional when process environment variables are supplied. Do not commit credentials. Docker excludes `.env`, Git metadata, and local frontend build files from its build context.

## Validation

```bash
go build ./...
go vet ./...
go test ./internal/api ./internal/checker ./internal/worker ./internal/scheduler
cd frontend && npm ci && npm run build
```

Repository tests require a **dedicated test database** with all migrations applied:

```bash
DATABASE_URL='postgres://user:password@localhost:5432/monitor_test?sslmode=disable' go test -race ./...
```

Never run repository tests against a production database: they insert/delete test monitors and claim due rows. HTTP tests open local ports.

## Operational scope

This is a small-service deployment, with four workers and a one-second polling interval. Run one app instance for this deployment. Interrupted checks can be retried after their leases expire; this is not an exactly-once system. Logs go to stdout/stderr. SIGINT and SIGTERM stop scheduling and drain completed results before closing the database.

No authentication is included. Keep the published listener on localhost or a trusted private network; use a TLS reverse proxy if exposing it remotely. Historical results are retained indefinitely: plan database backups and a retention policy appropriate to your workload. Load testing and multi-replica operation require separate validation.

## Structure

```text
cmd/monitor/       Application entry point and HTTP lifecycle
internal/api/      Read API, health endpoint, static dashboard
internal/checker/  HTTP checks and error classification
internal/monitor/  Monitor service and PostgreSQL repository
internal/scheduler/ Scheduling and result handling
internal/worker/   Bounded concurrent workers
internal/config/   Environment configuration
internal/database/ PostgreSQL connection
frontend/          React/Vite dashboard
migrations/        PostgreSQL schema migrations
scripts/           Docker migration runner
```

MIT License.
