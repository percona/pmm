# ClickHouse epic — Integration-test strategy

This is the end-to-end test plan that proves each phase works and feeds the
**release gate**: no `v3-*` tag ships unless these pass locally
(`agent/agents/clickhouse/testdata/run-matrix.sh`, extended per phase).

## Test infrastructure

- **Harness**: `agent/agents/clickhouse/testdata/run-matrix.sh` +
  `docker-compose.matrix.yml` — single-node and 2-node cluster (Keeper)
  topologies. Build tag `clickhouse_integration`.
- **Version matrix**: ClickHouse `26.3, 25.8, 25.3, 24.8, 24.3` × `{single,
  cluster}`. Phase 1 adds a `< 22.6` version (no native endpoint) to exercise
  the exporter-only path.
- **Fixtures**: a `<prometheus>`-enabled ClickHouse config and a disabled one
  (Phase 1 native/exporter probe); seed queries for QAN (Phase 3).
- Each combination is brought up, tested, torn down (`down -v`) in turn.

## Phase 1 — metrics, inventory, API, pmm-admin

| ID | Scenario | Asserts |
|---|---|---|
| IT-1.1 | exporter mode end to end (`<prometheus>` disabled) | `add clickhouse --metrics-source=exporter` → result has `clickhouse_exporter`; agent row `agent_type=clickhouse_exporter`; supervisor process `RUNNING`; `up{agent_type="clickhouse-exporter"}==1` in VM; `ClickHouseMetrics_*` series present |
| IT-1.2 | native mode end to end (CH 22.6+) | `--metrics-source=native` → result has a native (external) exporter `:9363/metrics`; **no** `clickhouse_exporter` process; VM scrape job targets `:9363`; `up==1` |
| IT-1.3 | auto-probe → native | `<prometheus>`-enabled server, `--metrics-source=auto` → native chosen; `clickhouse_options.native_endpoint=true` |
| IT-1.4 | auto-probe → exporter | `<prometheus>`-disabled server, `auto` → exporter chosen |
| IT-1.5 | forced native fails fast | `<prometheus>`-disabled, `--metrics-source=native` → `FailedPrecondition`; **no** service/agent rows persisted (rollback) |
| IT-1.6 | connection check | wrong password, no `--skip-connection-check` → auth error + rollback; with the flag → service added |
| IT-1.7 | remove | `remove clickhouse` deletes service + cascades exporter; inventory clean; VM job dropped |
| IT-1.8 | version matrix | run IT-1.1..1.4 across CH `<22.6` (exporter only) and `≥22.6` (both), single + cluster; chosen mode matches server capability |

## Phase 2 — dashboards

| ID | Scenario | Asserts |
|---|---|---|
| IT-2.1 | plugin build | CI `dashboards.yml` `build` artifact contains all 5 `ClickHouse/` JSONs |
| IT-2.2 | live provisioning | PMM Server + rebuilt pmm-app + a monitored ClickHouse → all 5 dashboards appear, load with no "not found"/datasource error |
| IT-2.3 | metric binding | with ClickHouse under load, headline panels of each dashboard render data (query Grafana `/api/dashboards/uid/<uid>`, POST targets to `/api/ds/query`, assert non-empty series) |
| IT-2.4 | template cascade | `$environment → $cluster → $node_name → $service_name` filters correctly, single-node and cluster |
| IT-2.5 | uid regression | the 5 UIDs present in the dashboard-inventory snapshot test |

## Phase 3 — QAN

| ID | Scenario | Asserts |
|---|---|---|
| IT-3.1 | basic bucket | `log_queries=1`; 5 SELECTs varying `LIMIT` literals; one interval → exactly 1 bucket, `NumQueries==5`, literal-free fingerprint, `m_read_rows_sum>0`, `MQueryTimeSum>0` |
| IT-3.2 | distinct fingerprints | a SELECT + an INSERT in one period → 2 buckets, distinct `Queryid`, `query_kind` SELECT/INSERT |
| IT-3.3 | incremental, no double-count | 3 queries → interval 1 `NumQueries==3`; 2 more → interval 2 `==2`; no `query_id` counted twice |
| IT-3.4 | boundary second | queries straddling a watermark-second boundary → sum across intervals == actual `query_log` row count |
| IT-3.5 | error query | invalid query → `NumQueriesWithErrors==1`, `Common.Errors` has the CH exception code |
| IT-3.6 | metric accuracy | heavy query; read `read_rows/read_bytes/memory_usage` directly by `query_id` → bucket sums equal within float tolerance; `min==max==sum` for a single-execution group |
| IT-3.7 | lazy table / `log_queries=0` | fresh CH or `log_queries=0` → agent `WAITING`, descriptive log, no panic; transitions to `RUNNING` after enabling |
| IT-3.8 | version columns | across the CH version matrix, `DESCRIBE TABLE` column detection adapts; missing columns degrade to 0 |
| IT-3.9 | qan-api2 round-trip | a ClickHouse bucket ingests via the INSERT and is queryable by `agent_type='qan-clickhouse-querylog-agent'` |

## Phase 4 — distribution

| ID | Scenario | Asserts |
|---|---|---|
| IT-4.1 | matrix collector | existing `TestClickHouseMatrix` still green across all versions × topologies |
| IT-4.2 | matrix exporter | `TestClickHouseExporterMatrix` — packaged-equivalent binary scrapes each endpoint; the `ClickHouseMetrics_*` / `ClickHouseProfileEvents_*` / `ClickHouseAsyncMetrics_*` families and a `ClickHouseStatusInfo_*` version series are present; cluster also asserts replication metrics |
| IT-4.3 | matrix QAN | `TestClickHouseQANMatrix` — drive known queries, run a QAN cycle, assert buckets (`fingerprint`, `num_queries`) |
| IT-4.4 | package smoke | clean EL9 + Debian: install built `pmm-client`; `pmm-admin add clickhouse`; supervisor starts `clickhouse_exporter`; metrics land in VM |
| IT-4.5 | end-to-end | `pmm-ui-tests` flow registers a ClickHouse service; metrics flow server-side; dashboards bind |
| IT-4.6 | upgrade | previous `pmm-client` (no ClickHouse) → upgrade to new package → `clickhouse_exporter` appears, existing exporters/`pmm-agent.yaml` untouched |

## Unit tests (per phase)

- **Phase 1**: converters, `add_clickhouse_test.go`, the native-probe logic.
- **Phase 3**: `makeBuckets`, `fingerprint` (table-driven CH SQL),
  `percentile` (n=1, n=2, all-equal); `managed/services/qan/client_test.go`
  `fillClickHouse`.
- **Phase 4**: `clickhouse_exporter/main_test.go` (`/metrics` 200 + metric
  names, flag parsing); expanded `config_test.go` (DSN edge cases).

## Release gate

`run-matrix.sh` must exit 0 (all of IT-1.8, IT-3.8, IT-4.1–4.3 across the
version/topology matrix) before any `v3-*` tag is pushed. The
`datacosmos-release.yml` workflow builds/publishes; the matrix is the local
pre-tag gate.
