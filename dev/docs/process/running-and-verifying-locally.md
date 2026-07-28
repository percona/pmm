# Running and verifying PMM locally

Referenced from [`AGENTS.md`](../../../AGENTS.md) (Definition of Done / workflow). Read this when a change is **user-visible** (metric, dashboard, API, exporter, UI) — unit tests alone are not enough; you must run PMM on a live server and verify against real data.

## Build, deploy, iterate

PMM development uses a **single "fat" Docker container** (`perconalab/pmm-server:3-dev-container`) that runs every server component (pmm-managed, grafana, victoriametrics, clickhouse, postgresql, qan-api2, vmproxy, pmm-agent, nginx, vmalert) under `supervisord`. The repo is bind-mounted into the container and Go/UI binaries are rebuilt on the host mount and hot-swapped into the running services. Start it **once** and reuse it — do **not** rebuild the image on every code change.

### Starting the environment
- Docker (with the compose plugin) is required. If `docker info` fails, the daemon isn't running — start it with `sudo dockerd > /tmp/dockerd.log 2>&1 &` (depending on the Docker version, you might need `/etc/docker/daemon.json` to set `"storage-driver": "fuse-overlayfs"` and `"features": {"containerd-snapshotter": false}`). To run `make`/`docker` without `sudo`, add your user to the `docker` group (`sudo usermod -aG docker "$USER"` then `newgrp docker`) rather than loosening socket permissions.
- Use the **dev** env file, not the stable one: `cp .env.dev.example .env` (sets the `3-dev-container` image and `GO_VERSION`). `.env.example` points at the release image `percona/pmm-server:3`, which lacks the dev toolchain.
- `make env-up` pulls the image and starts the container (`--wait` until healthy). The UI/API is then at `https://localhost` (self-signed cert; default login). `make env-down` / `make env-remove` to stop it.
- Before the first build, fix git's bind-mount ownership complaint inside the container: `docker exec -u root pmm-server bash -lc "git config --global --add safe.directory /root/go/src/github.com/percona/pmm"`.

