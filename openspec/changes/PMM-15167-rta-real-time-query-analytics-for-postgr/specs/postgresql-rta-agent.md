# PostgreSQL RTA Agent

Capability spec for the pmm-agent PostgreSQL Real-Time Analytics collector.

## Requirements

### REQ-PG-AGENT-001: Session polling

The PostgreSQL RTA agent SHALL poll `pg_stat_activity` at a configurable interval (default 2 seconds, range 1–5 seconds) and produce one `QueryData` record per active backend session (excluding the agent's own connection).

### REQ-PG-AGENT-002: Lock chain collection

The PostgreSQL RTA agent SHALL query `pg_locks` joined with `pg_stat_activity` on each poll tick and attach lock chain entries to each blocked session's `QueryPostgreSQLData.lock_chain`.

### REQ-PG-AGENT-003: Version-aware query_id

The agent SHALL populate `QueryData.query_id` from `pg_stat_activity.query_id` on PostgreSQL 14+ and fall back to a deterministic query fingerprint on PostgreSQL 12–13.

### REQ-PG-AGENT-004: Parallel worker metadata

On PostgreSQL 13+, the agent SHALL include `leader_pid` and `backend_type` so the UI can group parallel workers under their leader.

### REQ-PG-AGENT-005: Idle-in-transaction detection

The agent SHALL set `state` to the exact PostgreSQL session state (including `idle in transaction`) and include `transaction_start_time` and `state_change_time` for duration calculations.

### REQ-PG-AGENT-006: Permission error reporting

When the monitoring user lacks `pg_read_all_stats`, the agent SHALL report a specific, actionable error (not silently return empty data).

### REQ-PG-AGENT-007: QAN coexistence

The agent SHALL operate independently of QAN agents (`pg_stat_statements` / `pg_stat_monitor`) on the same PostgreSQL instance without conflict or data duplication.

## Scenarios

### Scenario: Active query session collected

**GIVEN** a PostgreSQL service with an enabled RTA session and a running `SELECT` query on PID 1234
**WHEN** the agent completes a poll tick
**THEN** a `QueryData` record is emitted with `query_text` matching the active query, `query_execution_duration` reflecting elapsed time, and `QueryPostgreSQLData.pid` = 1234

### Scenario: Lock chain populated

**GIVEN** session PID 200 holds a lock that blocks PID 201
**WHEN** the agent completes a poll tick
**THEN** the `QueryData` for PID 201 includes a `lock_chain` entry with PID 200 as blocker, including lock mode, query text, and duration

### Scenario: query_id on PostgreSQL 14+

**GIVEN** a PostgreSQL 15 instance with an active query that has a non-zero `query_id`
**WHEN** the agent collects the session
**THEN** `QueryData.query_id` equals the PostgreSQL `query_id` value as a string

### Scenario: Fingerprint fallback on PostgreSQL 12

**GIVEN** a PostgreSQL 12 instance (no `query_id` column)
**WHEN** the agent collects an active session
**THEN** `QueryData.query_id` contains a deterministic fingerprint derived from normalized query text

### Scenario: Missing pg_read_all_stats

**GIVEN** the monitoring user does NOT have `pg_read_all_stats` privilege
**WHEN** the agent attempts to poll `pg_stat_activity`
**THEN** the agent reports an error indicating the missing privilege (not an empty result set)

### Scenario: Parallel workers tagged

**GIVEN** a PostgreSQL 15 parallel query with leader PID 500 and worker PID 501
**WHEN** the agent collects both sessions
**THEN** worker PID 501 has `QueryPostgreSQLData.leader_pid` = 500 and `backend_type` = `parallel worker`
