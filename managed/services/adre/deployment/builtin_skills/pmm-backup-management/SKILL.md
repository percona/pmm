---
name: pmm-backup-management
description: "pmm_backup_error, pmm_mongodb_pbm_agent_down, pmm_mongodb_pbm_backup_failed, pmm_mongodb_pbm_backup_stale, pmm_mongodb_pbm_backup_duration, pmm_mongodb_pbm_backup_size: PMM Backup Management. Failed, stale or slow MySQL (xtrabackup) and MongoDB (PBM) backups, storage locations, restores."
---

# PMM Backup Management

## When to use
PMM backup alerts, or questions about backups and restores made from PMM (Backup > All backups / Scheduled backup jobs / Restores / Storage locations). For backups made outside PMM only the PBM metrics below apply.

## How PMM backups work
- pmm-managed sends a job to the service's pmm-agent, which runs the tool on the database host. A disconnected pmm-agent means no backup.
- **MySQL**: physical, snapshot, S3-compatible location only (no mysqldump, no PITR). pmm-agent pipes `xtrabackup --backup --compress --stream=xbstream` into `xbcloud`. Needs `xtrabackup`, `xbcloud` (same version) and `qpress` on the agent's PATH, an xtrabackup version compatible with the server, MySQL run by systemd as `mysql`/`mysqld` with datadir `/var/lib/mysql` writable by pmm-agent, `BACKUP_ADMIN` on 8.0+, one MySQL per node.
- **MongoDB**: Percona Backup for MongoDB (PBM >= 2.0.1), `pbm-agent` on every replica set member, service registered with a cluster label. Logical or physical; snapshot or PITR (PITR = logical + S3). Location: S3 or a filesystem path mounted at the same path on all members.
- Backup Management must be on: Configuration > Settings > Advanced settings, or `PMM_ENABLE_BACKUP_MANAGEMENT`.

## Steps
1. `pmm_list_firing_alerts()`: labels `artifact_name`, `service_id` (pmm_backup_error) or `cluster`, `name`, `node_name` (PBM).
2. PMM artifacts (instant query): `pmm_managed_backups_artifacts{status="error"}` gives one series per failed artifact with `artifact_name`, `service_name`, `artifact_vendor`, `type` (on_demand / scheduled), `data_model`, `mode`. Overview: `count by (service_name, status) (pmm_managed_backups_artifacts)`; statuses: pending, in_progress, paused, success, error, deleting, failed_to_delete, cleanup_in_progress. The metric has no timestamps; for "when was the last good backup" use the Created column in All backups.
3. MongoDB PBM (mongodb_exporter):
   - agents: `mongodb_pbm_agent_status{self="1"}` per node (0 OK, 1 error, 2 lost);
   - config: `mongodb_pbm_cluster_backup_configured`, `mongodb_pbm_cluster_pitr_backup_enabled`;
   - history by `name`, `status` (done, error, canceled...): `mongodb_pbm_backup_size_bytes`, `mongodb_pbm_backup_duration_seconds`, `mongodb_pbm_backup_last_transition_ts`;
   - age of last good backup: `time() - max by (cluster) (max_over_time(mongodb_pbm_backup_last_transition_ts{status="done"}[30d]))`.
   Dashboard `mongodb-backup-details` (panel 12 Backup Agent Status, 6 Backup history, 5 Backup Duration): render one only when it shows the problem.
4. Agent: `pmm_list_agents(service_id)` and the pmm-agent `connected` flag; disconnected -> exporter-scrape-failures.
5. Logs: each artifact has its own log in PMM (All backups > the artifact's logs; API `GET /v1/backups/{artifact_id}/logs`). No tool reads it: ask the user for the last lines. If the host ships logs, one `otel` query for the window with `Body ILIKE '%xtrabackup%'` / `'%xbcloud%'` / `'%pbm%'`.
6. Duration or size alerts: `pmm_metrics_snapshot` on the backup size/duration over 30 days, plus node `disk_io` / `network` routes during the backup window.

## Interpretation
- `pmm_backup_error` stays firing while the failed artifact exists (the metric is built from the artifact list and is always 1). It does not mean the latest backup failed; check for a newer success.
- `pmm_mongodb_pbm_backup_failed` likewise lasts while the errored backup is in PBM's list.
- MySQL failures, most common first: tools missing (`lookpath=xtrabackup` / `xbcloud` / `qpress`) or versions differ ("xtrabackup version != xbcloud version", "incompatible xtrabackup"); S3 credentials or permissions (List, Put, Get, Delete needed; the artifact log carries xbcloud's stderr); MySQL not under systemd or a different datadir; missing `BACKUP_ADMIN`.
- PBM agent status 2 (lost): `pbm-agent` stopped or cannot reach mongod. 1 (error): agent reports an error, usually storage unreachable or config mismatch. Backups fail while any member's agent is not OK.
- Stale (`pmm_mongodb_pbm_backup_stale`, default 24 h): schedule removed or failing, or only PITR running. Duration (default 3600 s) and size (default 1 GiB) are thresholds; growth that follows data growth is a tuning issue, not a failure.
- Restore limits: sharded-cluster artifacts cannot be restored from the UI; point-in-time restore exists only for MongoDB logical backups on S3.
- Kubernetes Operators and RDS/Azure: PMM Backup Management does not drive them (the Operator's backup resources or provider snapshots do).

## Remediation (self-managed DB hosts; PMM runs none of these)
- MySQL tooling: `xtrabackup --version; xbcloud --version; which qpress` on the DB host; install the Percona XtraBackup package matching the server major version (8.0 or 8.4) and qpress; `GRANT BACKUP_ADMIN ON *.* TO 'pmm'@'localhost';` (use the user pmm-agent connects with).
- PBM: `systemctl status pbm-agent`, `journalctl -u pbm-agent --since "-2h"`, `pbm status`, `pbm logs -s E -t 50` (read-only). PMM owns the PBM configuration: avoid other manual `pbm` commands.
- Storage: fix bucket, endpoint, region or keys in Backup > Storage locations; PMM can validate a config (`POST /v1/backups/locations:testConfig`).
- After the fix: run an on-demand backup, confirm success, then delete the failed artifact (All backups > Delete) to clear `pmm_backup_error`. Scheduled jobs can retry on their own (`retries`, `retry_interval` on the schedule).
