# Tasks

## Phase 1 — PostgreSQL RTA Agent

- [ ] 1.1 Add `QueryPostgreSQLData` and `LockChainEntry` messages to `api/realtimeanalytics/v1/query.proto`; regenerate API stubs
- [ ] 1.2 Add `RTAPostgreSQLAgent` to `api/inventory/v1/agents.proto`; regenerate API stubs
- [ ] 1.3 Add `RTAPostgreSQLAgentType` to `managed/models/agent_model.go` with DSN helpers and agent filters
- [ ] 1.4 Implement `agent/agents/postgresql/realtimeanalytics/` — connection, `pg_stat_activity` / `pg_locks` polling, lock chain builder
- [ ] 1.5 Wire `AGENT_TYPE_RTA_POSTGRESQL_AGENT` in supervisor; emit RTA collect requests
- [ ] 1.6 Add version feature flag for PostgreSQL RTA agent support in `version/feature.go`
- [ ] 1.7 Unit tests for PostgreSQL RTA agent (session parsing, lock chains, version-specific fields)

## Phase 2 — Server-Side & API Extension

- [ ] 2.1 Extend `getRTAAgentTypeForServiceType()` and `isRtaFeatureSupported()` for PostgreSQL
- [ ] 2.2 Add `AddRTAPostgreSQLAgent` / `ChangeRTAPostgreSQLAgent` in inventory service
- [ ] 2.3 Extend agent state builder, converters, and gRPC agents server for PostgreSQL RTA
- [ ] 2.4 Extend RealtimeAnalyticsService session start/stop/list for PostgreSQL services
- [ ] 2.5 Ensure in-memory store handles PostgreSQL QueryData payloads
- [ ] 2.6 Add PostgreSQL RTA telemetry metrics (mirroring MongoDB RTA counters)
- [ ] 2.7 Unit tests for managed service PostgreSQL RTA paths

## Phase 3 — UI Integration

- [ ] 3.1 Extend RTA service selector to show PostgreSQL services
- [ ] 3.2 Add PostgreSQL-specific columns to Real-Time overview table (state, wait event, locks)
- [ ] 3.3 Implement lock chain panel (blocker → blocked with query text and duration)
- [ ] 3.4 Add idle-in-transaction visual distinction with transaction duration
- [ ] 3.5 Collapse parallel worker sessions under leader (PG 13+)
- [ ] 3.6 Add query truncation tooltip for `track_activity_query_size`
- [ ] 3.7 Add permission error state for missing `pg_read_all_stats`
- [ ] 3.8 UI unit tests for PostgreSQL RTA components

## Phase 4 — Testing, Hardening & GA

- [ ] 4.1 Integration tests on PostgreSQL 12, 14, 15, 16 and Percona Distribution for PostgreSQL
- [ ] 4.2 Verify cloud variant compatibility (RDS, Aurora, Cloud SQL, Azure) with documented permission setup
- [ ] 4.3 Performance validation: <1% CPU overhead at 2s interval with ≤200 backends
- [ ] 4.4 Regression check: MongoDB RTA and PostgreSQL QAN unchanged
- [ ] 4.5 User documentation with version matrix, permissions, and known limitations
- [ ] 4.6 QA automation in `pmm-qa` for PostgreSQL RTA end-to-end flows
