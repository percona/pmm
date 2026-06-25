# Design

## Approach

Extend the existing MongoDB RTA architecture to PostgreSQL following the same data path:

```
pmm-agent (RTAPostgreSQLAgent)
  → polls pg_stat_activity + pg_locks every CollectInterval (1–5s)
  → builds QueryData with QueryPostgreSQLData payload
  → Supervisor RTARequests channel
  → pmm-agent client gRPC stream (CollectorService.Collect)
  → pmm-managed RealtimeAnalyticsService Store (in-memory TTL)
  → REST/gRPC SearchQueries API
  → PMM UI Real-Time tab
```

The implementation mirrors MongoDB RTA patterns already in the codebase:

| Layer | MongoDB (existing) | PostgreSQL (new) |
|-------|-------------------|------------------|
| Agent type | `rta-mongodb-agent` | `rta-postgresql-agent` |
| Agent package | `agent/agents/mongodb/realtimeanalytics` | `agent/agents/postgresql/realtimeanalytics` |
| Inventory proto | `RTAMongoDBAgent` | `RTAPostgreSQLAgent` |
| Query payload | `QueryMongoDBData` | `QueryPostgreSQLData` |
| Service type mapping | `MongoDBServiceType` → `RTAMongoDBAgentType` | `PostgreSQLServiceType` → `RTAPostgreSQLAgentType` |
| Supervisor case | `AGENT_TYPE_RTA_MONGODB_AGENT` | `AGENT_TYPE_RTA_POSTGRESQL_AGENT` |

### Agent collection

The PostgreSQL RTA agent connects using the same credentials as the existing PostgreSQL exporter / QAN agent for the service. On each tick:

1. Query `pg_stat_activity` for active sessions (filter out PMM's own connections).
2. Query `pg_locks` joined with `pg_stat_activity` to build lock chains (blocker PID → blocked PID).
3. Map PostgreSQL fields to `QueryPostgreSQLData`: `state`, `wait_event_type`, `wait_event`, `backend_type`, `query_id` (PG 14+), `xact_start`, `query_start`, `state_change`, leader/worker PID (PG 13+).
4. Compute `query_id` from `pg_stat_activity.query_id` on PG 14+; fall back to query fingerprint on PG 12–13.
5. Emit `QueryData` records via the standard RTA collect channel.

Default collect interval: 2 seconds (configurable via agent `RTAOptions.CollectInterval`, same as MongoDB).

### Server-side

- Extend `getRTAAgentTypeForServiceType()` in `managed/services/realtimeanalytics/service.go` to map `PostgreSQLServiceType`.
- Extend `isRtaFeatureSupported()` with a new version gate for PostgreSQL RTA agent support.
- Add inventory CRUD handlers in `managed/services/inventory/agents.go` (Add/Change RTAPostgreSQLAgent).
- Extend `managed/models/agent_model.go` with `RTAPostgreSQLAgentType`.
- Extend agent state builder in `managed/services/agents/state.go`.
- Store accepts PostgreSQL payloads in the existing in-memory TTL store — no ClickHouse persistence.

### API

Extend `api/realtimeanalytics/v1/query.proto`:

```protobuf
message QueryPostgreSQLData {
  int32 pid = 1;
  string state = 2;
  string wait_event_type = 3;
  string wait_event = 4;
  string backend_type = 5;
  google.protobuf.Timestamp transaction_start_time = 6;
  google.protobuf.Timestamp state_change_time = 7;
  int32 leader_pid = 8;
  repeated LockChainEntry lock_chain = 9;
  string database_name = 10;
  string username = 11 [(extensions.v1.sensitive) = REDACT_TYPE_FULL];
  string application_name = 12;
}

message LockChainEntry {
  int32 pid = 1;
  string lock_mode = 2;
  string lock_type = 3;
  bool granted = 4;
  string query_text = 5;
  google.protobuf.Duration duration = 6;
}
```

Add `QueryPostgreSQLData postgresql_payload = 10;` to the `QueryData.payload` oneof.

Extend `api/inventory/v1/agents.proto` with `RTAPostgreSQLAgent` message and add/change params (parallel to `RTAMongoDBAgent`).

### UI

Extend existing RTA pages under `ui/apps/pmm/src/pages/rta/`:

- Service selector already calls `listServices` — will include PostgreSQL once server returns them.
- Overview table: add PostgreSQL-specific columns (state, wait event, lock indicator).
- Lock chain panel: new component showing blocker → blocked visualization.
- Idle-in-transaction badge with transaction duration (not query duration).
- Parallel worker rows collapsed under leader (PG 13+).
- Query truncation tooltip referencing `track_activity_query_size`.
- Permission error state when agent reports insufficient privileges.

### Permissions

Monitoring user requires `pg_read_all_stats` (or superuser) to read all sessions in `pg_stat_activity`. Without it, the agent reports a specific error surfaced in the UI — not a blank screen.

Cloud variants (RDS, Aurora, Cloud SQL, Azure) use standard `pg_stat_activity` / `pg_locks`; no Aurora-specific views in this epic.

## Files / areas touched

| Area | Key files |
|------|-----------|
| Protobuf | `api/realtimeanalytics/v1/query.proto`, `api/inventory/v1/agents.proto` |
| pmm-agent | `agent/agents/postgresql/realtimeanalytics/`, `agent/agents/supervisor/supervisor.go` |
| pmm-managed | `managed/services/realtimeanalytics/service.go`, `managed/services/inventory/agents.go`, `managed/models/agent_model.go`, `managed/services/agents/state.go`, `managed/services/converters.go` |
| Version gate | `version/feature.go` |
| Telemetry | `managed/services/telemetry/config.default.yml` |
| UI | `ui/apps/pmm/src/pages/rta/`, `ui/apps/pmm/src/api/rta/` |
| Docs | `documentation/` (user-facing RTA PostgreSQL section) |
| Tests | Agent unit tests, managed service tests, UI component tests, `api-tests/` integration |

## Risks

| Risk | Mitigation |
|------|------------|
| High poll frequency increases PostgreSQL CPU | Default 2s interval; configurable; target <1% CPU overhead at ≤200 backends |
| `pg_stat_activity` column differences across PG versions | Version-aware queries; test matrix PG 12–16 + Percona Distribution |
| Lock chain query cost on busy systems | Single batched query per tick; limit chain depth |
| Cloud provider permission restrictions | Document required grants; surface actionable permission errors |
| MongoDB RTA regression | Existing MongoDB RTA tests must pass unchanged; no shared code path modifications beyond generic RTA plumbing |
| Query text truncation confusion | Tooltip explaining `track_activity_query_size` and restart requirement |
