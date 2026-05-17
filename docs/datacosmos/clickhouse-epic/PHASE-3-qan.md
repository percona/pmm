# Phase 3 — Query Analytics (QAN) for ClickHouse

**Outcome:** PMM's QAN shows ClickHouse query-level analytics — fingerprinted
queries, counts, latency and ClickHouse-specific metrics (rows/bytes read,
memory) — sourced from `system.query_log`, on par with MySQL/PostgreSQL QAN.

## Design

### Collection model — per-event watermark (not counter-diff)

`system.query_log` is an **append-only event table** (one row per query phase),
like the MySQL slowlog — **not** a cumulative-counter table like
`pg_stat_statements`. The QAN agent therefore tracks a **watermark** (`event_time`)
and reads new rows each interval; it does not cache+diff a snapshot. Because
every individual execution is observed, true `min/max` and an exact `p99` are
computable per query class.

### qan-api2 is NOT DB-agnostic

`api/qan/v1/collector.proto` `MetricsBucket`, `qan-api2/models/data_ingestion.go`
and the ClickHouse `metrics` table all have explicit per-engine columns, and
`agent_type` is a fixed `Enum8`. **Phase 3 requires qan-api2 migrations + enum
changes** (contrary to the earlier assumption).

### Engine-specific metrics

Choose **Option B (first-class)**: a real `MetricsBucket.ClickHouse` message and
real qan-api2 columns (`m_read_rows_*`, `m_read_bytes_*`, `m_memory_usage_*`,
`m_result_*`, `m_written_*`). Option A (reuse MySQL columns) is the
lower-risk fallback if qan-api2 migration is deferred.

## Development line (ordered)

### Stage A — API / proto (must complete first; B/C/D import generated code)
- `api/inventory/v1/agents.proto` — `AGENT_TYPE_QAN_CLICKHOUSE_QUERYLOG_AGENT = 21`
  (`20` is taken by `AGENT_TYPE_CLICKHOUSE_EXPORTER` from Phase 1).
- `api/agent/v1/collector.proto` — `message ClickHouse {…}` inside `MetricsBucket`
  + `ClickHouse clickhouse = 5`. Fields: `m_read_rows_*`, `m_read_bytes_*`,
  `m_result_rows_*`, `m_result_bytes_*`, `m_memory_usage_*`, `m_written_rows_*`,
  `m_written_bytes_*` (each `cnt/sum/min/max/p99`), `query_kind`.
- `api/qan/v1/collector.proto` — flat `m_*` ClickHouse fields, **new** field
  numbers (310+; current max 309).
- `api/qan/v1/qan.proto` — corresponding `m_*_sum_per_sec` report fields.
- `make gen` (buf) — regenerate.

### Stage B — pmm-agent: the QAN collector
New package `agent/agents/clickhouse/querylog/` (template:
`agent/agents/postgres/pgstatstatements/`):
- `querylog.go` — `ClickHouseQueryLog` agent (`New`, `Run`, `Changes`,
  `Collect`, `Describe`) implementing `agents.BuiltinAgent`. `Run` schedules on
  minute boundaries; preflight checks `log_queries` setting and
  `system.query_log` existence; watermark + dedup (see Risks).
- `models.go` — `system.query_log` row + per-query aggregation structs.
- `fingerprint.go` — ClickHouse SQL normalization (see below).
- `makeBuckets` — pure, testable: group rows by fingerprint hash, build one
  `MetricsBucket` per class with `Common` (query time, db, tables, user, errors,
  example) + `ClickHouse` (sum/cnt/min/max/p99).
- `agent/agents/supervisor/supervisor.go` — `startBuiltin` case wiring
  `querylog.New(querylog.Params{DSN, AgentID, MaxQueryLength, …})`.

### Stage C — pmm-managed
- `managed/models/agent_model.go` / `agent_helpers.go` —
  `QANClickHouseQueryLogAgentType AgentType = "qan-clickhouse-querylog-agent"`;
  DSN/validity/metadata entries.
- `managed/services/agents/` — `qanClickHouseQueryLogAgentConfig` building the
  `SetStateRequest_BuiltinAgent`; wire into `state.go`.
- inventory/management/converters — Add/Change/Remove plumbing for the new QAN
  agent (mirror `qan-postgresql-pgstatements-agent`).
