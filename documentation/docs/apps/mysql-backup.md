# Back up and restore MySQL databases

!!! warning "Tech Preview"
    This feature is not production-ready. Use for testing and feedback only.

Use MySQL Backup to protect your MySQL databases directly from PMM. Schedule recurring backups, restore from any saved backup, and monitor job progress. Find it under **Apps > MySQL Backups** in the sidebar, part of PMM’s growing set of [database management apps](index.md).

This app runs alongside the MySQL backup feature in **Backups > All backups**. Backups made in one won’t appear in the other. Once this app reaches GA, it will replace the MySQL backup part of that feature.

## Supported databases

You can back up and restore MySQL Community Edition, Percona Server, Percona XtraDB Cluster, and MariaDB. Your databases appear as MySQL-type services in PMM inventory, and backup and restore treat them identically.

Match your backup binary to your server version before you start. See [Backup tools](#backup-tools).

## Supported backup types

| Type | Tool | Best for |
|---|---|---|
| **XtraBackup** | `xtrabackup` (PXB 8.x); `innobackupex` (PXB 2.4 only); `mariadb-backup` (MariaDB) | Physical hot backups of large datasets with minimal locking. Supports full and incremental backups. Must run directly on the database host. Supports MariaDB via `mariadb-backup`. |
| **Mydumper** | `mydumper` | Logical SQL dumps with selective database and table restores. Can run from a remote executor host. |
| **Binlog** | `mysqlbinlog` | Continuous binary log capture. Combine with a base XtraBackup or Mydumper backup for point-in-time recovery. Not a standalone backup. |

## Backup tools

Not every compression algorithm works with every backup binary. If you manage multiple MySQL versions, your compression options will differ across hosts, and a backup compressed on one host may not be readable on another.

Find your binary below to see which compression algorithms you can select when configuring a backup:

| Binary | Version | Compression | Default | Notes |
|---|---|---|---|---|
| xtrabackup (PXB 8.4) | 8.4.0-6 | lz4, zstd | zstd | |
| xtrabackup (PXB 8.0) | 8.0.35-36 | lz4, zstd | zstd | |
| xtrabackup / innobackupex (PXB 2.4) | 2.4.29 | quicklz only | quicklz | `innobackupex` absent from PXB 8.x |
| mariadb-backup | MariaDB 12.3.3 | quicklz only | — | quicklz deprecated since MariaDB 10.1.31 / 10.2.13 |
| mydumper | — | gzip, zstd | — | Logical backup; connects over the network, no datadir access or binary version constraints |

## Before you start

Complete the following steps before creating your first backup.
{.power-number}

1. Enable Nomad on PMM Server by starting it with both `PMM_ENABLE_NOMAD=1` and `PMM_PUBLIC_ADDRESS` set. See [Configure Nomad](../reference/nomad.md).

2. Install the Nomad client on the execution host. The host needs PMM Client 3.2 or later, registered to this PMM Server. PMM Client ships with the Nomad client, so if you deploy PMM Client with Nomad enabled, it's already installed. PMM Client itself doesn't need to run on the execution host, only the Nomad client does.

3. Install the tool for your backup type and make sure it is available on `$PATH`:

    - XtraBackup: `xtrabackup`, `mariadb-backup`, or `innobackupex`
    - Mydumper: `mydumper`
    - Binlog: `mysqlbinlog`

    XtraBackup requires **root** on the execution host. Mydumper and Binlog do not.

    For XtraBackup, use a version that matches your MySQL version:

    - MySQL 5.5, 5.6, 5.7: PXB 2.4.x
    - MySQL 8.0.0–8.0.33: PXB 8.0.x (same version or newer)
    - MySQL 8.0.34+: PXB 8.0.34+
    - MySQL 8.1.x, 8.2.x, 8.3.x: matching PXB version
    - MySQL 8.4.x: any PXB 8.4.x

    The backup type also determines which host can run the task:

    - **XtraBackup**: the executor must be the database host itself. The task always connects to `localhost`.
    - **Mydumper**: the executor can be any host with network access to the database.
    - **Binlog**: the executor can be any host with network access to the database. Use **Alternative binlog host** to stream logs from a specific source host.

    For remote or cloud-hosted databases, select an executor host that has network access to the target.

4. Create two MySQL credential files on the executor host so that PMM and the backup binary can each authenticate to MySQL. PMM reads from these files and will not prompt for a password:

    - **PMM's own connection**: `~/.my.cnf` or `~/.mylogin.cnf`. This file must exist and be readable before you run a backup.
    - **The backup binary's connection**: a separate file passed as `--defaults-file`. This file is not merged with the system configuration, so it must be self-contained and include everything the binary needs to connect.

    For XtraBackup on MySQL 8.0+, the MySQL user must have the `BACKUP_ADMIN` privilege.

    <!-- VERIFY: privilege requirements for Mydumper and Binlog -->

5. Create a backup directory on the execution host and confirm it is writable. Set the path per task in the **Backup directory** field.

6. Trigger a sync from **Inventory** if your MySQL service does not appear in the backup form. Apps syncs from PMM, so services registered in PMM may not appear until a sync has run.

## Run a backup

To run a MySQL backup:
{.power-number}

1. Go to **Apps > MySQL Backups** in the sidebar.
2. Click **+ New MySQL Backup**.
3. Enter a task name, select the **backup type**, and select the **execution host**. For XtraBackup, the execution host must be the host running the MySQL service.
4. Under **Upload**, select one or more **Upload providers** and fill in the destination fields if uploading off-host.
5. Optionally configure compression, encryption, or retention in the relevant sections of the form.
6. Optionally check **Alert on failure** to receive an alert if the backup task fails.
7. Click **Run** to start the backup immediately. To run on a schedule instead, see [Schedule a backup](#schedule-a-backup).

Completed XtraBackup and Mydumper runs are recorded in the backup catalog with their location, upload destination, size, and timestamps. Binlog runs are not catalogued.

### Schedule a backup

Scheduling is a two-step process: first define the backup task, then attach a recurrence to it.
{.power-number}

1. [Run a backup](#run-a-backup) to create the backup task, then click **Run**. This saves the task definition.

2. On the **MySQL Backups** page, click **Schedules**, then click **Add new**.

3. Select the task from the **Task** drop-down menu.

4. Choose a recurrence type:
    - **Interval**: repeat every N minutes, hours, or days.
    - **Cron**: enter a cron expression and select a timezone.

5. Click **Save**.

Scheduled tasks appear in the **Schedules** list.

### Incremental XtraBackup backups

XtraBackup supports two incremental methods. Select one in **Incremental method**:

- **less_space**: smaller incremental files. Set **Incremental cycle** to control when the full backup runs: `daily`, `weekly`, or a specific weekday (Monday–Sunday).
- **fast_restore**: optimized for faster restores. The cycle is not configurable.

## Manage scheduled backups

To manage your scheduled backup tasks, click **Schedules** on the **MySQL Backups** page. From there you can:

- Enable or disable a schedule using the toggle.
- Edit, delete, or copy a schedule using the actions menu.


## Restore from a backup

To restore from a backup:
{.power-number}

1. Go to **Apps > MySQL Backups** and select the **Restore** tab.
2. Click **+ New MySQL Restore**.
3. Select the **backup type**.
4. Select the **destination service** to restore into. The **Backup source** list then shows that service's recorded backups. To use a different source, enter a path directly:

    - Local: `/backups/mydumper/20240101`
    - Remote: `db01:/path/to/backup`
    - S3: `s3://bucket/path`
    - GCS: `gs://bucket/path`

    Add `/latest` to any path to use the most recent backup.

5. Configure [Restore options by backup type](#restore-options-by-backup-type). PMM checks MySQL version compatibility before restoring, but make sure your backup binary version matches the target server. A mismatch fails in the backup tool, not in PMM.
6. Optionally check **Alert on failure** to receive an alert if the restore task fails.
7. Click **Create MySQL Restore**.

### Restore options by backup type

=== "Mydumper"

    The destination must be a MySQL service in inventory. Optionally scope the restore with **Include databases**, **Skip databases**, or **Restore to Database** to target a single schema.

    To restore to a different host, the destination must be a MySQL service in inventory — you cannot restore Mydumper backups to a host outside of PMM inventory.

=== "XtraBackup"

    The destination service is optional. You can restore to any reachable host, including hosts not in inventory. Key options:

    - **Kill MySQL**: kills the MySQL process before restoring. MySQL does **not** restart automatically. Start it manually after the restore completes.
    - **Skip incrementals**: applies the full backup only, skipping incremental layers.
    - **XtraBackup parallel**: number of threads for the restore (default: 4).
    - **Data directory**: override the target datadir path.
    - **Restore my.cnf**: restores the `my.cnf` configuration file as part of the restore.

    To restore to a different host, configure access via **SSH user**, **SSH port**, and **SSH key**.


=== "Binlog"

    Set start and stop positions to control how far to replay logs:

    - **Start file / Start position**: where to begin replaying.
    - **Stop file / Stop position**: where to stop. Leave empty to replay all available logs.

    To restore to a different host, configure access via **SSH user**, **SSH port**, and **SSH key**.

### Pre and post scripts

All restore types support **Pre-script** and **Post-script**: shell scripts that run on the execution host before and after the restore.

## Configure backup options

The following options are configured per task when creating a backup.

### Store backups off-host

By default, backups stay on the execution host. To store them off-host, go to **Apps > MySQL Backups**, click **+ New MySQL Backup**, and select one or more providers under **Upload**:

- S3-compatible storage: S3 bucket
- Google Cloud Storage: GCS bucket
- Rsync: Rsync destination path

You can select multiple providers at the same time. To upload to S3 or Google Cloud Storage, install the corresponding client on the execution host first.

### Limit how many backups to keep

Without retention configured, backups accumulate until you delete them manually. When creating a backup in **Apps > MySQL Backups**, set how long to keep backups in the **Retention** section:

- Mydumper: daily purge (days), weekly purge (weeks)
- XtraBackup: number of copies to keep
- Binlog: purge after (days)

### Compress backups

When creating a backup in **Apps > MySQL Backups**, enable **Compress backup data** and pick an algorithm. 

The algorithms shown after enabling **Compress backup data** depend on your backup tool and, for XtraBackup, which PXB version is installed on the host. See [Backup tools](#backup-tools) for a full breakdown.

- XtraBackup on PXB 8.x: `zstd`, `lz4`
- XtraBackup on PXB 2.4: `quicklz`
- Mydumper: `gzip`, `zstd`
- `mariadb-backup`: `quicklz`

If you manage databases across multiple MySQL versions, make sure the algorithm you pick is supported on the host you plan to restore on. A backup compressed with `quicklz` on a MySQL 5.7 host can't be read by a PXB 8.x binary.

### Encrypt backups

When creating a backup in **Apps > MySQL Backups**, configure encryption in the **Encryption** section. Two GPG encryption modes are available:

- **Encrypt backup**: encrypts the backup in place during the run. Combine with **Encrypt using tmpdir** to write to a temporary directory during encryption.
- **Encrypt after backup completes**: GPG-encrypts the finished backup as a post-run step. Mutually exclusive with **Encrypt using tmpdir**.

Both modes require an **Encryption recipient** (GPG key or recipient ID).

XtraBackup also supports **AES-256 encryption** via a keyfile, configured in the **AES-256 key file path** field.

`mariadb-backup` has no inline encryption option. For MariaDB backups, only **Encrypt after backup completes** is available. **Encrypt backup** (in-place) and the AES-256 keyfile both require an XtraBackup (PXB) binary.

## Monitor backups

Task status and execution history are visible in the **Apps > MySQL Backups** list. Use the **Status** filter to narrow results.

### Find and manage task logs                                                             
                             
Each backup and restore run captures `stdout` and `stderr` as task logs.                
                                                                                          
#### Find logs
Check the `taskhistory_log` table in PMM's embedded PostgreSQL database (`sep`). Logs are not written to disk or Docker container logs.                         
                                                                                          
#### Configure retention
Use the SEP API to set TASKS__LOG_RETENTION_DAYS and override the default 90-day retention period. The maximum is 365 days.

A daily purge job removes log bodies for finished tasks older than the configured threshold. Task history records (who ran what, when, and final status) are never purged.      
                                                                                        
#### Handle large runs
PMM caps each log stream at 100 MiB. If a run exceeds this limit, PMM drops the oldest log chunks. To retain all logs, keep runs under 100 MiB.

#### Executor host logs
Nomad allocation files on the execution host follow Nomad's own garbage collection, independent of `TASKS__LOG_RETENTION_DAYS`.
                                                                      