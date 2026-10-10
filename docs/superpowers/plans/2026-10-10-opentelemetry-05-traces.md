# OTel 05: Traces Implementation Plan (task level, re-plan before execution)

> **For agentic workers:** this plan is at **task level**. Its interfaces depend on code that plans 01–04 create. Before executing it, re-run superpowers:writing-plans on this file against the merged code to expand each task into steps. Then use superpowers:subagent-driven-development.

**Goal:** Applications instrumented with OpenTelemetry send traces to their node's collector. PMM samples them, stores them in `otel.otel_traces`, shows them in a Traces dashboard with a service map, and links spans to PMM services, QAN and logs.

**Architecture:**
- `pmm-admin add traces` turns on an OTLP receiver (gRPC and HTTP, `127.0.0.1` by default) in the node collector. That collector overwrites the PMM identity, samples probabilistically, caps the span rate, and forwards over the existing `/otlp/` path.
- The server collector gets a traces pipeline into ClickHouse.
- Correlation is done at query time, through an `otel.pmm_services` table that pmm-managed keeps in sync with inventory.

**Spec:**
- PMM-15579, PMM-15580, PMM-15582;
- [analysis](../specs/2026-10-10-opentelemetry-analysis.md) §1 (profiles and eBPF are out of scope), §3;
- the [roadmap](2026-10-10-opentelemetry-00-roadmap.md).

## Global Constraints

The roadmap's constraints apply. In addition, from PMM-15579:
- Without `add traces`, no OTLP port is open.
- With it, the receiver binds `127.0.0.1` unless `--listen-address` is set. Defaults are `--grpc-port` 4317 and `--http-port` 4318.
- Spans carry the node's `pmm.node_id` and `pmm.agent_id`, even when the sender set other values, and keep the sender's `service.name`.
- Sampling defaults to 10 %, set per node with `--sampling-percent`.
- A per-node span-rate cap uses a `tail_sampling` rate-limiting policy, with a provisional 1000 spans/s until PMM-15592.
- `traces_retention_days` applies to stored spans through plan 01's cleaner, which already handles tables starting with `otel_traces`.
- The dashboard uses no unsigned plugin.

## Review Focus

1. **A node that already runs a collector or Grafana Alloy on 4317/4318.** Expected: the collector fails to bind, and the source of the problem shows in `pmm-admin list` and in Inventory; other agents are unaffected. Test in Task 3.
2. **A sender that sets `pmm.node_id` or `pmm.service_id` itself.** Expected: `pmm.node_id` and `pmm.agent_id` are overwritten, and `pmm.service_id` is dropped. Only correlation (Task 5) assigns a service. Test in Task 3.
3. **A trace whose spans come from two nodes.** Expected: every span is stored with its own node's identity, and the trace view joins them by `TraceId`. Test in Task 1 (helper-table lookup).
4. **A trace id typed into the dashboard with quotes.** Expected: `${trace_id:sqlstring}`, so no SQL breakage. Test in Task 4.
5. **A span whose `server.address` is a DNS name while inventory holds an IP, or the reverse.** Expected: no link, and nothing fails. Only exact address-and-port matches link. Test in Task 5.

---

### Task 1: Trace schema

- **Files:**
  - `qan-api2/migrations/otel/sql/02_traces.up.sql` and `02_traces.down.sql`
  - `qan-api2/migrations/otel_test.go`
- **Produces:**
  - `otel.otel_traces`: the upstream exporter's columns, plus `PmmNodeId` and `PmmServiceId` materialized from `ResourceAttributes`. `PARTITION BY toDate(Timestamp)`, `ORDER BY (PmmNodeId, ServiceName, SpanName, toDateTime(Timestamp))`.
  - `otel.otel_traces_trace_id_ts (TraceId String, Start DateTime, End DateTime)`, a `ReplacingMergeTree` filled by a materialized view `otel_traces_trace_id_ts_mv`, copied from the pinned exporter's templates.
  - `otel.otel_traces_service_name (ServiceName LowCardinality(String), PmmNodeId LowCardinality(String), LastSeen DateTime)`, a `ReplacingMergeTree(LastSeen)` filled by `otel_traces_service_name_mv`.
- **Tests:**
  - `TestRunOtelCreatesTraces`: the tables exist, and inserting spans fills both helper tables.
  - `TestTraceLookupByID`: a trace whose spans come from two nodes is found by `TraceId` through the helper table.
- **Commit:** `PMM-15579 Add otel trace tables`

### Task 2: Server traces pipeline

- **Files:**
  - `managed/services/otel/server_config.go`, its golden file and its test
  - `managed/services/otel/contract_test.go`
- **Produces:**
  - a `traces` pipeline: `[otlp] → [memory_limiter, transform/redact, batch] → [clickhouse]`, with `traces_table_name: otel_traces`;
  - redaction that also runs on `span.attributes`, where `db.statement` can carry `password=` (`trace_statements`, with the same four patterns).