### Build/test/lint targets run as ROOT
The `run-*`, `test-*`, `check`, and `build-ui` targets install to `/usr/sbin`, call `supervisorctl`, and use the root-owned Go module-cache volume — so run them as root via **`make env-root TARGET=<t>`** from the host. The default `make env` runs as the `pmm` user and fails on these (`/usr/sbin` is read-only for it, and the module cache isn't writable); use plain `make env` only for read-only tasks like a shell. Use the `-ci` variants in automation — the non-`-ci` targets (`run-managed`, `run-qan`, …) end with `tail -f` and never return. `golangci-lint` isn't preinstalled: run `make env-root TARGET=init` once to install `bin/golangci-lint`.

| Step | Command | Notes |
|------|---------|-------|
| Start the server (once) | `make env-up` | Slow on first run (pulls the dev image); fast afterwards. `make env-up-rebuild` only when you need a fresh image. |
| Shell in the container | `make env` | Runs inside `pmm-server` as the `pmm` user (read-only tasks). |
| Hot-swap pmm-managed after a Go change | `make env-root TARGET=run-managed-ci` | Rebuilds only the binary and restarts it via `supervisorctl` — **no image rebuild**. Returns when done. |
| Hot-swap other Go services | `run-agent-ci`, `run-qan-ci`, `run-vmproxy-ci`, or `run-all` | Same `env-root` pattern per service. |
| Unit tests (shared/API packages) | `make env-root TARGET=test-common` | No external deps. `managed`/`api-tests` suites need the running server + internal PostgreSQL. |
| API integration tests | `make env-root TARGET=api-test` | Requires the server to be up. |
| Lint (changed files) | `make env-root TARGET=check` | buf lint + golangci-lint + go-sumtype; only lints changes since `merge-base main HEAD`. |
| Build + deploy the UI | `make env-root TARGET=build-ui` | Builds the UI into Grafana. `run-ui` starts Vite HMR (port 5173); the image ships a pre-built UI, so `https://localhost` works without rebuilding. |
| DB shell (pmm-managed) | `make env TARGET=psql` | |

**The iterate loop** (no image rebuild, container stays up):

1. `make env-up` — start the server once.
2. Change code.
3. `make env-root TARGET=run-managed-ci` — rebuild + hot-swap the affected binary.
4. `make env-root TARGET=api-test` (or `test-common`) — run the smallest test set.
5. On failure: read the error, fix, and repeat from step 3. **Do not re-run `env-up` each iteration.**

Use the `-ci` variants: `run-managed` (without `-ci`) tails the log and blocks; `run-managed-ci` returns.

**Server logs** live in `/srv/logs/` on `pmm-server` (each `run-*-ci` truncates its log first, so each iteration starts clean):

```bash
docker exec pmm-server tail -n 200 /srv/logs/pmm-managed.log   # also: pmm-agent.log, qan-api2.log, vmproxy.log
```

**Accessing it:** open **https://localhost** (accept the self-signed cert) and log in with PMM's default **`admin`/`admin`**. Hit the REST API directly — paths are in `api/swagger/swagger.json` (e.g. `curl -k -u admin:admin https://localhost/v1/...`). Host ports (defaults; override via `PMM_PORT_*` in `.env`): `443` (UI/API), `5432` (PostgreSQL), `9090` (VictoriaMetrics), `8123`/`9000` (ClickHouse), `5173` (Vite HMR, `make run-ui`), `2345` (Delve).

The built-in `pmm-agent` already self-monitors the server node and internal PostgreSQL. Add more monitored services from the UI (Inventory → Add Service), with `pmm-admin add ...` inside the container, or via pmm-qa (below).

Machine-specific paths (pmm-qa checkout, screenshot dir) belong in a gitignored `AGENTS.local.md` — **never secrets or tokens** (keep those in env vars or a secret store).

## Registering databases to test against

Unit and API tests don't need a real database, but validating a **metric, exporter, dashboard, or QAN** change does — you need a live monitored instance producing real data. Spin one up with the [pmm-qa](https://github.com/percona/pmm-qa) framework and register it against your local server:

```bash
cd <pmm-qa-checkout>/qa-integration/pmm_qa   # path is machine-specific — keep it in AGENTS.local.md
./virtenv/bin/python pmm-framework.py --verbose \
  --pmm-server-password=admin --client-version=3-dev-latest \
  --database PS=8.0
```

**Test the version matrix.** When a change is version-sensitive (a new/changed metric, an exporter query, a parser, or anything that differs across DB releases), register the **oldest and newest supported versions** of the affected database and validate on **both** — the change must work on the newer version **and not regress** on the older one:

```bash
./virtenv/bin/python pmm-framework.py --verbose --pmm-server-password=admin \
  --client-version=3-dev-latest --database PS=5.7   # older supported
./virtenv/bin/python pmm-framework.py --verbose --pmm-server-password=admin \
  --client-version=3-dev-latest --database PS=8.0   # latest supported
```

**Don't invent version numbers** — the valid versions per engine live in pmm-qa's `scripts/database_options.py` (imported into `pmm-framework.py` as `database_configs`; e.g. Percona Server = `5.7`, `8.0`, `8.4`, default `8.0`). Check there for what actually exists rather than assuming a release is out.

Confirm each instance appears as a monitored target (Inventory, and the metric shows up in VictoriaMetrics) **before** you start verifying.

## Verify by evidence — don't assume

For any user-visible change, "it compiles and unit tests pass" is **not done** — reproduce the problem, deploy the fix to a running server with real data, and confirm with evidence. Untested code is unfinished code.

**1. Understand and reproduce first.** Read the ticket fully. Restate in your own words what's broken vs. expected and which component it touches (exporter, pmm-managed, dashboard, API, UI). Reproduce the current (broken) behavior on a running server **before** changing code, so you can prove the fix later.

**2. Iterate until it actually works.** After each change, hot-swap the binary ([iterate loop](#build-deploy-iterate)), re-run the relevant tests, and re-check behavior on the running server. If anything is wrong — fix, redeploy, re-test. Don't stop at the first green compile.

**3. Verify by evidence — do every item that applies, for every version tested:**

- **Dashboards:** derive the **full** list of dashboards the change affects from the changed metric/feature (e.g. MySQL Instance Summary, MySQL Instance Overview, QAN, and any dashboard rendering the changed metric) — list them explicitly, then open **each** for **every** instance. Screenshot it, **open the screenshot**, and confirm panels are populated with correct values (non-empty, non-`NaN`). Don't verify just one dashboard.
- **Metrics / API:** query the underlying data directly, not just the rendered panel — VictoriaMetrics' Prometheus-compatible API on the server (e.g. `GET /prometheus/api/v1/query?query=<metric>`) and/or the pmm-managed REST API. Confirm the value is correct for each version.
- **Logs:** check `pmm-managed`, `vmagent`, and the relevant exporter (e.g. `mysqld_exporter`) for scrape errors and `unsupported`/parse warnings — for every version. Server logs are in `/srv/logs/` on `pmm-server`; exporter and vmagent logs live on the monitored-node containers created by pmm-qa (`docker logs <node-container>`).

**4. End with a proof section.** State what was broken, what you changed and why, and give side-by-side evidence (screenshots + metric/API values + log excerpts) for every version. When a change is version-sensitive, call out explicitly that the old version still works (no regression) and the new version now works.

> Red flags — none of these count as verification: "it looks right", "it's a small change", "I'll verify later". If you didn't run it and look at the result, it isn't done.
