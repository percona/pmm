---
title: Overview
slug: database-backups
categorySlug: backup-api
order: 0
---


This section provides a set of API endpoints that allow to backup databases. Currently, PMM Backup Management works with the following database families:

- MongoDB (Generally Available)
- MySQL (in Technical Preview)


To be able to make a backup, you should start by [preparing a backup location](https://docs.percona.com/percona-monitoring-and-management/get-started/backup/prepare_storage_location.html#prepare-a-location-for-local-backups), which is where the backup artifacts will be physically stored. Although the backup location can be re-used to store multiple backups, we generally recommend to create a backup location per database service, which will help organize your storage.

Here a few other references to :

- [Make a backup](ref:startbackup)
- [Restore the database from a backup](ref:restorebackup)
- [List restore history items](ref:listrestores)
- [List available backup locations](ref:listlocations)

Read [more](https://docs.percona.com/percona-monitoring-and-management/get-started/backup/index.html).