- `managed/services/qan/client.go` — `case m.Clickhouse != nil: fillClickHouse(…)`
  in the `Collect` switch + the `fillClickHouse` function.
- `pmm-admin add clickhouse --qan` flag (extends the Phase 1 command).

### Stage D — qan-api2
- New migrations `qan-api2/migrations/sql/` — add the `m_read_rows_*` etc.
  columns to the `metrics` table; `ALTER TABLE metrics MODIFY COLUMN agent_type
  Enum8(… , 'qan-clickhouse-querylog-agent'=<next ordinal>)`. Highest-numbered
  files so they apply last; down-migrations drop only the new columns.
- `qan-api2/models/data_ingestion.go` — new columns in the INSERT list + value
  list + the `agent_type` Enum8 cast + the ingestion row struct.
- `qan-api2/models/base.go` — add the agent type to `agentTypeToClickHouseEnum`.
- `qan-api2/models/reporter.go` — surface the new columns in the report mapping.

### Stage E — verification & docs
Unit + integration tests (below); update package and QAN docs.

Sequencing: **A → (B ∥ C ∥ D) → E**.

## Fingerprinting

`system.query_log` stores raw queries with literals. Strategy, in priority:
1. **Server hash** — when `normalized_query_hash` exists (CH ≥ 20.x, detect via
   `DESCRIBE TABLE system.query_log`), group by it (server-consistent, free).
2. **Client normalization** (`fingerprint.go`) — lexer-based literal stripping
   for the displayed fingerprint and old CH: numbers/strings → `?`,
   `IN (?,?,…)` → `IN (?)`, arrays `[…]` → `[?]`, tuples `(…)` → `(?)`,
   `LIMIT n[,m]` → `LIMIT ?`; keep `FORMAT …` and `{name:Type}` placeholders;
   strip comments (reuse `queryparser.MySQLComments`).
3. **Fallback** — raw truncated query hashed as-is on parse failure.
Hash with `cityHash64` → hex → `Common.Queryid`.

## Validation criteria

1. A ClickHouse service + QAN agent appears in the QAN UI with per-query rows.
2. N identical queries (varying literals) in one minute → **one** bucket,
   `num_queries == N`.
3. `m_query_time_sum`, `m_read_rows_sum`, `m_result_rows_sum`,
   `m_memory_usage_sum` non-zero and within tolerance of direct
   `system.query_log` values.
4. No double-counting across consecutive intervals (watermark/dedup correct).
5. Erroring queries → `num_queries_with_errors` + `Common.errors` populated.
6. `log_queries=0` or missing `system.query_log` → agent `WAITING` with a clear
   log message, no crash, auto-recovers when enabled.
7. Fingerprints group literal-only-different queries; distinct shapes differ.

## Integration tests

See [INTEGRATION-TESTS.md](INTEGRATION-TESTS.md) — IT-3.x: basic bucket,
distinct fingerprints (SELECT vs INSERT), incremental no-double-count, boundary
second, error query, metric accuracy vs `system.query_log`, lazy-table/`log_queries`
disabled, version-column matrix. Unit tests: `makeBuckets`, `fingerprint`,
`percentile`.

## Risks

- **`system.query_log` schema varies by CH version** — `result_bytes`,
  `normalized_query_hash`, `event_time_microseconds`, `query_kind` appeared in
  different releases. Mitigation: `DESCRIBE TABLE system.query_log` on startup,
  build the `SELECT` dynamically from available columns, default missing → 0.
- **Lazy table creation** — `system.query_log` exists only after the first
  logged query. Preflight + `WAITING` state + retry; never crash.
- **`log_queries` disabled** — no rows ever. Detect via `system.settings`;
  `WAITING` + WARNING; the agent must not change server settings.
- **Watermark/dedup** — `event_time` is 1-second granular on old CH; keep a
  `seenQueryIDs` set scoped to the watermark second; prefer
  `event_time_microseconds` when present.
- **qan-api2 enum/migration ordering** — the `agent_type` `Enum8` adds a new
  ordinal, never renumbers; migrations must be the highest-numbered files.
- **Proto field-number collisions** — `api/qan/v1/collector.proto` is densely
  numbered (max 309); new fields use 310+.
- **`query_log` flush latency** (`flush_interval_milliseconds`, ~7.5 s) and
  TTL/retention — tolerate; watermark starts at `now()` (no historical
  back-fill); never issue `SYSTEM FLUSH LOGS` from the agent.
