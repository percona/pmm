# OTEL ClickHouse schema — coroot alignment and rollouts

## Current state (PMM-managed DDL)

- **`otel.logs`** — Canonical raw logs table; DDL in `managed/otel/schema.go` follows the agreed **superset** shape (scope/resource fields, map attributes, indexes) suitable for coroot-node-agent OTLP logs.
- **`otel.otel_traces`**, **`otel.otel_metrics_sum`**, service-map rollups — `managed/otel/ebpf_schema.go` (`EnsureOtelTracesMetricsAndServiceMapTables`).
- **Helper tables / materialized views** — `EnsureOtelCorootHelperTables` creates:
  - `otel.logs_service_name_severity_text` + `_mv`
  - `otel.otel_traces_trace_id_ts` + `_mv`
  - `otel.otel_traces_service_name` + `_mv`

Supervisord calls these ensures when OTEL is enabled (`managed/services/supervisord/supervisord.go`).

## Full migration story (plan Phase 9 — if legacy drift appears)

If an older PMM deployment has a **narrower** `otel.logs` definition, a staged migration may be required:

1. Create `otel.logs_v2` (or new table) with final DDL.
2. Dual-write from the collector exporter for a bounded window.
3. Backfill `INSERT SELECT` with defaults for new columns.
4. Switch readers (API/UI) to the new table.
5. Rename/swap and drop legacy after validation.

That dual-write/cutover **is not automated in this document**; it is only needed when an environment’s existing table definition diverges from `EnsureOtelSchema`. Green-field installs use the managed DDL as-is.

## UI/API reader integration

Using **`logs_service_name_severity_text`** (and trace helpers) for facet/dropdown queries reduces scan cost versus raw tables. Wire PMM read paths to prefer helpers where semantically equivalent; keep raw tables for row retrieval.

Canonical SQL fragments for Go callers live in **`managed/otel/queries.go`** (`LogsServiceSeverityFacetSQL`, `TracesServiceLastSeenFacetSQL`, `TracesTraceIDWindowSQL`).
