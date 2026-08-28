# MySQL Backup

!!! warning "Tech Preview"
    This feature is not production-ready. Use for testing and feedback only.

MySQL Backup lets you run and schedule MySQL backups against MySQL services in your PMM inventory, restore from those backups, and track job status, type, and execution history, all from within PMM.

This capability is part of the [Management framework](index.md) integration and is intended to replace the existing PMM MySQL backup feature once it reaches GA.

## Supported backup types

- **XtraBackup** — physical backup using Percona XtraBackup
- **Mydumper** — logical backup using Mydumper
- **Binlog** — binary log backup

## Before you start

- PMM Client 3.10.0 or later must be installed on the monitored host.
- The MySQL service must be registered in your PMM inventory.

## Run a MySQL backup

1. Go to **Management > MySQL Backup** in the left navigation.
2. Click **+ New MySQL Backup**.
3. Select the target MySQL service, backup type, and destination.
4. Click **Run** to start immediately, or configure a schedule.

You can monitor job status, view logs, and review execution history from the same screen.

## Restore from a backup

1. Go to **Management > MySQL Backup** and select the **Restore** tab.
2. Click **+ New MySQL Restore**.
3. Select the backup to restore from and the target host.
4. Click **Restore**.
