# PMM-15167 Real-Time Query Analytics for PostgreSQL

## Why

PMM provides two query analysis modes for PostgreSQL:

- **Query Analytics (QAN)** aggregates completed queries into one-minute buckets stored in ClickHouse (`pg_stat_statements` or `pg_stat_monitor`). End-to-end visibility lag is 1–2 minutes — too slow for live incident triage.
- **Real-Time Analytics (RTA)** polls actively executing operations every 1–5 seconds and streams them to the UI without aggregation or persistence. RTA launched for MongoDB in PMM v3.7.0; the production architecture (`RTAMongoDBAgentType`, `RealtimeAnalyticsService`, in-memory TTL store, UI with service selector, pause/resume, raw diagnostics panel) is proven.

PostgreSQL users currently have no visibility into actively executing queries inside PMM. During lock pile-ups, long-running transactions, or idle-in-transaction sessions holding locks, they must leave PMM and run manual queries against `pg_stat_activity` and `pg_locks`.

## What changes

- Add a **PostgreSQL RTA agent** (`RTAPostgreSQLAgentType`) in pmm-agent that polls `pg_stat_activity` and `pg_locks` every 1–5 seconds and forwards session records to PMM Server via the existing RTA gRPC collector channel.
- Extend **RealtimeAnalyticsService** and the Inventory API to register, route, and expose PostgreSQL RTA data through the same REST/gRPC endpoints consumed by the UI.
- Extend **Query Analytics → Real-Time tab** to list PostgreSQL services, display live sessions, and surface PostgreSQL-specific diagnostics: lock chains, idle-in-transaction detection, wait event breakdown, and parallel worker grouping (PG 13+).
- Add `QueryPostgreSQLData` payload to the RTA protobuf schema alongside the existing MongoDB payload.
- Enable RTA by default for registered PostgreSQL services (Phase 4 / GA).
- Document version matrix, permission requirements (`pg_read_all_stats`), and known limitations.

## Out of scope

- PostgreSQL `EXPLAIN` / execution plan capture in the real-time view.
- `pg_wait_sampling` and `pgsentinel` integration.
- Cancel / terminate query action from RTA UI.
- Active Session History (ASH) style persistence for PostgreSQL RTA data.
- Aurora-specific `aurora_stat_activity` extended view integration.
- PostgreSQL logical replication lag monitoring in RTA.
