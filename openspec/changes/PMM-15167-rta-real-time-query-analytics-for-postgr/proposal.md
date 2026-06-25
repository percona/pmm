# PMM-15167 [RTA] Real-Time Query Analytics for PostgreSQL

## Why

PMM already provides Query Analytics (QAN) for PostgreSQL using `pg_stat_statements` and `pg_stat_monitor`, but QAN aggregates completed queries into one-minute buckets in ClickHouse with 1–2 minutes of end-to-end lag. During live incidents—lock pile-ups, long-running transactions blocking writes, or idle-in-transaction sessions holding locks—DBAs must leave PMM and run manual queries against `pg_stat_activity` and `pg_locks`.

Real-Time Analytics (RTA) for MongoDB shipped in PMM 3.7.0 with a proven architecture: dedicated agent type, `RealtimeAnalyticsService` (gRPC + REST), in-memory TTL store, and a UI under **Query Analytics → Real-time**. PostgreSQL users need the same live visibility inside PMM, extended with PostgreSQL-specific diagnostics (lock chains, wait events, idle-in-transaction detection, parallel worker grouping).

## What changes

- Add **RTA PostgreSQL agent** (`rta-postgresql-agent`) in pmm-agent that polls `pg_stat_activity` and `pg_locks` every 1–5 seconds and streams session records to PMM Server.
- Extend **Inventory API** with `RTAPostgreSQLAgent` (add/change/list/get) mirroring the MongoDB RTA agent pattern; reuse existing PostgreSQL exporter credentials where possible.
- Extend **RealtimeAnalyticsService** and `QueryData` protobuf with a PostgreSQL payload (`QueryPostgreSQLData`) including session state, wait events, lock chains, transaction duration, and parallel-worker metadata.
- Update **UI Real-time tab** to list PostgreSQL services alongside MongoDB, render PostgreSQL-specific columns and panels (lock chain, idle-in-transaction badge, parallel worker collapse), and surface permission/truncation errors clearly.
- Add **telemetry** counters for PostgreSQL RTA agents (registered, enabled, disabled).
- Document PostgreSQL RTA in official PMM docs with version matrix (`query_id`, cloud permissions) and known limitations.

## Out of scope

- `EXPLAIN` / execution plan capture in the real-time view (future; may leverage `pg_stat_monitor` separately).
- `pg_wait_sampling` and `pgsentinel` integration.
- Cancel / terminate query action from RTA UI.
- ASH-style persistence for PostgreSQL RTA data.
- Aurora-specific `aurora_stat_activity` extended view integration.
- PostgreSQL logical replication lag monitoring in RTA.

## Related work

- Epic: PMM-14550 (MongoDB RTA MVP — reference architecture).
- Jira: [PMM-15167](https://perconadev.atlassian.net/browse/PMM-15167)

## Phases (delivery)

| Phase | Scope | Done when |
|-------|-------|-----------|
| 1 — Agent | Collect from `pg_stat_activity` + `pg_locks` | Agent produces session records with lock chains and idle-in-transaction info; verified on PG 12, 14, 15, 16, and Percona Distribution for PostgreSQL |
| 2 — Server & API | Extend RealtimeAnalyticsService + Inventory | Server routes PostgreSQL RTA data via existing REST/gRPC API; MongoDB behavior unchanged |
| 3 — UI | Real-time tab integration | User selects PostgreSQL service and sees live sessions at 1–5s refresh with PG-specific visuals |
| 4 — Hardening & GA | Tests, docs, cloud variants | Feature enabled for registered PostgreSQL services; integration tests pass; docs published |
