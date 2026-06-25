# Design: PostgreSQL Real-Time Query Analytics

## Approach

Extend the existing MongoDB RTA architecture to PostgreSQL by adding a parallel agent type and protobuf payload while reusing server-side session management, in-memory store, and UI shell. The implementation follows established PMM patterns: inventory agent registration → pmm-agent polling loop → gRPC collector stream → `managed/services/realtimeanalytics` store → REST API → React UI.

### Data collection (pmm-agent)

The PostgreSQL RTA agent runs inside pmm-agent and connects using the same DSN/credentials as the PostgreSQL exporter for the monitored service (with optional override via inventory, matching MongoDB RTA behavior).

On each collect interval (default 2s, configurable 1–5s via `RTAOptions.collect_interval`):

1. Query `pg_stat_activity` for non-idle backends (include `idle in transaction` as a distinct state).
2. Query `pg_locks` joined with `pg_stat_activity` to build blocker → blocked lock chains.
3. Normalize each active backend into a `QueryData` message with `postgres_payload`.
4. Send changes over the existing RTA collector gRPC stream (`CollectorService`).

**Primary SQL sources:**

```sql
-- pg_stat_activity: session identity, state, query text, timings, wait events
SELECT pid, datname, usename, application_name, client_addr, client_port,
       backend_type, state, query, query_id, backend_start, xact_start,
       query_start, state_change, wait_event_type, wait_event, leader_pid
FROM pg_stat_activity
WHERE pid <> pg_backend_pid()
  AND backend_type = 'client backend'
  AND state <> 'idle';

-- pg_locks: lock mode, relation, granted/blocked
SELECT ... FROM pg_locks l JOIN pg_stat_activity a ON l.pid = a.pid ...
```

**PostgreSQL-specific processing:**

| Concern | Handling |
|---------|----------|
| `query_id` (PG 14+) | Populate from `pg_stat_activity.query_id`; PG 12–13 use query fingerprint hash |
| Query text truncation | Set `query_text_truncated=true` when `length(query) >= track_activity_query_size`; include tooltip metadata |
| `idle in transaction` | Use `xact_start` for duration display (not `query_start`); distinct UI badge |
| Parallel workers (PG 13+) | Group rows where `leader_pid` is set under leader session; collapsed by default in UI |
| Lock chains | Build directed graph from `pg_locks` where `granted=false`; attach chain to blocked session payload |
| Permissions | Require `pg_read_all_stats` (or superuser); return structured agent error if missing |

**Performance target:** <1% additional CPU on PostgreSQL host with ≤200 active backends at 2s polling interval (acceptance criterion).

### Server-side (pmm-managed)

Extend existing components with minimal branching:

| Component | Change |
|-----------|--------|
| `models/agent_model.go` | Add `RTAPostgreSQLAgentType` (`rta-postgresql-agent`) |
| `managed/services/inventory/agents.go` | `AddRTAPostgreSQLAgent`, `ChangeRTAPostgreSQLAgent` |
| `managed/services/management/postgresql.go` | Auto-register RTA agent when adding PostgreSQL service (optional flag, mirror MongoDB) |
| `managed/services/realtimeanalytics/service.go` | Map `PostgreSQLServiceType` → `RTAPostgreSQLAgentType` in `getRTAAgentTypeForServiceType`; extend `ListServicesResponse` |
| `managed/services/realtimeanalytics/store.go` | Store PostgreSQL `QueryData` in existing TTL map (no schema change beyond proto) |
| `managed/services/telemetry/config.default.yml` | Add PostgreSQL RTA telemetry queries |

MongoDB code paths remain unchanged; service-type dispatch selects agent implementation.

### API (protobuf)

**`api/realtimeanalytics/v1/query.proto`:**

```protobuf
message QueryPostgreSQLData {
  string db_instance_address = 1;
  string database_name = 2;
  string username = 3;
  string application_name = 4;
  string session_state = 5;           // active, idle in transaction, ...
  google.protobuf.Timestamp transaction_start_time = 6;
  google.protobuf.Timestamp query_start_time = 7;
  string wait_event_type = 8;
  string wait_event = 9;
  int32 backend_pid = 10;
  int32 leader_pid = 11;                // 0 if not a parallel worker
  bool query_text_truncated = 12;
  int32 track_activity_query_size = 13;
  repeated LockChainLink lock_chain = 14;
}

message LockChainLink {
  int32 blocker_pid = 1;
  int32 blocked_pid = 2;
  string lock_mode = 3;
  string relation_name = 4;
  string blocker_query_text = 5;
  google.protobuf.Duration blocker_duration = 6;
}
```

