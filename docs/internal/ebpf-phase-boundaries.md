# Phase 1 vs Phase 2 boundaries (eBPF / service map)

## Phase 1 (in scope)

- OTLP metrics + sampled spans to existing PMM ClickHouse (`otel` DB extension).
- RED metrics as canonical DB client signal for MySQL, PostgreSQL, MongoDB.
- Grafana service map MVP (Node Graph target; tabular / trace-derived edges acceptable for early builds).
- Proxy legs as first-class network entities: HAProxy, ProxySQL, pgBouncer, mongos (role tags + edges).
- Patroni (and similar) as **control-plane markers** (failover windows), not deep parser metrics.
- `pmm-admin management add otel ebpf` (merges Phase 1 eBPF labels on the single node OTEL collector).

## Phase 2 (explicitly out of scope for Phase 1)

- Deep per-component parsers (full proxy query semantics, orchestrator internals).
- External OTLP tenants beyond PMM-managed collectors.
- Advanced PMM-app-only map UX (beyond Grafana MVP).
