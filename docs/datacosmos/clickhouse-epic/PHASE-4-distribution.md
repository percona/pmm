# Phase 4 — Distribution, packaging, tests, docs

**Outcome:** the `clickhouse_exporter` binary ships inside the `pmm-client`
package (RPM + DEB), the ClickHouse exporter and QAN agent are unit-tested, the
integration matrix exercises exporter + QAN across ClickHouse versions, and the
feature is documented.

## Development line (ordered)

**Step 1 — confirm the exporter binary entry point (Phase 1 hand-off).**
Packaging needs a buildable `main` at `agent/cmd/clickhouse_exporter/main.go`
(consistent with `agent/cmd/pmm-agent-entrypoint`). Binary name everywhere is
exactly `clickhouse_exporter` (underscore, like `node_exporter`). If Phase 1
has not produced it, Phase 4 is blocked — flag it.

**Step 2 — `build/Makefile.clickhouse`** *(new — does not exist today)*.
A thin Make include: `build-clickhouse-exporter`
(`go build -o bin/clickhouse_exporter ./agent/cmd/clickhouse_exporter`),
`test-clickhouse-unit` (`go test ./agent/agents/clickhouse/...`),
`test-clickhouse-matrix` (`bash agent/agents/clickhouse/testdata/run-matrix.sh`).
`include` it from the root `Makefile`.

**Step 3 — `build/scripts/build-client-binary`.** The exporter lives in the
`pmm` repo, so no `build-client-source` change. In `build-client-binary`
`main()`, add (after the other exporters):
`gobuild_component "clickhouse_exporter" "pmm" "" "agent/cmd/clickhouse_exporter"`
— hits the generic `go build` branch. `CGO_ENABLED=0` is fine (the
`clickhouse-go/v2` driver is pure Go).

**Step 4 — RPM spec** (`build/packages/rpm/client/pmm-client.spec`). In
`%install`: `install -m 0755 bin/clickhouse_exporter
$RPM_BUILD_ROOT/usr/local/percona/pmm/exporters`. Add a `%changelog` entry. The
`%files` wildcard already covers it.

**Step 5 — DEB packaging.** `build/packages/deb/rules` `override_dh_auto_install`
— copy `clickhouse_exporter`; `build/packages/deb/install` — add
`clickhouse_exporter /usr/local/percona/pmm/exporters/`.

**Step 6 — Docker client image.** Check `build/docker/client/Dockerfile.el9` —
patch only if it enumerates exporters explicitly (it copies the `exporters/`
dir, so likely automatic — verify).

**Step 7 — verify the supervisor path.** The packaged path
`/usr/local/percona/pmm/exporters/clickhouse_exporter` must match the
`agent/config/config.go` `Paths.ClickHouseExporter` default + the supervisor
template (Phase 1) — a mismatch silently breaks launch.

**Step 8 — unit tests, exporter.** `agent/cmd/clickhouse_exporter/main_test.go`
— start the HTTP handler with a mock registry, `GET /metrics`, assert `200` +
`ClickHouseMetrics_*` / `ClickHouseAsyncMetrics_*` names; flag-parsing test.
Expand `config_test.go` for DSN edge cases. (`collector_test.go` with
`go-sqlmock` already exists.)

**Step 9 — unit tests, QAN.** `agent/agents/clickhouse/querylog/querylog_test.go`
— `makeBuckets`, `fingerprint`, `percentile` (Phase 3 deliverables).

**Step 10 — extend `run-matrix.sh`.** Today it runs only `TestClickHouseMatrix`
(the collector). Add:
- `TestClickHouseExporterMatrix` (`agent/agents/clickhouse/exporter_integration_test.go`,
  tag `clickhouse_integration`) — start the real `clickhouse_exporter` binary
  per endpoint, scrape `/metrics`, assert the `ClickHouseMetrics_*` /
  `ClickHouseProfileEvents_*` / `ClickHouseAsyncMetrics_*` families are present.
- `TestClickHouseQANMatrix` (`agent/agents/clickhouse/querylog/qan_integration_test.go`)
  — drive queries, run a QAN cycle, assert buckets.
- Change the `go test -run` filter to
  `-run 'TestClickHouse(Matrix|ExporterMatrix|QANMatrix)'`; build
  `clickhouse_exporter` at the top of `run-matrix.sh` and pass its path via
  `CLICKHOUSE_EXPORTER_BIN`. Keep the `--profile single|cluster` × version loop.

**Step 11 — CI workflow.** A job that runs `test-clickhouse-unit` always and a
reduced `run-matrix.sh` on changes to `agent/agents/clickhouse/**` (mirror the
matrix-override env vars). On the datacosmos fork this complements the
`datacosmos-release.yml` gate.

**Step 12 — documentation.** `.github/instructions/clickhouse.instructions.md`
(or extend `agent.instructions.md`); user docs for `pmm-admin add clickhouse`
under `documentation/docs/`; update `docs/datacosmos/BUILD.md` and the exporter
lists.

## Validation criteria

1. `go build ./agent/cmd/clickhouse_exporter` succeeds (`CGO_ENABLED=0`).
2. After a client build, `bin/clickhouse_exporter` exists and is executable.
3. `rpm -qlp pmm-client-*.rpm` and `dpkg -c pmm-client_*.deb` both list
   `/usr/local/percona/pmm/exporters/clickhouse_exporter`.
4. Installed: binary at that path, mode `0755`, owner `pmm-agent`.
5. `clickhouse_exporter --version` / `--help` work; `--web.listen-address`
   serves `/metrics`.
6. `go test ./agent/agents/clickhouse/... ./agent/cmd/clickhouse_exporter/...`
   passes (no integration tag); `go vet` / golangci-lint clean.
7. The binary path matches the supervisor's expected `exporters/` path.

## Integration tests

See [INTEGRATION-TESTS.md](INTEGRATION-TESTS.md) — IT-4.x: matrix collector
(existing, still green); matrix exporter; matrix QAN; package smoke test
(install the built RPM/DEB, `pmm-admin add clickhouse`, exporter launches,
metrics in VM); end-to-end via `pmm-ui-tests`; upgrade test (previous
pmm-client → new, ClickHouse appears, nothing else disturbed).

## Risks

- **Blocked on Phase 1/3 deliverables** — needs the `clickhouse_exporter` `main`
  package (Phase 1) and the QAN agent (Phase 3). Steps 1, 9 are explicit
  hand-off points; if absent, those steps stub and must be flagged.
- **`build/Makefile.clickhouse` does not exist** — it must be created (Step 2);
  `qan-api2/Makefile.clickhouse` is unrelated.
- **Binary-name consistency** — `clickhouse_exporter` must be identical across
  build script, RPM, DEB, supervisor templates, and config defaults.
- The full Docker/rpmbuild client build is heavy and validated in CI per
  `docs/datacosmos/BUILD.md`.
