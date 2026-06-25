# PostgreSQL RTA UI

Capability: Query Analytics → Real-time tab displays live PostgreSQL sessions with PostgreSQL-specific diagnostics.

## Requirements

### REQ-PG-UI-001: Service selection

The Real-time tab service selector SHALL list registered PostgreSQL services alongside MongoDB services. Users SHALL start RTA sessions for PostgreSQL without navigating to a new page.

### REQ-PG-UI-002: Live refresh

The overview table SHALL refresh at the user-selected auto-refresh interval (1–5 seconds, default 2 seconds) with end-to-end latency ≤5 seconds.

### REQ-PG-UI-003: Overview columns

For PostgreSQL sessions, the overview table SHALL display at minimum: session state, database, username/application, wait event (type + name), elapsed duration, and query text.

### REQ-PG-UI-004: Lock chain panel

The Details pane SHALL include a lock chain visualization showing blocker → blocked relationships with query text and duration for each link in the chain.

### REQ-PG-UI-005: Idle in transaction

Sessions in state `idle in transaction` SHALL be visually distinguished (badge or row styling) and SHALL display transaction duration derived from `xact_start`, not query duration.

### REQ-PG-UI-006: Parallel workers

On PostgreSQL 13+, parallel worker sessions (where `leader_pid` is set) SHALL be collapsed under their leader row by default with an expand control to show workers.

### REQ-PG-UI-007: Truncated query indicator

When `query_text_truncated` is true, the UI SHALL indicate truncation (icon or label) and show a tooltip explaining `track_activity_query_size` and that increasing it requires a PostgreSQL server restart.

### REQ-PG-UI-008: Permission error

When the agent reports missing `pg_read_all_stats`, the UI SHALL display a specific actionable error message instead of an empty table or generic failure.

### REQ-PG-UI-009: Raw data tab

The Raw data tab SHALL show the full JSON snapshot from the agent (pg_stat_activity and lock fields) for troubleshooting.

### REQ-PG-UI-010: MongoDB parity features

PostgreSQL RTA SHALL support the same pause/resume, share link, multi-session, and session management flows as MongoDB RTA.

### REQ-PG-UI-011: No regression

Existing MongoDB RTA UI behavior and styling SHALL remain unchanged when no PostgreSQL services are present.

## Scenarios

### Scenario: Start PostgreSQL RTA from Real-time tab

**GIVEN** an Admin on Query Analytics → Real-time with no active sessions  
**WHEN** they select a PostgreSQL service and click Start session  
**THEN** the live operations table appears and begins auto-refreshing with PostgreSQL session rows

### Scenario: View lock chain in Details

**GIVEN** a PostgreSQL session blocked by another backend  
**WHEN** the user clicks the blocked session row and opens Details  
**THEN** the lock chain panel shows the blocker PID, blocker query text, blocker duration, and lock mode

### Scenario: Idle in transaction badge

**GIVEN** a session in state `idle in transaction` running for 45 seconds  
**WHEN** displayed in the overview table  
**THEN** the row shows an idle-in-transaction indicator and duration of 45s based on transaction start time

### Scenario: Expand parallel workers

**GIVEN** a parallel query leader with two worker backends  
**WHEN** the overview table renders  
**THEN** workers are hidden under the collapsed leader row; clicking expand reveals worker rows with their PIDs and wait events

### Scenario: Truncated query tooltip

**GIVEN** a query with `query_text_truncated=true` and `track_activity_query_size=1024`  
**WHEN** the user hovers the truncation indicator  
**THEN** a tooltip explains the query was truncated at 1024 bytes and increasing `track_activity_query_size` requires a server restart

### Scenario: Permission error displayed

**GIVEN** the RTA agent reports insufficient privileges for pg_stat_activity  
**WHEN** the Real-time page loads  
**THEN** an error banner explains that `pg_read_all_stats` (or superuser) is required with a link to documentation

### Scenario: Pause preserves PostgreSQL view

**GIVEN** an active PostgreSQL RTA session with auto-refresh enabled  
**WHEN** the user clicks Pause  
**THEN** the table freezes on the current snapshot while the agent continues collecting in the background

### Scenario: Share link with PostgreSQL filter

**GIVEN** a user viewing PostgreSQL service X in RTA overview  
**WHEN** they click Share  
**THEN** the copied URL includes the service filter and opens Real-time with PostgreSQL service X selected
