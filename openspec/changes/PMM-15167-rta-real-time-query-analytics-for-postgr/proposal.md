# PMM-15167 Real-Time Query Analytics for PostgreSQL

## Why

PMM PostgreSQL users currently rely on Query Analytics (QAN), which aggregates completed queries into one-minute buckets in ClickHouse with 1–2 minute visibility lag. During live incidents — lock pile-ups, long-running transactions, idle-in-transaction sessions — they must leave PMM and query `pg_stat_activity` and `pg_locks` manually. MongoDB RTA shipped in PMM v3.7.0 with a proven end-to-end architecture; PostgreSQL has no equivalent despite richer real-time diagnostics (session states, lock chains, wait events, parallel workers).

## What changes

- Add a **PostgreSQL RTA agent** (`rta-postgresql-agent`) in pmm-agent that polls `pg_stat_activity` and `pg_locks` every 1–5 seconds and streams session records to PMM Server.
- Extend **Inventory API** with `RTAPostgreSQLAgent` type (add/change/list) and wire it through pmm-managed agent state management.
- Extend **RealtimeAnalyticsService** and **CollectorService** to accept, route, and expose PostgreSQL session data via existing REST/gRPC endpoints consumed by the UI.
- Extend **QueryData** protobuf with `QueryPostgreSQLData` payload (session state, wait events, lock chains, idle-in-transaction, parallel worker metadata).
- Integrate **UI Real-Time tab** so users can select PostgreSQL services in Query Analytics → Real-Time with lock chain visualization, idle-in-transaction highlighting, and parallel worker grouping — no new navigation.
- Add **telemetry** counters for PostgreSQL RTA agents (mirroring MongoDB RTA telemetry).
- Document version matrix, cloud permission requirements, and known limitations.

## Out of scope

- `EXPLAIN` / execution plan capture in the real-time view.
- `pg_wait_sampling` and `pgsentinel` integration.
- Cancel / terminate query action from RTA UI.
- Active Session History (ASH) style persistence for PostgreSQL RTA data.
- Aurora-specific `aurora_stat_activity` extended view integration.
- PostgreSQL logical replication lag monitoring in RTA.

## Acceptance criteria (from Jira)

- User selects any registered PostgreSQL service in Query Analytics → Real-Time and sees executing sessions with ≤5s refresh latency.
- RTA and QAN (`pg_stat_statements` / `pg_stat_monitor`) operate independently on the same instance with no conflict.
- Works on PostgreSQL 12+, Percona Distribution for PostgreSQL, and major cloud variants (RDS, Aurora, Cloud SQL, Azure).
- Lock chain panel shows blocker → blocked chains with query text and duration per link.
- `idle in transaction` sessions are visually distinguished with transaction duration.
- Parallel worker sessions collapsed under leader by default (PostgreSQL 13+).
- `query_id` populated on PostgreSQL 14+; fingerprint fallback on 12–13.
- Query text truncation indicated with tooltip explaining `track_activity_query_size`.
- Actionable error when monitoring user lacks `pg_read_all_stats`.
- No regressions in MongoDB RTA or PostgreSQL QAN stored metrics.
- Performance overhead of 2s polling interval is <1% additional CPU on hosts with ≤200 active backends.
- Feature documented in PMM official docs.

## Phases

| Phase | Scope |
|-------|-------|
| 1 | PostgreSQL RTA agent — collect from `pg_stat_activity` / `pg_locks`, forward to server |
| 2 | Server-side & API extension — inventory, RealtimeAnalyticsService, in-memory store |
| 3 | UI integration — Real-Time tab for PostgreSQL services |
| 4 | Testing, hardening & GA — version matrix, integration tests, docs |

## References

- Related epic: PMM-14550 (MongoDB RTA MVP — Done)
- Research doc: [PostgreSQL RTA research](https://docs.google.com/document/d/1IesHktqVKV0tpRLwryBsarml8XyZwmcv-n2AwZ73-MM/edit)
