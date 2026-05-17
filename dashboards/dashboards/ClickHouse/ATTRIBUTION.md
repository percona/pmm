# ClickHouse Dashboards — Attribution

The dashboards in this directory monitor ClickHouse via the PMM
Prometheus/VictoriaMetrics datasource, using the `ClickHouseMetrics_*`,
`ClickHouseAsyncMetrics_*` and `ClickHouseProfileEvents_*` metric families
emitted by the Phase 1 ClickHouse collector
(`agent/agents/clickhouse/collector.go`).

## Design reference — ClickHouse mix-in

| Field | Value |
|-------|-------|
| Name | ClickHouse mix-in (published as Grafana dashboard `23415`, "Prom-Exporter Instance Dashboard v2") |
| URL | https://github.com/ClickHouse/clickhouse-mixin |
| Grafana dashboard ID | `23415` |
| Commit | `c54f56e0ad91779fe24b900a92629e54e3ed2126` |
| License | **Unconfirmed** — the repository carries no `LICENSE` file. |

Because the mix-in's license is unconfirmed, **none of its `dashboard.json`
content was copied verbatim** into this Apache-2.0 tree. The mix-in was used
**only as a design reference** — i.e. which panels, metrics and PromQL
shapes are mission-critical for ClickHouse observability. Every panel here was
**rebuilt from scratch** on PMM's own dashboard shell. Attribution to Grafana
dashboard `23415` is retained in the `description` field of each dashboard JSON
and in this file.

## Shell source — PMM Apache-2.0 dashboards

The structure (templating cascade, datasource variable, annotations, tags, UID
conventions, layout idioms and `cleanup-dash.py` normalization) is cloned from
PMM's existing, Apache-2.0-licensed dashboards in this repository:

- `dashboards/dashboards/Valkey/*.json` — templating cascade and panel idioms.
- `dashboards/dashboards/PostgreSQL/PostgreSQL_Instance_Summary.json`,
  `PostgreSQL_Instances_Overview.json`,
  `PostgreSQL_Instances_Compare.json` — instance-summary and fleet-view shells,
  cross-linking via data links.

## Dashboards

| File | UID |
|------|-----|
| `ClickHouse_Instance_Summary.json` | `clickhouse-instance-summary` |
| `ClickHouse_Query_Performance.json` | `clickhouse-query-performance` |
| `ClickHouse_Replication.json` | `clickhouse-replication` |
| `ClickHouse_Instances_Overview.json` | `clickhouse-instances-overview` |
| `ClickHouse_Instances_Compare.json` | `clickhouse-instances-compare` |
