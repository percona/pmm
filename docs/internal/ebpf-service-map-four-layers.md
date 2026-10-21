# Service map four layers (Phase 1)

1. **Flow extraction** — eBPF (later) or stub OTLP emits RED metrics + sampled spans; enrichment with node/pod metadata.
2. **Identity / topology** — stable `service_id` / `pmm.node_id` / peer keys; directed edges; proxy/HA roles (`docs/internal/ebpf-proxy-ha-correlation.md`).
3. **Graph rollups** — ClickHouse tables `otel.otel_traces`, `otel.otel_metrics_sum`, rollups `otel.service_map_nodes_1m`, `otel.service_map_edges_1m` (MVs / scheduled aggregations as load requires).
4. **UX** — Grafana Node Graph + drilldowns to traces / PMM dashboards; PMM-app custom map deferred post-MVP.

Dependency order for implementation: **1 → 2 → 3 → 4**; stub OTLP validates 3–4 before real probes land.
