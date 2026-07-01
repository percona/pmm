# eBPF / OTLP identity contract v1 (Phase 1)

Version: **v1** (2026-03-28). Applies to **metrics** and **spans** emitted by PMM-managed collectors (stub, synthetic, or eBPF-backed) exported via OTLP to PMM Server.

## Purpose

Enable **service map rollups**, **inventory correlation**, and **Grafana Node Graph** without unbounded cardinality. Records that do not satisfy this contract are **excluded from service-map rollups** (they may still be stored for other uses if the pipeline accepts them).

## Resource attributes (required)

| Attribute | Semantics |
|-----------|-----------|
| `service.name` | Logical client/service name for the emitting process. |
| `pmm.node_id` | PMM inventory node identifier (`runs_on_node_id` / container node). |
| `pmm.agent_id` | PMM inventory agent identifier for the collector. |
| `net.peer.name` OR `net.peer.ip`+`net.peer.port` | Stable peer key for the remote leg (normalized IP:port or DNS). At least one of these must be present. |
| `db.system` | For database legs: `mysql`, `postgresql`, or `mongodb`. For non-DB hops use `generic` or omit only if `pmm.component_role` is not `database`. |
| `pmm.component_role` | One of: `app`, `proxy`, `database`, `router`, `unknown`. |

## Span attributes (required when exporting spans)

Same resource attributes are expected on the span’s resource. Additionally:

| Attribute | When |
|-----------|------|
| `error.type` | Present on error spans. |
| `pmm.map_edge_target` | Optional; **directed edge** for Node Graph: target service id (string) for MVP rollups derived from spans. |

## Metric names (RED — canonical Phase 1)

- Counter: `db.client.requests` (labels: `db.system`, result/status dimension as allowed label set).
- Counter: `db.client.errors`
- Histogram (preferred) or summary: `db.client.operation.duration` (bounded labels; **no raw SQL** as label).

Label allow-list and cardinality caps are enforced in collector and server configuration (see `docs/internal/ebpf-clickhouse-ingestion-profile.md`).

## Span naming

- `db.<system>.<operation>` (e.g. `db.mysql.query`)
- `net.flow` when protocol parsing is not yet available

## Sampling

- Default: head-based sampling for spans.
- Optional: error-biased sampling for incidents (collector flag).

## Observability

Implementations SHOULD increment a rejection counter (e.g. `otel_identity_rejected_total`) when map-critical telemetry is dropped due to missing required attributes.
