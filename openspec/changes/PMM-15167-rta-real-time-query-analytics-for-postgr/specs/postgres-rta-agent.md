# PostgreSQL RTA Agent

Capability: pmm-agent collects live PostgreSQL session data and streams it to PMM Server.

## Requirements

### REQ-PG-AGENT-001: Poll active sessions

The RTA PostgreSQL agent SHALL poll `pg_stat_activity` at a configurable interval between 1 and 5 seconds (default 2 seconds) when its session is running.

### REQ-PG-AGENT-002: Exclude idle backends

The agent SHALL include backends in states `active`, `idle in transaction`, and other non-`idle` client backend states. Pure `idle` backends SHALL be excluded unless they are `idle in transaction`.

### REQ-PG-AGENT-003: Collect lock information

The agent SHALL query `pg_locks` (joined with `pg_stat_activity`) on each collection cycle to determine lock contention and build blocker → blocked relationships.

### REQ-PG-AGENT-004: Reuse PostgreSQL credentials

The agent SHALL connect using credentials from the PostgreSQL exporter for the same PMM service unless explicitly overridden in inventory agent configuration.

### REQ-PG-AGENT-005: Permission error handling

When the monitoring user lacks `pg_read_all_stats` (and is not a superuser), the agent SHALL report a structured error with an actionable message rather than returning empty data silently.

### REQ-PG-AGENT-006: Query identification

- On PostgreSQL 14+, the agent SHALL populate `query_id` from `pg_stat_activity.query_id`.
- On PostgreSQL 12–13, the agent SHALL fall back to a stable query fingerprint hash.

### REQ-PG-AGENT-007: Query text truncation

When query text length equals or exceeds `track_activity_query_size`, the agent SHALL set a truncation flag and include the configured `track_activity_query_size` value in the payload.

### REQ-PG-AGENT-008: Parallel worker metadata

On PostgreSQL 13+, the agent SHALL include `leader_pid` for parallel worker backends.

### REQ-PG-AGENT-009: Performance budget

At a 2-second collect interval with ≤200 active backends, the agent SHALL add less than 1% additional CPU utilization on the PostgreSQL host.

## Scenarios

### Scenario: Active query appears in stream

**GIVEN** a running RTA session for a PostgreSQL service with an active `SELECT` query  
**WHEN** the agent completes a collection cycle  
**THEN** a `QueryData` message is emitted with `session_state=active`, non-empty `query_text`, and `query_execution_duration` reflecting elapsed query time

### Scenario: Idle in transaction session

**GIVEN** a backend in state `idle in transaction` with `xact_start` 30 seconds ago  
**WHEN** the agent collects session data  
**THEN** the payload includes `session_state=idle in transaction` and duration is computed from `xact_start`, not `query_start`

### Scenario: Lock chain detected

**GIVEN** session A holds a lock and session B is blocked waiting on A  
**WHEN** the agent collects lock data  
**THEN** session B's payload includes a lock chain with A as blocker, including A's query text and duration

### Scenario: Missing pg_read_all_stats

**GIVEN** the monitoring user cannot read all rows in `pg_stat_activity`  
**WHEN** the agent attempts collection  
**THEN** the agent status reports an error containing guidance to grant `pg_read_all_stats` or use a superuser role

### Scenario: Truncated query text

**GIVEN** `track_activity_query_size=1024` and a query longer than 1024 characters  
**WHEN** the agent collects the session  
**THEN** `query_text_truncated=true` and `track_activity_query_size=1024` are set in the payload

### Scenario: Parallel worker grouping metadata

**GIVEN** PostgreSQL 15 with a parallel query where worker PID 12345 has `leader_pid=12340`  
**WHEN** the agent collects sessions  
**THEN** the worker record includes `leader_pid=12340` and `backend_pid=12345`
