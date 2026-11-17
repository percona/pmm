# Phase 1 eBPF MVP — acceptance criteria (locked defaults)

| Area | MVP |
|------|-----|
| Sources | PMM-managed collectors only |
| Identity | Strict required attributes for map rollups (`docs/internal/ebpf-otel-identity-v1.md`) |
| Ingestion SLO | Drop rate **&lt; 1%** under reference load (measure at collector queue + CH exporter) |
| Retention | Metrics-oriented sum table **90d**; spans **7d**; logs per PMM setting (default 7d) |
| UX | Grafana-first service map + drilldown links; PMM-app pages post-MVP |
| Delivery | Vertical slice first: stub OTLP → server → CH → Grafana; real CO-RE probes per protocol after |
