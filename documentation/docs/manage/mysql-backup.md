# MySQL Backups

!!! warning "Tech Preview"
    This feature is not production-ready. Use for testing and feedback only.

MySQL Backup lets you run and schedule MySQL backups against MySQL services in your inventory, restore from those backups, and track job status and execution history.

It is part of PMM's [Management framework](index.md) (SEP) integration, accessible from **Apps > MySQL Backups** in the sidebar.

The legacy PMM MySQL backup feature under **Backups > All backups** continues to work alongside this app. Backups created in one are not visible in the other.

## Supported backup types

| Type | Tool | Best for |
|---|---|---|
| **XtraBackup** | `xtrabackup`, `mariadb-backup`, or `innobackupex` | Physical hot backups of large datasets with minimal locking. Supports full and incremental backups. Must run directly on the database host. |
| **Mydumper** | `mydumper` | Logical SQL dumps with selective database and table restores. Can run from a remote executor host. |
| **Binlog** | `mysqlbinlog` | Continuous binary log capture. Use alongside a base XtraBackup or Mydumper backup for point-in-time recovery — not a standalone complete backup. |

MariaDB is supported via the `mariadb-backup` binary.

## Before you start

### Enable Nomad on PMM Server

The MySQL Backups app dispatches tasks via Nomad. To enable Nomad, start PMM Server with both `PMM_ENABLE_NOMAD=1` and `PMM_PUBLIC_ADDRESS` set. See [Configure Nomad](../reference/nomad.md).

<!-- VERIFY: is any additional enablement step required beyond Nomad, e.g. a settings.yaml APPS entry or PMM UI toggle? -->

### Install required tools on the execution host

Install the tool for your backup type and make sure it is available on `$PATH`:

| Backup type | Required tool |
|---|---|
| XtraBackup | `xtrabackup`, `mariadb-backup`, or `innobackupex` |
| Mydumper | `mydumper` |
| Binlog | `mysqlbinlog` |

XtraBackup requires **root** on the execution host. Mydumper and Binlog do not.

For XtraBackup, use a version that matches your MySQL version:

| MySQL version | Percona XtraBackup version |
|---|---|
| 5.5, 5.6, 5.7 | PXB 2.4.x |
| 8.0.0–8.0.33 | PXB 8.0.x (same version or newer) |
| 8.0.34+ | PXB 8.0.34+ |
| 8.1.x, 8.2.x, 8.3.x | Matching PXB version |
| 8.4.x | Any PXB 8.4.x |

### Configure database credentials on the host

SEP reads MySQL credentials from `~/.my.cnf` or `~/.mylogin.cnf` on the executor host. The backup task does not prompt for a password — this file must exist and be readable before running a backup.

For XtraBackup on MySQL 8.0+, the MySQL user must have the `BACKUP_ADMIN` privilege.

<!-- VERIFY: privilege requirements for Mydumper and Binlog -->

### Prepare a backup directory

Backups are written to a local directory on the execution host. Create the directory and confirm it is writable before creating a task. Set the path per task in the **Backup directory** field.

### Sync your MySQL service

The MySQL service must appear in the SEP inventory. SEP syncs from PMM — if a service is registered in PMM but a sync has not run, it will not appear in the backup form. Trigger a sync from **Inventory** if needed.

## Storage

Backups are written locally to the execution host by default. You can optionally upload to one or more remote destinations, configured per task:

| Provider | Required field |
|---|---|
| S3-compatible storage | S3 bucket |
| Google Cloud Storage | GCS bucket |
| Rsync | Rsync destination path |

You can select multiple upload providers simultaneously.

### Retention

Without retention configured, backups accumulate until deleted manually. Set retention per task:

| Backup type | Retention options |
|---|---|
| Mydumper | Daily purge (days), Weekly purge (weeks) |
| XtraBackup | Number of copies to keep |
| Binlog | Purge after (days) |

## Compression and encryption

### Compression

Enable **Compress backup data** and select an algorithm. Available algorithms vary by backup type:

| Backup type | Supported algorithms |
|---|---|
| XtraBackup | zstd, lz4, quicklz |
| Mydumper | gzip, zstd |

### Encryption

Two independent GPG encryption modes are available:

- **Encrypt backup** — encrypts the backup in place during the run. Combine with **Encrypt using tmpdir** to write to a temporary directory during encryption.
- **Encrypt after backup completes** — GPG-encrypts the finished backup as a post-run step. Mutually exclusive with **Encrypt using tmpdir**.

Both modes require setting an **Encryption recipient** (GPG key or recipient ID).

