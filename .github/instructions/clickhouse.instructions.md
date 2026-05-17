---
applyTo: agent/agents/clickhouse/**,agent/cmd/clickhouse_exporter/**
---
# ClickHouse Exporter & QAN Development Guidelines

> **Parent guide**: [AGENTS.md](../../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [agent.instructions.md](agent.instructions.md) (pmm-agent) · [build.instructions.md](build.instructions.md) (packaging)

PMM monitors ClickHouse through two metric sources and an in-process Query
Analytics (QAN) agent. This guide covers the code under
`agent/agents/clickhouse/**` and the exporter binary under
`agent/cmd/clickhouse_exporter/**`.

## Components

### `clickhouse_exporter` (`agent/cmd/clickhouse_exporter/`)

A standalone Prometheus exporter, packaged inside `pmm-client` and run by the
pmm-agent supervisor as a process agent — exactly like `node_exporter` and the
other exporters. It is used for ClickHouse servers that do **not** expose the
native `<prometheus>` endpoint (including ClickHouse older than 22.6).

- Binary name is exactly `clickhouse_exporter` (underscore), installed at
  `/usr/local/percona/pmm/exporters/clickhouse_exporter`.
- The path is configured by `agent/config/config.go` `Paths.ClickHouseExporter`
  (default `clickhouse_exporter`, resolved relative to `ExportersBase`).
- `main.go` wires `clickhouse.NewCollector` into an HTTP `/metrics` handler.
  Flags: `--web.listen-address`, `--web.telemetry-path`, `--clickhouse.dsn`,
  `--version`. The DSN also reads from `CLICKHOUSE_EXPORTER_DSN`.
- The driver is `clickhouse-go/v2` (pure Go), so the binary builds with
  `CGO_ENABLED=0`.

### ClickHouse collector (`agent/agents/clickhouse/`)

`collector.go` implements `prometheus.Collector`, querying `system.metrics`,
`system.asynchronous_metrics`, `system.events`, and version info. It is shared
by the exporter binary. `config.go` holds the exporter defaults.

### QAN query log agent (`agent/agents/clickhouse/querylog/`)

A built-in agent that reads `system.query_log`, fingerprints queries, and
produces QAN buckets. `fingerprint.go` normalizes SQL; `buckets.go` aggregates
metrics; `percentile` handles latency percentiles. The server must run with
`log_queries=1`.

## Build & test

The `build/Makefile.clickhouse` include adds:

- `make build-clickhouse-exporter` — build the binary into `bin/`.
- `make test-clickhouse-unit` — run exporter + QAN unit tests.
- `make test-clickhouse-matrix` — run the integration matrix (Docker required).

Unit tests must pass without the `clickhouse_integration` build tag:

```sh
go test ./agent/agents/clickhouse/... ./agent/cmd/clickhouse_exporter/...
```

Integration tests are tagged `clickhouse_integration` and driven by
`agent/agents/clickhouse/testdata/run-matrix.sh`, which brings up the supported
ClickHouse versions in single-node and cluster topologies. The matrix is the
local pre-tag release gate.

## Conventions

- Keep the exporter's metric families aligned with the ClickHouse native
  `<prometheus>` endpoint so `auto` / `native` / `exporter` sources are
  interchangeable from the dashboards' point of view.
- The collector and QAN agent only read from the `system` database — never
  require write privileges.
- Mirror the surrounding Percona style and Apache 2.0 license headers; all
  comments and docs are in English.
