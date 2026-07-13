# Phase 1 ClickHouse ingestion profile (eBPF / OTLP)

Aligned with PMM server OTEL collector → ClickHouse (`otel` database).

## Collector (server)

- Receivers: OTLP gRPC `:4317`, HTTP `:4318`.
- Processors: `memory_limiter`, `transform` (logs only), `batch` (1s / 1024–2048).
- Exporters: three ClickHouse exporter instances — `clickhouse/logs`, `clickhouse/traces`, `clickhouse/metrics` (OTEL component type `clickhouse` with distinct names) — `create_schema: false` (PMM-managed DDL).

## TTL

- Logs: from PMM settings `GetOtelLogsRetentionDays()` (default 7d).
- Traces (`otel.otel_traces`): **7d** (MVP).
- Sum metrics (`otel.otel_metrics_sum`): **90d** (MVP RED).
- Service map rollup tables (`service_map_*_1m`): 32d partition TTL (adjust with product defaults).

## Batching / reliability (client agent `otelcol-contrib`)

- `batch`: timeout **2s**, `send_batch_size` **10000** (existing PMM client template).
- Server export: OTLP HTTP to PMM `/otlp` with auth headers.

## Async inserts / compression

- Rely on ClickHouse exporter defaults (`async_insert`, `lz4`) where supported by the bundled `otelcol-contrib` version.

## Cardinality

- Enforce identity contract label allow-list; never emit raw SQL text as metric labels.
