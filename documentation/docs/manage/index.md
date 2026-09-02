# Manage

!!! warning "Tech Preview"
    Management capabilities are not production-ready. Use for testing and feedback only.

PMM is expanding beyond monitoring into database management. Through the Management framework integration, formerly known as SEP (Services Enablement Platform), you can now trigger and track database operations on your hosts directly from PMM, without SSH access or extra software on those hosts.

This is the first step in a broader initiative to give you a single place to monitor, manage, and act on your entire database infrastructure.

## Available capabilities

- [MySQL Backups](mysql-backup.md) — run and schedule MySQL backups using XtraBackup, Mydumper, or Binlog, and restore from them.
- [Support Diagnostics](support-diagnostics.md) — collect diagnostic data from your hosts and send it directly to your Percona support case in ServiceNow.

More operations and database types will be added in future releases.
