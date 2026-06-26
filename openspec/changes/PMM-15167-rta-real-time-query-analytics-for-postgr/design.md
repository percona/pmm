# Design

## Approach

Extend the proven MongoDB RTA architecture to PostgreSQL. The data path mirrors the existing pipeline:

```
pmm-agent (RTAPostgreSQLAgent)
  → polls pg_stat_activity + pg_locks every collect_interval (1–5s)
  → builds rtav1.QueryData with QueryPostgreSQLData payload
  → RTAChannel (gRPC client stream)
    → pmm-managed RealtimeAnalyticsService (CollectorService.Collect)
      → in-memory TTL Store
        → REST/gRPC SearchQueries, ListServices, session management
          → PMM UI Real-Time tab
```

### Phase 1 — PostgreSQL RTA Agent (`agent/`)

- New package: `agent/agents/postgresql/realtimeanalytics/` following the MongoDB agent pattern (`mongodb/realtimeanalytics/mongodb.go`).
- Poll query joins `pg_stat_activity` with `pg_locks` to produce session records including:
  - Session state, wait event type/class, query text, backend type
  - Lock chain (blocker → blocked) with query text and duration per link
  - Idle-in-transaction detection with transaction duration (distinct from query duration)
  - Parallel worker grouping under leader (PostgreSQL 13+)
  - `query_id` on PG 14+; fingerprint fallback on PG 12–13
- Reuse existing PostgreSQL connection helpers from QAN agents where applicable.
- Register agent in supervisor (`agent/agents/supervisor/supervisor.go`) and wire through `agent/client`.
- Required PostgreSQL privileges: `pg_read_all_stats` (or superuser). Surface actionable error when missing.

### Phase 2 — Server-Side & API (`managed/`, `api/`)

- **Protobuf** (`api/realtimeanalytics/v1/`):
  - Add `QueryPostgreSQLData` message with PostgreSQL-specific fields (state, wait_event, lock_chain, backend_type, database, user, transaction_duration, is_idle_in_transaction, leader_pid, query_truncated).
  - Extend `QueryData.payload` oneof with `postgresql_payload`.
  - Extend `ListServicesResponse` with `repeated inventory.v1.PostgreSQLService postgresql`.
- **Inventory API** (`api/inventory/v1/agents.proto`):
  - Add `RTAPostgreSQLAgent` agent type, add/change/get/list handlers mirroring `RTAMongoDBAgent`.
- **Models** (`managed/models/`):
  - Add `RTAPostgreSQLAgentType` constant; extend agent model helpers and DSN resolution.
- **RealtimeAnalyticsService** (`managed/services/realtimeanalytics/service.go`):
  - Extend `getRTAAgentTypeForServiceType` for `PostgreSQLServiceType`.
  - Extend `ListServices`, session lifecycle, and query search for PostgreSQL payloads.
  - Add feature flag / version gate (`version.PostgreSQLRtaAgentSupportVersion`).
- **Inventory & agent state** (`managed/services/inventory/`, `managed/services/agents/`):
  - `AddRTAPostgreSQLAgent`, `ChangeRTAPostgreSQLAgent`, state conversion, auto-provisioning on PostgreSQL service add (Phase 4).

### Phase 3 — UI Integration (`ui/`)

- Extend `rta.types.ts` with `QueryPostgreSQLData` and `postgresql` in `AvailableServicesResponse`.
- Update Real-Time overview table columns for PostgreSQL: state, wait event, database, lock indicator.
- Details pane: lock chain panel, idle-in-transaction badge with transaction duration, parallel worker collapse.
- Query text truncation tooltip explaining `track_activity_query_size` and restart requirement.
- Service selector: include PostgreSQL services alongside MongoDB.
- MongoDB RTA behavior unchanged.

### Phase 4 — Testing, Hardening & GA

- Integration tests across PostgreSQL 12, 14, 15, 16, Percona Distribution for PostgreSQL.
- Cloud variant validation (RDS, Aurora, Cloud SQL, Azure) with documented permission setups.
- Performance validation: <1% additional CPU at 2s polling with ≤200 active backends.
- Update user docs (`documentation/docs/use/qan/QAN-realtime-analytics.md`).
- Enable by default for all registered PostgreSQL services.

## Files / areas touched

| Area | Key paths |
|------|-----------|
| Agent collector | `agent/agents/postgresql/realtimeanalytics/`, `agent/agents/supervisor/supervisor.go`, `agent/client/` |
| Protobuf / API | `api/realtimeanalytics/v1/query.proto`, `api/realtimeanalytics/v1/realtimeanalytics.proto`, `api/inventory/v1/agents.proto` |
| Server backend | `managed/services/realtimeanalytics/`, `managed/services/inventory/`, `managed/services/agents/`, `managed/models/` |
| UI | `ui/apps/pmm/src/pages/rta/`, `ui/apps/pmm/src/types/rta.types.ts`, `ui/apps/pmm/src/hooks/api/useRealtime.ts` |
| Version / feature flags | `version/features.go` |
| Documentation | `documentation/docs/use/qan/QAN-realtime-analytics.md` |

## Risks

| Risk | Mitigation |
|------|------------|
| `pg_stat_activity` polling overhead on busy instances | Configurable 1–5s interval (default 2s); validate <1% CPU target; reuse connection pool |
| Missing `pg_read_all_stats` on cloud RDS/Azure | Detect at agent start; return specific actionable error in session status |
| Query text truncation (`track_activity_query_size`) | Visual indicator + tooltip; document server-restart requirement |
| `query_id` unavailable on PG 12–13 | Fingerprint fallback; document in version matrix |
| RTA/QAN conflict on same instance | Independent agents; RTA reads `pg_stat_activity`, QAN reads `pg_stat_statements`/`pg_stat_monitor` — no shared state |
| MongoDB RTA regression | Existing MongoDB code paths unchanged; shared service layer extended via type switch |
| Protobuf regeneration scope | Run `make -C api gen` only for edited `.proto` files — never `make gen` at repo root |
