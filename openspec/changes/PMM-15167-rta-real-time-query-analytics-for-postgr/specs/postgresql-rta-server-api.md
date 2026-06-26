# PostgreSQL RTA Server & API

## Requirement: Inventory agent type for PostgreSQL RTA

The Inventory API MUST support `RTAPostgreSQLAgent` with add, change, get, and list operations mirroring the existing `RTAMongoDBAgent` lifecycle.

### Scenario: Add PostgreSQL RTA agent via inventory API

- **GIVEN** a registered PostgreSQL service and connected pmm-agent
- **WHEN** an Admin calls `AddAgent` with type `RTAPostgreSQLAgent` for that service
- **THEN** the agent is created in inventory and pmm-agent receives configuration to start the collector

## Requirement: ListServices includes PostgreSQL

`RealtimeAnalyticsService.ListServices` MUST return PostgreSQL services when filtered by `SERVICE_TYPE_POSTGRESQL`.

### Scenario: List PostgreSQL RTA-capable services

- **GIVEN** two PostgreSQL services and one MongoDB service registered in PMM
- **WHEN** `GET /v1/realtimeanalytics/services?service_type=SERVICE_TYPE_POSTGRESQL` is called
- **THEN** the response contains both PostgreSQL services and no MongoDB services

## Requirement: Session lifecycle for PostgreSQL

Start and stop session endpoints MUST work for PostgreSQL services using the same REST/gRPC API as MongoDB.

### Scenario: Start RTA session for PostgreSQL

- **GIVEN** a registered PostgreSQL service with no active RTA session
- **WHEN** an Admin calls `POST /v1/realtimeanalytics/sessions:start` with the service ID
- **THEN** a session is created with status `SESSION_STATUS_RUNNING` and the RTA agent begins collecting

### Scenario: Stop RTA session for PostgreSQL

- **GIVEN** an active RTA session for a PostgreSQL service
- **WHEN** an Admin calls `POST /v1/realtimeanalytics/sessions:stop` with the service ID
- **THEN** the session stops and the agent status becomes `AGENT_STATUS_DONE`

## Requirement: SearchQueries returns PostgreSQL payloads

`SearchQueries` MUST return `QueryData` records with `postgresql_payload` populated for active PostgreSQL sessions.

### Scenario: Search live PostgreSQL sessions

- **GIVEN** an active RTA session with executing queries on a PostgreSQL service
- **WHEN** `POST /v1/realtimeanalytics/queries:search` is called with the service ID
- **THEN** the response contains queries with `postgresql_payload` including state, wait event, database, and lock chain data

## Requirement: In-memory TTL store for PostgreSQL queries

The server MUST store PostgreSQL RTA query data in the existing in-memory TTL store with the same retention semantics as MongoDB RTA (no persistence to ClickHouse).

### Scenario: Stale queries expire from store

- **GIVEN** a query that was active in the previous poll but has since completed
- **WHEN** the next `SearchQueries` call occurs after TTL expiry
- **THEN** the completed query no longer appears in the response

## Requirement: MongoDB RTA unchanged

All existing MongoDB RTA server behavior MUST remain unchanged after PostgreSQL support is added.

### Scenario: MongoDB RTA regression check

- **GIVEN** an active MongoDB RTA session before and after PostgreSQL RTA deployment
- **WHEN** `SearchQueries` is called for the MongoDB service
- **THEN** responses contain `mongo_db_payload` with the same fields and refresh behavior as before

## Requirement: Actionable permission errors in session status

When the PostgreSQL RTA agent reports a permission error, the session status MUST reflect `SESSION_STATUS_ERROR` with a human-readable message.

### Scenario: Session shows permission error

- **GIVEN** a PostgreSQL RTA agent that failed to start due to missing `pg_read_all_stats`
- **WHEN** `GET /v1/realtimeanalytics/sessions` is called
- **THEN** the session status is `SESSION_STATUS_ERROR` and the error message mentions `pg_read_all_stats`

## Requirement: Feature version gate

The server MUST reject RTA session start for PostgreSQL when the connected pmm-agent version does not support PostgreSQL RTA.

### Scenario: Old pmm-agent version

- **GIVEN** a pmm-agent version older than the PostgreSQL RTA support version
- **WHEN** an Admin attempts to start an RTA session for a PostgreSQL service
- **THEN** the request fails with an error indicating the agent must be upgraded