- **Tests:**
  - an updated golden file;
  - `TestCollectorContractTraces`: one OTLP JSON span with `db.statement` containing `password=hunter2` lands in `otel.otel_traces` with the password masked.
- **Commit:** `PMM-15579 Store traces on PMM Server`

### Task 3: Node traces receiver, sampling, and `pmm-admin add/remove traces`

- **Files:**
  - `api/agent/v1/otel.proto`: `OtelTracesReceiver { bool enabled = 1; string listen_address = 2; uint32 grpc_port = 3; uint32 http_port = 4; uint32 sampling_percent = 5; uint32 max_spans_per_second = 6; }` as `OtelCollectorParams.traces_receiver = 5`
  - `api/otel/v1/otel.proto`: `ChangeTracesReceiver`
  - `managed/models/agent_model.go` and the next migration: a JSONB `otel_options` column on `agents`, `OtelCollectorOptions{TracesEnabled bool; ListenAddress string; GRPCPort, HTTPPort, SamplingPercent uint32}`
  - `managed/services/agents/otelcollector.go`
  - `agent/agents/otelcollector/render.go`:
    - an `otlp` receiver;
    - a `traces` pipeline: `[otlp] → [memory_limiter, resource/pmm (upsert pmm.node_id, pmm.agent_id; delete pmm.service_id, pmm.service_name), probabilistic_sampler, tail_sampling (rate_limiting), batch] → [otlphttp]`
  - `version/features.go`: `OtelTracesSupportVersion`
  - `admin/commands/management/add_traces.go` and `remove_traces.go` (following plan 04's D9 outcome)
  - `admin/commands/list.go`: receiver address, ports and sampling
- **Tests:**
  - `TestRenderTracesReceiverDefaults`: `127.0.0.1:4317` and `127.0.0.1:4318`, sampling 10.
  - `TestRenderNoOTLPWithoutTraces`.
  - `TestRenderTracesOverwritesIdentity`: the resource processor upserts `pmm.node_id` and `pmm.agent_id` and deletes `pmm.service_id`.
  - `TestAddTracesFlags` and `TestRemoveTraces`.
  - `TestChangeTracesReceiverOldAgent` (`FailedPrecondition`).
  - On a CHAOS VM: a port clash shows in `pmm-admin list`; `telemetrygen traces --otlp-insecure` at 100 % sampling lands in `otel.otel_traces`, and at the default about 10 % does.
- **Commit:** `PMM-15579 Add pmm-admin add/remove traces`

### Task 4: Traces dashboard with a service map

- **Files:**
  - `dashboards/dashboards/OTel/Traces.json` (uid `pmm-otel-traces`). Start from `git show origin/tibi-holmes:dashboards/dashboards/Experimental/OTel_ClickHouse_Traces_and_Service_Map.json`, and remove the `pmm-service-map-panel` panels and the coroot recording-rule panels.
  - `dashboards/pmm-app/src/plugin.json`
  - the dashboards Python test suite
- **Panels:**
  - spans and distinct traces;
  - error spans and error %;
  - throughput;
  - duration p50, p95 and p99;
  - top services and top span names;
  - a `SpanKind` breakdown;
  - DB client spans;
  - a Node Graph service map with edge drill-down;
  - a span table linking to Logs by `TraceId`.
- **Filters:** node, PMM service and app.
- **Tests:**
  - `test_otel_trace_id_is_sqlstring`;
  - `test_no_unsigned_panels` (every panel `type` is a core Grafana panel);
  - `cleanup-dash.py --check-only`.
- **Commit:** `PMM-15580 Add the Traces dashboard`

### Task 5: Correlation

- **Files:**
  - `qan-api2/migrations/otel/sql/03_pmm_services.up.sql`: `otel.pmm_services (service_id String, service_name String, service_type LowCardinality(String), node_id String, address String, port UInt16, version UInt64, is_deleted UInt8) ENGINE = ReplacingMergeTree(version, is_deleted) ORDER BY (address, port, service_id)`
  - `managed/services/otel/service_sync.go`: a leader service that runs every 60 s and inserts inventory changes, with tombstones for deleted services, through the privileged ClickHouse client
  - the Logs and Traces dashboards: a span's (`server.address`, `server.port`), falling back to `net.peer.name` and `net.peer.port`, joined to `pmm_services FINAL WHERE is_deleted = 0`; links to the service dashboard and to QAN (`/graph/d/pmm-qan/pmm-query-analytics?var-service_name=…&${__url_time_range}`)
  - the service summary dashboards: a link to Traces filtered by `service_id`
- **Produces:** `func SyncServices(ctx context.Context, q *reform.Querier, ch ClickHouse, since uint64) (uint64, error)`
- **Tests:**
  - `TestSyncServicesInsertsAndTombstones` (sqlmock);
  - `test_trace_service_join_exact_match` (dashboard SQL contains the address-and-port equality);
  - on a Feature Build: a span to a monitored MySQL's address carries a link that opens QAN filtered to that service.
- **Commit:** `PMM-15582 Correlate spans with PMM services`
