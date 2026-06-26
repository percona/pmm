# PostgreSQL RTA UI

## Requirement: PostgreSQL services in Real-Time tab

The Query Analytics → Real-Time tab MUST allow selecting any registered PostgreSQL service to start an RTA session, with no new navigation required.

### Scenario: Start session for PostgreSQL service

- **GIVEN** a registered PostgreSQL service and an Admin user
- **WHEN** the user navigates to Query Analytics → Real-Time, selects the PostgreSQL service, and clicks Start session
- **THEN** the live operations table appears and begins auto-refreshing

## Requirement: Live session refresh latency

The Real-Time view MUST display executing PostgreSQL sessions with ≤5 second refresh latency.

### Scenario: Session data refreshes within 5 seconds

- **GIVEN** an active RTA session for a PostgreSQL service with auto-refresh set to 2 seconds
- **WHEN** a new long-running query starts on the database
- **THEN** the query appears in the table within 5 seconds

## Requirement: PostgreSQL-specific table columns

The overview table MUST display PostgreSQL-relevant columns including session state, wait event, and database name.

### Scenario: Wait event visible for waiting session

- **GIVEN** a PostgreSQL session waiting on a lock
- **WHEN** the Real-Time table refreshes
- **THEN** the row shows the wait event type/class in the table

## Requirement: Lock chain panel in details pane

The details pane MUST show a lock chain panel with blocker → blocked relationships, query text, and duration for each link.

### Scenario: Lock chain displayed in details

- **GIVEN** a blocked PostgreSQL session visible in the Real-Time table
- **WHEN** the user clicks the row to open the details pane
- **THEN** the lock chain panel shows the blocker session with its query text and duration

## Requirement: Idle-in-transaction visual distinction

Sessions in `idle in transaction` state MUST be visually distinguished and show transaction duration (not query duration).

### Scenario: Idle-in-transaction badge

- **GIVEN** a session in `idle in transaction` state
- **WHEN** displayed in the Real-Time table or details pane
- **THEN** an idle-in-transaction indicator is shown with transaction duration

## Requirement: Parallel worker collapse

On PostgreSQL 13+, parallel worker sessions MUST be collapsed under their leader by default.

### Scenario: Workers grouped under leader

- **GIVEN** a parallel query with a leader and two workers on PostgreSQL 14
- **WHEN** the Real-Time table displays the sessions
- **THEN** workers appear collapsed under the leader row and can be expanded

## Requirement: Query text truncation indicator

When query text is truncated due to `track_activity_query_size`, the UI MUST indicate truncation with a tooltip explaining the setting and server-restart requirement.

### Scenario: Truncated query shows tooltip

- **GIVEN** a query whose text exceeds `track_activity_query_size`
- **WHEN** displayed in the Real-Time table
- **THEN** a truncation indicator is shown and hovering reveals a tooltip mentioning `track_activity_query_size`

## Requirement: Permission error display

When the monitoring user lacks required privileges, the UI MUST show a specific actionable error instead of a blank screen.

### Scenario: Missing pg_read_all_stats error in UI

- **GIVEN** a PostgreSQL RTA session that failed due to missing `pg_read_all_stats`
- **WHEN** the user views the Real-Time tab for that service
- **THEN** an error message explains that `pg_read_all_stats` must be granted to the monitoring user

## Requirement: Pause and resume

The existing pause/resume controls MUST work for PostgreSQL RTA sessions identically to MongoDB.

### Scenario: Pause freezes PostgreSQL view

- **GIVEN** an active PostgreSQL RTA session with live data refreshing
- **WHEN** the user clicks Pause
- **THEN** the table stops updating while the agent continues collecting in the background

## Requirement: MongoDB RTA UI unchanged

The MongoDB Real-Time experience MUST remain unchanged after PostgreSQL support is added.

### Scenario: MongoDB session still works

- **GIVEN** an active MongoDB RTA session
- **WHEN** the user views the Real-Time tab with a MongoDB service selected
- **THEN** MongoDB-specific columns (operation, collection, plan summary) display as before

## Requirement: Raw diagnostics panel

The Raw data tab in the details pane MUST show the full JSON payload from the PostgreSQL RTA agent for diagnostic purposes.

### Scenario: Raw JSON available for PostgreSQL session

- **GIVEN** an active PostgreSQL session in the Real-Time table
- **WHEN** the user opens the details pane and selects the Raw data tab
- **THEN** the full `queryRawJson` payload is displayed
