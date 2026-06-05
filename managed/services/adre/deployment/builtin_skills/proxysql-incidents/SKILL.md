---
name: proxysql-incidents
description: Percona_MS_ProxySQLIsNotRunning, Percona_MS_HostGroupDoesNotHaveOnlineServer,
  Percona_MS_ProxySQL_IPController — ProxySQL process down, empty hostgroup, ProxySQL-side
  VIP/controller
---

# ProxySQL incidents

## Goal

Determine whether ProxySQL is down, backends are absent from a hostgroup, or ProxySQL VIP controller failed—using PMM and inventory, then operator shell on ProxySQL and monitor hosts.

## Workflow

### Alert routing

| Alert | Focus |
|--------|--------|
| **Percona_MS_ProxySQLIsNotRunning** | proxysql process / exporter up |
| **Percona_MS_HostGroupDoesNotHaveOnlineServer** | No healthy backend in hostgroup |
| **Percona_MS_ProxySQL_IPController** | VIP script on monitor for ProxySQL pair |

1. **Scope**
   * **pmm-inventory** → **service_id** for ProxySQL service and backend MySQL services.

2. **PMM metrics and panels**
   * **No ProxySQL observability-map route yet** — use **`pmm_list_dashboard_panels`** on the ProxySQL dashboard UID, then **`pmm_render_grafana_panel`**. **Embed every successful `image_url`** in the answer.
   * Scoped metrics only: **`pmm_list_metric_names`** with prefix `proxysql_` (max 50) or **`pmm_discover_series_labels`** with `service_id` — never global `__name__` browse.

3. **Logs**
   * **otel.logs** for ProxySQL container/host around failure.

4. **Branch: ProxySQL not running**
   * Operator:
   ```bash
   sudo systemctl status proxysql
   sudo systemctl start proxysql
   sudo journalctl -u proxysql -e
   ```
   * If crash loop: disk, config (`/etc/proxysql.cnf` or datadir), SELinux.

5. **Branch: hostgroup has no online server**
   * In ProxySQL admin (operator):
   ```sql
   SELECT * FROM runtime_mysql_servers;
   SELECT * FROM runtime_mysql_replication_hostgroups;
   SHOW PROXYSQL STATUS;
   ```
   * Check `mysql_servers` / Galera / replication hostgroup rules; confirm backends up (**mysql-availability** skill).
   * Common fixes: restore mysqld on backends; fix `weight`, `status=OFFLINE_HARD`; reload:
   ```sql
   LOAD MYSQL SERVERS TO RUNTIME;
   SAVE MYSQL SERVERS TO DISK;
   ```

6. **Branch: ProxySQL IP controller**
   * On **monitor** host:
   ```bash
   ps -ef | grep -i ip_controller
   less ~/.local/percona/ip_controller_<proxysql_cluster>.log
   ```
   * If ProxySQL nodes down, restore ProxySQL first; verify cron and GAS version per internal docs (see **Percona_MS_ProxySQL_IPController** in Node alert runbooks).

## Synthesize Findings

Distinguish ProxySQL layer vs all backends unhealthy vs VIP/script stall. Name hostgroup id and affected backends from runtime tables.

### Environment availability matrix (shared)

Before prescribing a command, decide **which surfaces this deployment exposes**. PMM tools (inventory, observability map, metrics snapshot, QAN on `pmm.metrics`, `otel.logs`, Grafana render) are available in **every** environment — always try those first. Host shell, DB superuser, and config-file edits are **not** universal.

| Surface | Self-managed (VM / bare-metal / Docker) | Kubernetes / Operator (PSMDB, PXC, PG operator) | Managed cloud (RDS / Aurora / Cloud SQL / Atlas) |
|---|---|---|---|
| PMM metrics / QAN / observability map | ✅ | ✅ | ✅ (exporter may be **remote/cloud-mode**) |
| `otel.logs` (PMM-collected) | ✅ | ✅ | ⚠️ partial — DB logs often only in the cloud console |
| Host shell (`journalctl`, `grep /var/log/...`, `openssl`, `df`) | ✅ operator | ⚠️ via `kubectl exec` into the pod, not the node | ❌ none |
| DB shell (`psql`, `mongosh`, `mysql`) | ✅ operator | ✅ operator via `kubectl exec` | ✅ but **no superuser** (rds_superuser / atlasAdmin only) |
| `ALTER SYSTEM` / `mongod.conf` / `my.cnf` edits | ✅ | ❌ — change the **Custom Resource**, not the file (operator reverts file edits) | ❌ — change a **parameter group / cluster config**, then reboot/apply |
| `pg_terminate_backend`, `KILL`, force-primary, oplog resize | ✅ with approval | ✅ with approval | ⚠️ often restricted or wrapped by a cloud API |

