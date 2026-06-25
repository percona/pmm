# qan-api2 Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [managed/AGENTS.md](../managed/AGENTS.md) (forwards QAN data) · [agent/AGENTS.md](../agent/AGENTS.md) (QAN collectors) · [api/AGENTS.md](../api/AGENTS.md) (QAN API definitions)

**qan-api2** is the Query Analytics API service for PMM. It receives query performance data from pmm-agent (via pmm-managed), stores it in ClickHouse, and serves analytics queries (reports, filters, metrics, examples) through gRPC and REST APIs.

## Architecture

### Data Flow

```
pmm-agent (QAN collectors)
  → pmm-managed (gRPC receiver)
    → qan-api2 CollectorService.Collect (gRPC, port 9911)
      → MetricsBucket (batched writer)
        → ClickHouse `metrics` table

PMM UI / API clients
  → qan-api2 QANService (gRPC/REST, ports 9911/9922)
    → Reporter / Metrics models (SQL queries)
      → ClickHouse
```

### Server Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 9911 | gRPC | `CollectorService` (data ingestion) and `QANService` (analytics) |
| 9922 | HTTP/JSON | gRPC-Gateway REST API |
| 9933 | HTTP | Debug endpoints (`/debug/metrics`, `/debug/pprof`, `/debug/vars`) |

## Domain Model

### Core Data: `metrics` Table (ClickHouse)

The `metrics` table stores one row per query fingerprint per collection period:
- **Dimensions**: `service_name`, `database`, `schema`, `username`, `client_host`, `node_id`, `service_id`, `service_type`, `node_name`, `node_type`, `machine_id`, `container_id`, `container_name`
- **Query identity**: `queryid` (fingerprint), `fingerprint`, `query` (example)
- **Metrics**: `num_queries`, `m_query_time_sum/min/max/p99`, `m_lock_time_*`, `m_rows_sent_*`, `m_rows_examined_*`, `m_bytes_sent_*`, and many more
- **Time**: `period_start` (DateTime), `period_length` (seconds)

### Key Types

| Type | File | Purpose |
|------|------|---------|
| `MetricsBucket` | `models/data_ingestion.go` | Batched writer: buffers metrics (500ms or 100 items), bulk-inserts to ClickHouse |
| `Reporter` | `models/reporter.go` | Builds dynamic SQL for reports using `text/template` |
| `Metrics` | `models/metrics.go` | Metric name resolution and sparkline queries |

## Database: ClickHouse

### Access Pattern
- **Driver**: `github.com/ClickHouse/clickhouse-go/v2` (native protocol)
- **Query layer**: `github.com/jmoiron/sqlx` on top of `database/sql` — **no ORM**
- **Reads**: `db.Select()`, `db.Queryx()` with raw SQL and `text/template` for dynamic query building
- **Writes**: Prepared `INSERT INTO metrics` statement via `stmt.Exec()` in batched writer
- **Cluster support**: ReplicatedMergeTree engine when `--clickhouse-cluster` is set

### Migrations
- **Tool**: `github.com/golang-migrate/migrate/v4` with ClickHouse driver
- **Source**: Embedded SQL files (`migrations/sql/`) via `embed.FS`
- **Templating**: `utils/templatefs` selects MergeTree vs ReplicatedMergeTree based on cluster config
- **Flow**: `migrations.Run()` in `db.go` runs `Up()` on startup; handles dirty-state recovery
- **Standalone**: `cmd/clickhouse_migrate/main.go` for backup/restore scenarios

### Data Retention
- `DropOldPartition()` runs periodically (default every 24h) to drop partitions older than `--data-retention`

## Patterns and Conventions

### Do
- Use `sqlx` for all ClickHouse queries — no ORM
- Use `text/template` for building dynamic SQL (see `models/reporter.go`)
- Batch writes via `MetricsBucket` — never insert individual rows
- Use prepared statements for inserts
- Add ClickHouse migrations as numbered SQL files in `migrations/sql/`
- Support cluster mode: use template conditions for ReplicatedMergeTree vs MergeTree
- Use LBAC (Label-Based Access Control) filters from `X-Proxy-Filter` header when building reports

### Don't
- Don't use an ORM for ClickHouse — raw SQL with sqlx is the pattern
- Don't insert metrics one at a time — always use the batched writer
- Don't modify migration files after they've been released — create new migrations
- Don't hardcode table engines — use templates for cluster compatibility

## Configuration

CLI flags (parsed via `kingpin`):
- `--grpc-bind` (default `:9911`) — gRPC listen address
- `--json-bind` (default `:9922`) — JSON/REST listen address
- `--listen-debug-addr` (default `127.0.0.1:9933`) — debug endpoint
- `--dsn` — ClickHouse DSN (alternative to individual `--clickhouse-*` flags)
- `--clickhouse-addr`, `--clickhouse-database`, `--clickhouse-user`, `--clickhouse-password` — ClickHouse connection parameters
- `--clickhouse-cluster` — enable cluster mode
- `--clickhouse-cluster-name` — cluster name for ReplicatedMergeTree
- `--data-retention` — how long to keep data (default 30 days)
- `--debug`, `--trace` — logging levels

## Testing

- Unit tests: `*_test.go` next to implementation
- Test data: JSON fixtures in `test_data/`
- Run: `make test` (requires ClickHouse; use `docker-compose.yaml` for local dev)
- Coverage: `maincover_test.go` with `maincover` build tag

## Code Generation

- Protobuf types come from `/api/qan/v1/` — run `make gen` from repo root if proto files change
- No reform or other model generation inside qan-api2

## Key Files to Reference

- `qan-api2/main.go` — entry point, server wiring, flag definitions
- `qan-api2/db.go` — ClickHouse connection, migrations, partition management
- `qan-api2/models/data_ingestion.go` — `MetricsBucket` batched writer
- `qan-api2/models/reporter.go` — dynamic SQL report builder
- `qan-api2/services/analytics/profile.go` — `GetReport` implementation
- `qan-api2/services/receiver/receiver.go` — data ingestion from agents
- `qan-api2/migrations/sql/` — ClickHouse schema migrations
