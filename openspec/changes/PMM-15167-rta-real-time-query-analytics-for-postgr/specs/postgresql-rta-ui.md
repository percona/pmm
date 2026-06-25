# PostgreSQL RTA UI

Capability spec for PMM UI Real-Time tab integration with PostgreSQL services.

## Requirements

### REQ-PG-UI-001: Service selection

The Real-Time tab service selector SHALL list registered PostgreSQL services alongside MongoDB services, using the existing Query Analytics → Real-Time navigation (no new pages).

### REQ-PG-UI-002: Live refresh

The Real-Time overview table SHALL refresh PostgreSQL session data at 1–5 second intervals matching the agent collect interval, with perceived latency ≤5 seconds.

### REQ-PG-UI-003: Lock chain panel

The UI SHALL display a lock chain panel showing blocker → blocked relationships with query text and duration for each link in the chain.

### REQ-PG-UI-004: Idle-in-transaction distinction

Sessions in `idle in transaction` state SHALL be visually distinguished from active queries, displaying transaction duration (time since `xact_start`) rather than query execution duration.

### REQ-PG-UI-005: Parallel worker grouping

On PostgreSQL 13+, parallel worker sessions SHALL be collapsed under their leader row by default, with an expand control to reveal workers.

### REQ-PG-UI-006: Query truncation indicator

When query text is truncated, the UI SHALL indicate truncation visually and show a tooltip explaining `track_activity_query_size` and that increasing it requires a PostgreSQL server restart.

### REQ-PG-UI-007: Permission error

When the agent reports insufficient privileges (`pg_read_all_stats`), the UI SHALL display a specific actionable error message instead of an empty or loading state.

### REQ-PG-UI-008: MongoDB unchanged

The existing MongoDB RTA UI experience SHALL remain unchanged when PostgreSQL RTA is added.

## Scenarios

### Scenario: Select PostgreSQL service in Real-Time tab

**GIVEN** a user with an active RTA session on a PostgreSQL service named `prod-pg-01`
**WHEN** the user navigates to Query Analytics → Real-Time and selects `prod-pg-01`
**THEN** the overview table displays live executing sessions refreshing every 1–5 seconds

### Scenario: Lock chain visualization

**GIVEN** a blocked session with a lock chain showing PID 100 blocking PID 200
**WHEN** the user views the Real-Time overview for that service
**THEN** a lock chain panel displays PID 100 → PID 200 with query text and duration for each link

### Scenario: Idle-in-transaction badge

**GIVEN** a session in `idle in transaction` state that started a transaction 45 seconds ago
**WHEN** the session appears in the Real-Time table
**THEN** the row is visually marked as idle-in-transaction and shows transaction duration of ~45 seconds (not query duration)

### Scenario: Parallel workers collapsed

**GIVEN** a parallel query with leader PID 300 and two worker PIDs 301, 302
**WHEN** the user views the Real-Time table
**THEN** workers 301 and 302 are collapsed under leader PID 300 by default

### Scenario: Query truncation tooltip

**GIVEN** an active query whose text exceeds `track_activity_query_size`
**WHEN** the user hovers over the truncated query text
**THEN** a tooltip explains that truncation is due to `track_activity_query_size` and increasing it requires a server restart

### Scenario: Permission error displayed

**GIVEN** an RTA session where the monitoring user lacks `pg_read_all_stats`
**WHEN** the user opens the Real-Time tab for that service
**THEN** an error message explains the missing privilege and how to grant it (not a blank screen)

### Scenario: MongoDB RTA still works

**GIVEN** an active MongoDB RTA session
**WHEN** PostgreSQL RTA is available in the same PMM instance
**THEN** selecting a MongoDB service in Real-Time shows MongoDB sessions exactly as before
