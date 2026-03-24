---
title: Overview
slug: database-backups
category:
  uri: backup-api
position: 0
---


This section provides a set of API endpoints that allow to backup databases. Currently, PMM Backup Management works with the following database families:

- MongoDB (Generally Available)
- MySQL (in Technical Preview)


To be able to make a backup, you should start by [preparing a backup location](https://docs.percona.com/percona-monitoring-and-management/3/backup/prepare_storage_location.html), which is where the backup artifacts will be physically stored. Although the backup location can be re-used to store multiple backups, we generally recommend to create a backup location per database service, which will help organize your storage.

### Backup Locations

- [List backup locations](ref:listlocations)
- [Add, change, remove, and test backup locations](ref:manage-backup-locations)

### On-Demand and Scheduled Backups

- [Make a backup](ref:startbackup)
- [Schedule, list, change, and remove scheduled backups](ref:scheduled-backups)

### Backup Artifacts

- [List, delete artifacts, check compatible services, and get logs](ref:manage-backup-artifacts)

### Restore

- [Restore the database from a backup](ref:restorebackup)
- [List restore history items and get restore logs](ref:listrestores)

For more information, see [Back up and restore](https://docs.percona.com/percona-monitoring-and-management/3/backup/index.html) in the user documentation.
