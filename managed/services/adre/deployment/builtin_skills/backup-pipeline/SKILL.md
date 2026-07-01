---
name: backup-pipeline
description: Percona_MS_StaleBackup, Percona_MS_StaleUpload, Percona_MS_BackupFailed,
  Percona_MS_StaleUploadLog, Percona_MS_StaleBackupLog — backup jobs not finishing,
  uploads stalled, logical/physical backup failures, MySQL log backup staleness
---

# Backup and upload pipeline (managed service)

## Goal

Determine whether backup or upload is stalled, failed, or blocked by an upstream issue (disk, MySQL blocking, S3 permissions, prior alert). Distinguish threshold tuning (legitimately long runs) from real incidents.

## Workflow

### Alert routing

- **Percona_MS_StaleBackup**: Mydumper/Xtrabackup job exceeded expected completion window (often ~26h since last success) — may block replication on backup server or queue other backups.
- **Percona_MS_StaleUpload**: Upload to object storage not finished in window — often follows failed/stale backup; fix backup first.
- **Percona_MS_BackupFailed**: Mydumper/Xtrabackup exited with errors on backup host.
- **Percona_MS_StaleBackupLog / Percona_MS_StaleUploadLog**: Binary log backup or its upload — shorter default windows (e.g. ~65m for upload); resolve log backup before upload.

1. **Correlated alerts**
   * Use **pmm_list_firing_alerts** if available. If **StaleUpload** follows **StaleBackup** or **BackupFailed**, treat backup recovery as primary; upload often self-heals after a good backup.

2. **Scope in PMM**
   * **pmm-inventory**: map labels to **node_id** / **service_id** for the backup instance and source MySQL service (if named).

3. **PMM metrics and panels**
   * **pmm_observability_map**: `engine=node`, `intent=disk_io`, `service_id` or node-scoped vars for backup host; render disk/free-space panels. **Embed every successful `image_url`**.
   * Scoped backup metrics: **`pmm_list_metric_names`** with prefix from textfile/exporter if known — never global browse.

4. **Logs via PMM**
   * **otel.logs** for backup host around failure time — search for `xtrabackup`, `mydumper`, `s3cmd`, `upload`, `binlog` (ClickHouse-safe time predicates).

5. **Operator steps on backup instance (SSH — Holmes cannot run)**
   * Recent summary logs:
     ```bash
     ssh <backup_instance>
     tail -n 20 /var/log/percona/backups/mydumper.log
     tail -n 20 /var/log/percona/backups/xtrabackup.log
     tail -n 100 /var/log/percona/backups/upload.log
     ```
   * Latest per-run logs:
     ```bash
     ls -alhrt /var/log/percona/backups | egrep "xtrabackup-|mydumper-" | tail -n 2
     less /var/log/percona/backups/xtrabackup-<alias>-<date>.log
     less /var/log/percona/backups/mydumper-<alias>-<date>.log
     ```
   * Stale/hung backup: check processes and growth:
     ```bash
     ps aux | egrep "xtrabackup|mydumper"
     du -cs /path/to/backup/type/day/
     ```
   * MySQL blocking (on source DB, operator):
     ```sql
     SHOW PROCESSLIST;
     ```
     For mydumper stalls, check `lsof` on backup host for open files under backup path:
     ```bash
     sudo lsof -a -c mydumper | grep "/path/to/backup/"
     ```
   * **Binlog backup failure**:
     ```bash
     tail -n 20 /var/log/percona/backups/binlog_puller_<primary_alias>.log
     ps aux | grep mysqlbinlog
     ```
   * **S3 / upload errors**: inspect s3cmd logs:
     ```bash
     ls -alhrt /var/log/percona/backups | egrep "s3cmd-" | tail -n 3
     less /var/log/percona/backups/s3cmd-<alias>-X-<date>.log
     less /var/log/percona/backups/s3cmd-<alias>-M-<date>.log
     less /var/log/percona/backups/s3cmd-<alias>-B-<date>.log
     ```
   * **Retention and disk** (from source runbook):
     ```bash
     cat /home/percona/.config/percona/backup/backup_config.yml | egrep "BINLOG_PURGE_DAYS|MYDUMPER_DAILY_PURGE|MYDUMPER_WEEKLY_PURGE|XTRABACKUP_COPIES|BACKUP_DIR"
     sudo du -hd1 $(grep BACKUP_DIR /home/percona/.config/percona/backup/backup_config.yml | cut -d: -f2 | xargs)
     ```
     Ensure backup mount has roughly **≥ 2.5×** MySQL `datadir` size; expand disk or reduce retention with customer approval.

6. **Manual re-run (operator, screen session)**
   * Upload:
     ```bash
     screen -S INC<ticket>
     sudo su -
     PERCONA_BACKUP_TEXTFILE_COLLECTOR_DIR="/home/percona/pmm/collectors/textfile-collector/low-resolution/" PEX_SCRIPT=upload.py /home/percona/bin/percona-backup --config /home/percona/.config/percona/backup/backup_config.yml
     ```
   * Full backup:
     ```bash
     # mydumper
     PERCONA_BACKUP_TEXTFILE_COLLECTOR_DIR="/home/percona/pmm/collectors/textfile-collector/low-resolution/" PEX_SCRIPT=backup_mydumper.py /home/percona/bin/percona-backup --config /home/percona/.config/percona/backup/backup_config.yml
     # xtrabackup
     PERCONA_BACKUP_TEXTFILE_COLLECTOR_DIR="/home/percona/pmm/collectors/textfile-collector/low-resolution/" PEX_SCRIPT=backup_xtrabackup.py /home/percona/bin/percona-backup --config /home/percona/.config/percona/backup/backup_config.yml
     ```

7. **Threshold tuning**
   * If last successful run historically exceeds default window, adjust alert to **last_success_duration + margin** (e.g. +2–4h for full backup; +1h for binlog upload if large S3 transfers overlap).

## Synthesize Findings

State: backup vs upload vs binlog path; root cause (disk, permissions, hung query, network, cron/driver); whether replication on backup infra was impacted; whether alert threshold needs change vs code/config fix.

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

* **Disk**: add space or purge old backups per agreed retention; edit `backup_config.yml` and remove oldest copies only after approval.
* **S3 permissions**: fix IAM/bucket policy; verify with provider CLI per internal backup documentation.
* **Hung backup**: after log review, kill only per procedure; escalate if no progress, no FD activity, no size growth.
* **Auth**: fix `.my.cnf` / backup user on primary; test login manually.
* **Cron/driver**: verify `/etc/cron.d/percona_crons` and backup documentation for driver process.
