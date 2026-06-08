# qan

`qan` is the Query Analytics service for PMM. It replaces `qan-api2`: it ingests
metrics buckets from pmm-agent (via pmm-managed) and serves the QAN UI, storing
data in ClickHouse as composable, pre-aggregated rollups.

## Design highlights

- **Pre-aggregated rollups** (`metrics_raw` → `metrics_1h` → `metrics_1d`) built with
  `AggregatingMergeTree` + `SimpleAggregateFunction`, so reads hit the coarsest tier
  that covers the requested range.
- **Mergeable percentiles** via DDSketch bucket-count maps merged with `sumMap` and
  resolved at read time (`utils/ddsketch`). No version-coupled aggregate state.
- **Filters** served from a precomputed `dim_values` table (no fact-table scans).
- Drop-in replacement: same gRPC/gateway/debug ports (9911/9922/9933) and the same
  `pmm` ClickHouse database as `qan-api2`.

## Layout

- `migrations/` — ClickHouse schema (one statement per file; golang-migrate).
- `ddsketch/` — frozen DDSketch bucket layout and read-time quantiles.
- `models/` — ingestion and read (reporter) data access.
- `services/receiver/` — `CollectorService` (ingestion).
- `services/analytics/` — `QANService` (serving).

Run `make release` to build the `qan` binary into `../bin/qan`.
