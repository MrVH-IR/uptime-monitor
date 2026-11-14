# Uptime Monitor

A lightweight uptime monitoring service written in Go.

Uptime Monitor periodically checks configured websites and records their availability and response times.

## Features

- HTTP availability checks
- HTTP status code monitoring
- Response time measurement
- Configurable HTTP methods (`GET`, `HEAD`)
- Configurable check intervals and timeouts
- Concurrent monitoring using Goroutines and Worker Pools
- PostgreSQL for storing monitors and check history

> **Development status:** The project is under active development. HTTP Checker, database access, monitor management, and their initial tests are implemented. Scheduler and Worker Pool integration is in progress.

## Tech Stack

- **Language:** Go
- **Database:** PostgreSQL
- **Database Driver:** pgx
- **Concurrency:** Goroutines, Channels, Worker Pools
- **HTTP Client:** Go `net/http`

## Project Structure

```text
.
├── cmd/
│   └── monitor/
│       └── main.go
├── internal/
│   ├── checker/
│   │   ├── checker.go
│   │   ├── http.go
│   │   └── http_test.go
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── postgres.go
│   ├── monitor/
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── scheduler/
│   │   └── scheduler.go
│   └── worker/
│       └── pool.go
├── migrations/
│   ├── 000001_create_monitors.up.sql
│   ├── 000001_create_monitors.down.sql
│   ├── 000002_create_check_results.up.sql
│   └── 000002_create_check_results.down.sql
├── docker-compose.yml
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

## Requirements

- Go
- PostgreSQL
- Docker and Docker Compose (optional)

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/MrVH-IR/uptime-monitor.git
cd uptime-monitor
```

### 2. Configure environment variables

```bash
cp .env.example .env
```

Update `.env`:

```ini
APP_ENV=development
DATABASE_URL=postgres://monitor:secret@localhost:5432/monitor?sslmode=disable
```

### 3. Start PostgreSQL using Docker Compose

```bash
docker compose up -d
```

### 4. Run database migrations

Make sure `psql` is installed and your PostgreSQL database is accessible.

```bash
set -a
source .env
set +a

psql "$DATABASE_URL" -f migrations/000001_create_monitors.up.sql
psql "$DATABASE_URL" -f migrations/000002_create_check_results.up.sql
```

### 5. Install Go dependencies

```bash
go mod tidy
```

### 6. Run the application

```bash
go run ./cmd/monitor
```

## Running Tests

Run all tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

Run static analysis:

```bash
go vet ./...
```

## Building

Build the application:

```bash
go build -o bin/uptime-monitor ./cmd/monitor
```

Run the compiled binary:

```bash
./bin/uptime-monitor
```

## Database Schema

### monitors

Stores websites and their monitoring configurations.

| Column | Description |
|---|---|
| id | Unique monitor identifier |
| name | Monitor name |
| url | Target URL |
| http_method | HTTP method (GET / HEAD) |
| interval_seconds | Monitoring interval |
| timeout_seconds | Request timeout |
| is_active | Monitor activation status |
| next_check_at | Next scheduled check |
| last_checked_at | Last check timestamp |
| created_at | Creation timestamp |
| updated_at | Last update timestamp |

### check_results

Stores historical monitoring results.

| Column | Description |
|---|---|
| id | Unique result identifier |
| monitor_id | Related monitor |
| status_code | HTTP response status |
| response_time_ms | Response time in milliseconds |
| is_success | Check result |
| error_type | Error classification |
| error_message | Error description |
| started_at | Check start timestamp |
| finished_at | Check completion timestamp |

## Development Roadmap

- [x] Project structure and architecture
- [x] Database schema and migrations
- [x] Configuration management
- [x] PostgreSQL connection
- [x] Monitor model and repository
- [x] Monitor service and input validation
- [x] HTTP Checker
- [x] HTTP Checker tests
- [x] Worker Pool and concurrent checks
- [x] Scheduler
- [x] Persist check results
- [ ] Update monitor check timestamps
- [ ] Graceful shutdown
- [ ] Integration tests
- [ ] Dockerized application deployment

## License

MIT