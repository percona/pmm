# Tasks

## Phase 1 — PostgreSQL RTA Agent

- [ ] 1.1 Add `QueryPostgreSQLData` message to `api/realtimeanalytics/v1/query.proto`; extend `QueryData.payload` oneof; run `make -C api gen`
- [ ] 1.2 Add `RTAPostgreSQLAgent` to `api/inventory/v1/agents.proto`; run `make -C api gen`
- [ ] 1.3 Add `RTAPostgreSQLAgentType` to `managed/models/agent_model.go` and related helpers
- [ ] 1.4 Implement `agent/agents/postgresql/realtimeanalytics/` collector (pg_stat_activity + pg_locks polling, lock chains, idle-in-transaction, parallel workers)
- [ ] 1.5 Wire PostgreSQL RTA agent into supervisor and pmm-agent client channel
- [ ] 1.6 Unit tests for parser/collector (lock chain resolution, query_id vs fingerprint, idle-in-transaction)

## Phase 2 — Server-Side & API

- [ ] 2.1 Extend `RealtimeAnalyticsService.ListServices` to return PostgreSQL services
- [ ] 2.2 Extend `getRTAAgentTypeForServiceType` and session/query handlers for PostgreSQL
- [ ] 2.3 Implement `AddRTAPostgreSQLAgent` / `ChangeRTAPostgreSQLAgent` in inventory service
- [ ] 2.4 Wire agent state provisioning in `managed/services/agents/state.go`
- [ ] 2.5 Add `version.PostgreSQLRtaAgentSupportVersion` feature gate
- [ ] 2.6 Handle missing-permission errors with actionable session status messages

## Phase 3 — UI Integration

- [ ] 3.1 Extend `rta.types.ts` and `useRealtime.ts` for PostgreSQL payload and service list
- [ ] 3.2 Update Real-Time overview table with PostgreSQL-specific columns and indicators
- [ ] 3.3 Add lock chain panel and idle-in-transaction badge in details pane
- [ ] 3.4 Collapse parallel workers under leader (PG 13+); query truncation tooltip
- [ ] 3.5 Include PostgreSQL services in service selector; verify MongoDB unchanged
- [ ] 3.6 Run `make -C ui lint`

## Phase 4 — Testing, Hardening & GA

- [ ] 4.1 Dev smoke test: provision PostgreSQL, start RTA session, verify live sessions via API and UI
- [ ] 4.2 Validate on PostgreSQL 12, 14, 15, 16 and Percona Distribution for PostgreSQL
- [ ] 4.3 Validate cloud variants (RDS, Aurora, Cloud SQL, Azure) with documented permissions
- [ ] 4.4 Performance check: <1% CPU overhead at 2s interval with ≤200 backends
- [ ] 4.5 Update `documentation/docs/use/qan/QAN-realtime-analytics.md` with PostgreSQL section and version matrix
- [ ] 4.6 Enable RTA by default for registered PostgreSQL services
- [ ] 4.7 Open `Percona-Lab/pmm-submodules` FB PR after percona/pmm PR is ready
