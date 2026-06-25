# Tasks

Check off items as implemented. Phases align with Jira epic delivery order.

## Phase 1 — PostgreSQL RTA Agent

- [x] 1.1 Add `RTAPostgreSQLAgentType` to inventory protobuf and `managed/models`
- [x] 1.2 Implement `agent/agents/postgres/realtimeanalytics` collector (pg_stat_activity + pg_locks)
- [x] 1.3 Build lock chain graph from pg_locks (blocker → blocked with query text and duration)
- [x] 1.4 Map session fields: state, wait events, xact_start vs query_start, leader_pid
- [x] 1.5 Populate query_id on PG 14+; fingerprint fallback on PG 12–13
- [x] 1.6 Detect query text truncation vs track_activity_query_size
- [x] 1.7 Return actionable error when monitoring user lacks pg_read_all_stats
- [x] 1.8 Wire agent into pmm-agent SetState / QANCollect-equivalent RTA stream
- [x] 1.9 Unit tests: lock chain builder, state mapping, truncation detection, permission error

## Phase 2 — Server-Side & API Extension

- [x] 2.1 Add QueryPostgreSQLData and LockChainLink to query.proto; regenerate stubs
- [x] 2.2 Extend ListServicesResponse with postgresql services
- [x] 2.3 Implement AddRTAPostgreSQLAgent / ChangeRTAPostgreSQLAgent in inventory service
- [x] 2.4 Extend realtimeanalytics service dispatch for PostgreSQLServiceType
- [x] 2.5 Store and serve PostgreSQL QueryData via existing in-memory TTL store
- [x] 2.6 Add PostgreSQL RTA telemetry metrics (registered, enabled, disabled counts)
- [x] 2.7 Optional: auto-register RTA agent when adding PostgreSQL service (mirror MongoDB flag)
- [ ] 2.8 Integration tests: start session, collect queries, stop session for PostgreSQL
- [ ] 2.9 Verify MongoDB RTA regression tests still pass

## Phase 3 — UI Integration

- [x] 3.1 Extend rta.types.ts and API client for PostgreSQL payload and services list
- [x] 3.2 Include PostgreSQL services in Real-time service selector and session modal
- [ ] 3.3 Overview table: PostgreSQL columns (state, wait event, DB, duration, query)
- [x] 3.4 Details pane: lock chain panel (blocker → blocked with query + duration)
- [x] 3.5 Visual distinction for idle in transaction (transaction duration, badge)
- [ ] 3.6 Collapse parallel workers under leader by default (PG 13+)
- [x] 3.7 Truncated query tooltip (track_activity_query_size + restart note)
- [ ] 3.8 Permission error banner when pg_read_all_stats missing
- [x] 3.9 Raw data tab shows full pg_stat_activity / lock JSON snapshot
- [ ] 3.10 Component tests for PostgreSQL-specific rendering

## Phase 4 — Testing, Hardening & GA

- [ ] 4.1 Benchmark agent CPU overhead at 2s interval (≤200 backends, <1% target)
- [ ] 4.2 Verify QAN (pg_stat_statements + pg_stat_monitor) unaffected on same instance
- [ ] 4.3 Test matrix: PG 12, 14, 15, 16; Percona Distribution; RDS; Aurora; Cloud SQL; Azure
- [ ] 4.4 Update QAN-realtime-analytics.md (PostgreSQL section, remove MongoDB-only warning)
- [ ] 4.5 Document cloud permission setup and query_id version matrix
- [ ] 4.6 pmm-qa Playwright scenarios for PostgreSQL RTA happy path and error states
- [ ] 4.7 Enable feature by default for registered PostgreSQL services (or document opt-in if required)

## Verification checklist (acceptance criteria mapping)

- [ ] AC-1: PostgreSQL service selectable; sessions refresh ≤5s
- [ ] AC-2: RTA + QAN coexist without conflict on same instance
- [ ] AC-3: Works on PG 12+, Percona Distribution, RDS, Aurora, Cloud SQL, Azure
- [ ] AC-4: Lock chain panel shows blocker → blocked with query text and duration
- [ ] AC-5: idle in transaction visually distinct; shows transaction duration
- [ ] AC-6: Parallel workers collapsed under leader (PG 13+)
- [ ] AC-7: query_id on PG 14+; fingerprint fallback on 12–13
- [ ] AC-8: Truncated query text indicated with tooltip
- [ ] AC-9: Missing pg_read_all_stats shows actionable error
- [ ] AC-10: No regressions in MongoDB RTA or PostgreSQL QAN
- [ ] AC-11: <1% CPU overhead at 2s polling (≤200 backends)
- [ ] AC-12: Official docs updated with version matrix and cloud permissions
