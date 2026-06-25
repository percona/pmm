# PostgreSQL RTA Server & API

Capability: PMM Server registers PostgreSQL RTA agents, accepts streamed session data, and exposes it via RealtimeAnalyticsService.

## Requirements

### REQ-PG-SRV-001: Inventory agent type

PMM SHALL support `rta-postgresql-agent` in the Inventory API with add, change, list, and get operations mirroring `rta-mongodb-agent` (PostgreSQL-appropriate connection fields).

### REQ-PG-SRV-002: Service type mapping

`RealtimeAnalyticsService.ListServices` with `service_type=POSTGRESQL` SHALL return PostgreSQL services eligible for RTA (registered, with running pmm-agent, not already in an active session unless listed as running).

### REQ-PG-SRV-003: Session lifecycle

Starting and stopping RTA sessions for PostgreSQL services SHALL use the same REST/gRPC endpoints as MongoDB (`/v1/realtimeanalytics/sessions:start`, `sessions:stop`, `sessions`, `queries:search`).

### REQ-PG-SRV-004: PostgreSQL query payload

`SearchQueries` responses for PostgreSQL services SHALL include `QueryData.postgres_payload` with session state, wait events, lock chain, and PostgreSQL-specific timestamps.

### REQ-PG-SRV-005: In-memory store

PostgreSQL RTA data SHALL be stored in the existing in-memory TTL store with the same retention semantics as MongoDB RTA (no persistence to ClickHouse).

### REQ-PG-SRV-006: MongoDB compatibility

Existing MongoDB RTA behavior, API contracts, and stored MongoDB payloads SHALL remain unchanged.

### REQ-PG-SRV-007: QAN independence

PostgreSQL RTA SHALL NOT write to ClickHouse or interfere with QAN agents (`pg_stat_statements`, `pg_stat_monitor`) on the same service.

### REQ-PG-SRV-008: Telemetry

PMM SHALL emit telemetry metrics for PostgreSQL RTA agents: total registered, currently enabled (running session), and disabled counts.

### REQ-PG-SRV-009: Access control

Starting and stopping PostgreSQL RTA sessions SHALL require Admin role; viewing live data from running sessions SHALL follow the same role rules as MongoDB RTA.

## Scenarios

### Scenario: List PostgreSQL RTA services

**GIVEN** two registered PostgreSQL services and one without pmm-agent  
**WHEN** a user calls `ListServices` with `service_type=POSTGRESQL`  
**THEN** only services with a eligible pmm-agent and no conflicting active session are returned

### Scenario: Start PostgreSQL session

**GIVEN** an Admin user and a registered PostgreSQL service with RTA agent configured  
**WHEN** the user calls `StartSession` with the service ID  
**THEN** the RTA agent is enabled, session status is `RUNNING`, and collect interval is applied from agent RTAOptions

### Scenario: Search queries returns PostgreSQL payload

**GIVEN** a running PostgreSQL RTA session with active queries  
**WHEN** the user calls `SearchQueries` with the service ID  
**THEN** the response contains `QueryData` entries with populated `postgres_payload` including lock chain when contention exists

### Scenario: Stop session disables agent

**GIVEN** a running PostgreSQL RTA session  
**WHEN** an Admin calls `StopSession`  
**THEN** the session status becomes `DOWN`, the agent is disabled, and in-memory queries for that service are evicted per existing TTL rules

### Scenario: MongoDB session unaffected

**GIVEN** an active MongoDB RTA session  
**WHEN** a PostgreSQL RTA session is started on a different service  
**THEN** MongoDB `SearchQueries` responses continue to return `mongo_db_payload` without schema or behavior changes

### Scenario: Concurrent QAN collection

**GIVEN** a PostgreSQL service with both QAN (pg_stat_monitor) and RTA agents enabled  
**WHEN** both agents run for 10 minutes  
**THEN** QAN continues to receive completed query metrics in ClickHouse and RTA streams live sessions without duplicate or corrupted data