**Routing rules:**

- **Always lead with PMM/QAN/`otel.logs`.** They work everywhere and need no host/DB access.
- **Gate host-shell and superuser steps.** Label them **"operator, self-managed / k8s only"**. On managed cloud, **skip** them and use the cloud-native equivalent (console logs, Performance Insights / Cloud Monitoring, parameter groups, provider CLI).
- **Config changes:** on **operator** deployments edit the **CR** (e.g. `PerconaServerMongoDB`, `PerconaXtraDBCluster`, `PerconaPGCluster`) and let the operator roll it out; on **managed cloud** edit the **parameter group / flag** and apply per the provider; only on **self-managed** edit the file directly and reload/restart.
- **Detect the environment** from inventory (`service_type`, `node_model`/cloud labels, address like `*.rds.amazonaws.com`, `*.mongodb.net`) before recommending a step. If unknown, state the assumption and give the **PMM-only** path plus the operator path as an option.
- **Never tell the user to run a command their environment can't execute** (e.g. `journalctl` on RDS). Offer the reachable alternative instead.
### PMM observability and panel embed rules (shared)

**Evidence hierarchy:** inventory → **`pmm_observability_map`** → **`pmm_metrics_snapshot`** on panel `expr` → **`pmm_render_grafana_panel`** (embed PNGs, best-effort) → QAN + MySQL EXPLAIN → scoped fallbacks (`pmm_discover_series_labels`, `pmm_list_metric_names` prefix max 50) → **`execute_prometheus_range_query`** at most one per turn, last resort.

**Render failures are non-blocking:** 502, timeout, blank PNG, or curl error on render must **never** skip snapshot, QAN, EXPLAIN, or final synthesis. State **`rendered M/N`**; deliver analysis from snapshot + QAN even when **`rendered 0/N`**.

**FORBIDDEN:** unfiltered `GET /api/v1/label/__name__/values`, full dashboard JSON to the LLM, guessing panel IDs or metric names.

**Observability map:** `GET /v1/grafana/observability-map` via **`pmm_observability_map`**. Pass **`engine`** (`mysql`, `postgresql`, `mongodb`, `valkey`, `node`), **`intent`**, **`service_id`**. Use returned **`primary.dashboard_uid`**, **`panels[].id`**, **`panels[].expr`**. Fall back to **`pmm_list_dashboard_panels`** only when the map warns or intent is unknown (e.g. ProxySQL — no map route yet).

**Metrics snapshot:** **`pmm_metrics_snapshot`** on panel `expr` — returns server-computed **stats** (min/max/mean/median/p25/p75/p95/p99), **change_points**, **anomalies**; not raw matrices. Run **immediately after map** — does not require render. `start`/`end`: RFC3339 or Unix (not `now-24h`). Requires ADRE enabled.

**Render panels (best-effort when this skill uses Grafana):**
1. Inventory first: **`service_id`**, **`node_id`**, **`agent_id`**, **`service_name`**, **`node_name`**, **`version`** → pass all as `var_*` overrides to **`pmm_render_grafana_panel`**. Missing **`agent_id`** often causes blank/timeout renders.
2. Tool: **`POST /v1/grafana/render/resolve`** only (via **`pmm_render_grafana_panel`**).
3. Render **sequentially**; `from`/`to` without quotes (e.g. `now-6h`, `now`). Retry once with shorter window or fixed vars on failure — then **continue snapshot/QAN/EXPLAIN**.
4. **Embed every successful render** in the user-visible answer:
   ```markdown
   ![Panel title](/v1/grafana/render/blob/{hash}.png)
   [Open in Grafana](dashboard_url)
   ```
   Use **`image_url`** and **`dashboard_url` exactly** from tool JSON. Do not skip images for successful renders; do not use `dashboard_url` as img src, do not rebuild URLs.
5. State **`rendered M/N`** if any panel failed; **do not abort** the investigation.

**Scoped metric fallback (when map warns):** **`pmm_discover_series_labels`** with `service_id` + **`metric_prefix`** from map fallback, or **`pmm_list_metric_names`** with prefix only (max 50).

**QAN (database services):** For workload-correlated incidents, query **`pmm.metrics`** top slow patterns by **`service_id`** — see **`general`** skill for query shape and MySQL EXPLAIN Step 3b when applicable. **Always run** even when render fails.
## Recommended Remediation Steps

* Start/repair ProxySQL service; fix config; ensure disk space.
* Hostgroup: align hostgroup IDs with replication or Galera rules; bring at least one `ONLINE` backend; save to disk.
* IP controller: repair script/cron; upgrade gas-tools; restore ProxySQL quorum.