XtraBackup also supports **AES-256 encryption** via a keyfile, configured in the **AES-256 key file path** field.

## About execution hosts

The execution host is the Nomad agent that runs the backup task. The backup type determines where it must run:

- **XtraBackup** — the executor must be the database host itself. The task always connects to `localhost`.
- **Mydumper** — the executor can be any host with network access to the database.
- **Binlog** — the executor can be any host with network access to the database. Use **Alternative binlog host** to stream logs from a specific source host.

For remote or cloud-hosted databases, select an executor host that has network access to the target.

## Run a MySQL backup

To run a MySQL backup:
{.power-number}

1. Go to **Apps > MySQL Backups** in the sidebar.
2. Click **+ New MySQL Backup**.
3. Enter a task name, select the **backup type**, and select the **execution host**. For XtraBackup, the execution host must be the host running the MySQL service.
4. Under **Upload**, select one or more **Upload providers** and fill in the destination fields if uploading off-host.
5. Optionally configure compression, encryption, or retention in the relevant sections of the form.
6. Optionally check **Alert on failure** to receive an alert if the backup task fails.
7. Click **Run** to start immediately, or set a schedule and click **Schedule**.

Completed XtraBackup and Mydumper runs are recorded in the backup catalog with their location, upload destination, size, and timestamps. Binlog runs are not catalogued.

### Incremental XtraBackup backups

XtraBackup supports two incremental methods. Select one in **Incremental method**:

- **less_space** — smaller incremental files. Set **Incremental cycle** to control when the full backup runs: `daily`, `weekly`, or a specific weekday (Monday–Sunday).
- **fast_restore** — optimized for faster restores. The cycle is not configurable.

## Restore from a backup

To restore from a backup:
{.power-number}

1. Go to **Apps > MySQL Backups** and select the **Restore** tab.
2. Click **+ New MySQL Restore**.
3. Select the **backup type**.
4. Optionally select a **destination service**. Selecting a known service populates the **Backup source** list with that service's recorded backups. You can also enter a path directly:

    | Format | Example |
    |---|---|
    | Local path | `/backups/mydumper/20240101` |
    | Remote path | `db01:/path/to/backup` |
    | S3 | `s3://bucket/path` |
    | GCS | `gs://bucket/path` |

    Append `/latest` to any path to use the most recent backup automatically.

5. Configure restore options for your backup type (see below).
6. Optionally check **Alert on failure** to receive an alert if the restore task fails.
7. Click **Create MySQL Restore**.

### Restore options by backup type

**Mydumper**

The destination must be a MySQL service in inventory. Optionally scope the restore with **Include databases**, **Skip databases**, or **Restore to Database** to target a single schema.

**XtraBackup**

The destination service is optional — you can restore to any reachable host, including hosts not in inventory. Key options:

| Option | Description |
|---|---|
| Kill MySQL | Kills the MySQL process before restoring. MySQL does **not** restart automatically — start it manually after the restore completes. |
| Skip incrementals | Applies the full backup only, skipping incremental layers. |
| XtraBackup parallel | Number of threads for the restore (default: 4). |
| Data directory | Override the target datadir path. |
| Restore my.cnf | Restores the `my.cnf` configuration file as part of the restore. |

**Binlog — point-in-time recovery**

Set start and stop positions to control how far to replay logs:

| Field | Description |
|---|---|
| Start file / Start position | Where to begin replaying. |
| Stop file / Stop position | Where to stop. Leave empty to replay all available logs. |

### Restoring to a different host

| Backup type | Cross-host restore |
|---|---|
| Mydumper | Destination must be a MySQL service in inventory. |
| XtraBackup | Any reachable host, including hosts not in inventory. Configure access via **SSH user**, **SSH port**, and **SSH key**. |
| Binlog | Any reachable host. Same SSH options as XtraBackup. |

<!-- VERIFY: MySQL version or OS constraints for cross-host XtraBackup restores -->

### Pre and post scripts

All restore types support **Pre-script** and **Post-script** — shell scripts that run on the execution host before and after the restore.

## Scheduling

To manage scheduled backup or restore tasks, click **Schedules** on the **MySQL Backups** or **Restore** page. Scheduled tasks are listed under **Scheduled Tasks** and can be added with **+ Add new**, edited, or deleted without losing their execution history.

<!-- VERIFY: schedule field format (cron, UI picker, presets); minimum interval; overlap behavior when previous run is still in progress -->

## Monitoring

Task status and execution history are visible in the **Apps > MySQL Backups** list. Use the **Status** filter to narrow results.

<!-- VERIFY: exact status values and their meaning; log location on host and retention period; cancel and retry support -->