**`api/realtimeanalytics/v1/realtimeanalytics.proto`:**

- Extend `ListServicesResponse` with `repeated inventory.v1.PostgreSQLService postgresql = 2;`

**`api/inventory/v1/agents.proto`:**

- Add `RTAPostgreSQLAgent`, `AddRTAPostgreSQLAgentParams`, `ChangeRTAPostgreSQLAgentParams` (mirror MongoDB RTA agent fields minus MongoDB-specific auth; reuse PostgreSQL TLS/ssl fields from existing PostgreSQL agents).

### UI (ui/apps/pmm)

Extend existing RTA pages without new navigation:

| Area | Change |
|------|--------|
| `types/rta.types.ts` | Add `QueryPostgreSQLData`, extend `RawQueryData` with `postgresPayload` |
| `api/rta.ts` | Handle PostgreSQL in `listServices` response |
| `hooks/api/useRealtime.ts` | Include PostgreSQL services in available-services filter |
| `pages/rta/overview/table/` | PostgreSQL column set: state, wait event, DB, duration, query |
| `pages/rta/overview/details-pane/` | Lock chain panel, idle-in-transaction section, raw JSON tab |
| `pages/rta/components/selection-form/` | Show PostgreSQL services in autocomplete |

Reuse MongoDB patterns for pause/resume, auto-refresh (1–5s), session management, and share links.

### QAN coexistence

RTA polling reads `pg_stat_activity` and `pg_locks` only. QAN agents continue using `pg_stat_statements` / `pg_stat_monitor` independently. No shared state, no ClickHouse writes from RTA. Verify no lock contention or duplicate query collection on the same extensions.

### Cloud provider notes

| Provider | Considerations |
|----------|----------------|
| RDS / Aurora | Standard `pg_stat_activity`; grant `pg_read_all_stats` via parameter group / role |
| Cloud SQL | May require `cloudsqlsuperuser`; document IAM/database user setup |
| Azure | Flexible Server supports `pg_stat_activity`; document required roles |

Aurora `aurora_stat_activity` is out of scope for v1.

## Files / areas touched

**Agent**
- `agent/agents/postgres/realtimeanalytics/` (new package)
- `agent/agents/agents.go` — register RTA PostgreSQL agent
- `agent/client/client.go` — agent state handling

**API**
- `api/realtimeanalytics/v1/query.proto`
- `api/realtimeanalytics/v1/realtimeanalytics.proto`
- `api/inventory/v1/agents.proto`
- Regenerated `.pb.go`, swagger, json clients

**Managed**
- `managed/models/agent_model.go`, `agent_helpers.go`
- `managed/services/inventory/agents.go`, `grpc/agents_server.go`
- `managed/services/management/postgresql.go`
- `managed/services/realtimeanalytics/service.go`, `store.go`
- `managed/services/converters.go`
- `managed/services/telemetry/config.default.yml`

**UI**
- `ui/apps/pmm/src/types/rta.types.ts`
- `ui/apps/pmm/src/api/rta.ts`
- `ui/apps/pmm/src/hooks/api/useRealtime.ts`
- `ui/apps/pmm/src/pages/rta/**`

**Docs**
- `documentation/docs/use/qan/QAN-realtime-analytics.md` — add PostgreSQL section, remove "MongoDB only" warning
- New or updated PostgreSQL RTA permissions doc

**Version**
- `version/features.go` — feature gate constant for PostgreSQL RTA agent support

## Risks

| Risk | Mitigation |
|------|------------|
| High poll frequency increases DB load | Default 2s interval; configurable 1–5s; benchmark ≤200 backends |
| `pg_read_all_stats` missing on cloud | Structured error in agent + UI; document per-provider setup |
| Query text truncated by `track_activity_query_size` | Visual indicator + tooltip; document server restart requirement to increase |
| Lock chain query cost on busy systems | Limit chain depth; single round-trip join; benchmark under load |
| Proto/API breaking changes | Additive only; MongoDB payload unchanged |
| Parallel worker UI complexity | Collapse under leader by default; expand on click |

## Testing strategy

- **Unit:** Agent SQL parsing, lock chain builder, proto converters, store TTL
- **Integration (pmm-managed):** Start/stop session for PostgreSQL service; SearchQueries returns PG payload
- **Integration (pmm-qa):** Playwright RTA flow for PostgreSQL service selection, live table refresh, lock chain panel
- **Manual matrix:** PG 12, 14, 15, 16; Percona Distribution; RDS; verify QAN still works on same instance
