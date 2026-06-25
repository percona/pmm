# PostgreSQL RTA Server & API

Capability spec for pmm-managed inventory, RealtimeAnalyticsService, and API extensions for PostgreSQL RTA.

## Requirements

### REQ-PG-SRV-001: Inventory agent type

The Inventory API SHALL support `RTAPostgreSQLAgent` with add, change, list, and get operations parallel to `RTAMongoDBAgent`.

### REQ-PG-SRV-002: Service type mapping

`RealtimeAnalyticsService.ListServices` SHALL include PostgreSQL services when filtered by `SERVICE_TYPE_POSTGRESQL` or when no filter is specified.

### REQ-PG-SRV-003: Session lifecycle

Users SHALL be able to start, stop, and list RTA sessions for PostgreSQL services via the existing RealtimeAnalytics REST/gRPC API (`StartSession`, `StopSession`, `ListSessions`).

### REQ-PG-SRV-004: Data ingestion

`CollectorService.Collect` SHALL accept `QueryData` records with `QueryPostgreSQLData` payloads and store them in the in-memory TTL store.

### REQ-PG-SRV-005: Search queries

`SearchQueries` SHALL return PostgreSQL session data with the `postgresql_payload` field populated for active sessions.

### REQ-PG-SRV-006: Version gate

The server SHALL reject RTA session start for PostgreSQL services when the linked pmm-agent version does not support PostgreSQL RTA.

### REQ-PG-SRV-007: MongoDB unchanged

All existing MongoDB RTA API behavior SHALL remain unchanged after PostgreSQL RTA is added.

### REQ-PG-SRV-008: Telemetry

The server SHALL expose telemetry metrics for PostgreSQL RTA agents (registered count, enabled count, disabled count) mirroring MongoDB RTA metrics.

## Scenarios

### Scenario: Start RTA session for PostgreSQL service

**GIVEN** a registered PostgreSQL service with a compatible pmm-agent
**WHEN** a user calls `StartSession` with the service ID
**THEN** an `rta-postgresql-agent` is created (or enabled) and the response includes session status `RUNNING` with the configured collect interval

### Scenario: List PostgreSQL in available services

**GIVEN** two registered services: one MongoDB and one PostgreSQL, both with RTA-capable agents
**WHEN** a user calls `ListServices` without a type filter
**THEN** both services appear in the response

### Scenario: Search returns PostgreSQL sessions

**GIVEN** an active RTA session on a PostgreSQL service with 3 executing queries
**WHEN** a user calls `SearchQueries` for that service
**THEN** the response contains 3 `QueryData` entries each with `postgresql_payload` populated

### Scenario: Outdated pmm-agent rejected

**GIVEN** a PostgreSQL service linked to a pmm-agent version that predates PostgreSQL RTA support
**WHEN** a user calls `StartSession`
**THEN** the API returns an error indicating the agent version is too old

### Scenario: MongoDB RTA unaffected

**GIVEN** an active MongoDB RTA session
**WHEN** PostgreSQL RTA is deployed and a PostgreSQL session is started
**THEN** the MongoDB session continues collecting and returning data without change
