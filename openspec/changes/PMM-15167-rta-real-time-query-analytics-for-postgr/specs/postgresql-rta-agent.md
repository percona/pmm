# PostgreSQL RTA Agent

## Requirement: Poll active sessions from pg_stat_activity

The PostgreSQL RTA agent MUST poll `pg_stat_activity` (joined with `pg_locks` where applicable) at the configured collect interval (1–5 seconds, default 2 seconds) and produce `QueryData` messages with a `QueryPostgreSQLData` payload for each active backend session.

### Scenario: Active query appears in next poll cycle

- **GIVEN** a PostgreSQL service with RTA session running and collect interval set to 2 seconds
- **WHEN** a long-running `SELECT` query is executing on the monitored instance
- **THEN** the agent sends a `QueryData` record within the next poll cycle containing the query text, session state, and query execution duration

### Scenario: Collect interval is configurable

- **GIVEN** an RTA session with collect interval set to 5 seconds
- **WHEN** the agent runs its collection loop
- **THEN** polls occur at approximately 5-second intervals (±500ms tolerance)

## Requirement: Lock chain resolution

The agent MUST resolve lock contention by joining `pg_stat_activity` with `pg_locks` to identify blocker → blocked chains, including query text and duration for each link in the chain.

### Scenario: Blocked session shows lock chain

- **GIVEN** session A holds a row lock and session B is waiting on that lock
- **WHEN** the agent polls during the contention
- **THEN** session B's record includes a lock chain with session A as blocker, showing A's query text and duration

## Requirement: Idle-in-transaction detection

The agent MUST detect sessions in `idle in transaction` state and report transaction duration separately from query execution duration.

### Scenario: Idle-in-transaction session is flagged

- **GIVEN** a session that started a transaction, ran a query, and is now idle within the open transaction
- **WHEN** the agent polls
- **THEN** the record has state `idle in transaction`, `is_idle_in_transaction=true`, and transaction duration greater than query execution duration

## Requirement: Parallel worker grouping

On PostgreSQL 13+, the agent MUST identify parallel worker sessions and associate them with their leader PID for UI collapse.

### Scenario: Parallel workers linked to leader

- **GIVEN** a parallel query running with a leader and two workers on PostgreSQL 14
- **WHEN** the agent polls
- **THEN** worker records include `leader_pid` pointing to the leader session's PID

## Requirement: query_id with fingerprint fallback

On PostgreSQL 14+, the agent MUST populate `query_id` from `pg_stat_activity.query_id`. On PostgreSQL 12–13, it MUST fall back to a query fingerprint.

### Scenario: query_id on PostgreSQL 14+

- **GIVEN** a monitored PostgreSQL 14 instance with an active query
- **WHEN** the agent collects the session
- **THEN** `query_id` is populated from `pg_stat_activity.query_id`

### Scenario: Fingerprint fallback on PostgreSQL 12

- **GIVEN** a monitored PostgreSQL 12 instance with an active query
- **WHEN** the agent collects the session
- **THEN** `query_id` contains a deterministic fingerprint of the query text

## Requirement: Permission error handling

When the monitoring user lacks `pg_read_all_stats`, the agent MUST report a specific actionable error (not an empty result set).

### Scenario: Missing pg_read_all_stats

- **GIVEN** a PostgreSQL service where the monitoring user does not have `pg_read_all_stats`
- **WHEN** the RTA agent starts
- **THEN** the agent status is `AGENT_STATUS_INITIALIZATION_ERROR` with a message explaining the required grant

## Requirement: RTA and QAN independence

RTA and QAN agents MUST operate independently on the same PostgreSQL instance without conflict or data duplication.

### Scenario: Both QAN and RTA active

- **GIVEN** a PostgreSQL service with both QAN (pg_stat_statements) and RTA agents running
- **WHEN** queries execute on the instance
- **THEN** QAN continues collecting to ClickHouse and RTA continues streaming live sessions with no interference

## Requirement: Supported PostgreSQL versions

The agent MUST work on PostgreSQL 12+, Percona Distribution for PostgreSQL, and major cloud variants (RDS, Aurora, Cloud SQL, Azure).

### Scenario: Percona Distribution for PostgreSQL

- **GIVEN** a Percona Distribution for PostgreSQL 16 instance registered in PMM
- **WHEN** RTA session is started
- **THEN** live session data is collected and forwarded successfully

## Requirement: Performance overhead

At a 2-second polling interval, the agent MUST add less than 1% additional CPU on a host with ≤200 active backends.

### Scenario: CPU overhead within budget

- **GIVEN** a PostgreSQL instance with 200 active backends and RTA polling at 2 seconds
- **WHEN** baseline and RTA-enabled CPU usage are measured over 5 minutes
- **THEN** additional CPU consumption is less than 1% of baseline
