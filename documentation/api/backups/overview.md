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

Here a few other references to :

- [Make a backup](ref:startbackup)
- [Restore the database from a backup](ref:restorebackup)
- [List restore history items](ref:listrestores)
- [List available backup locations](ref:listlocations)

For more information, see [Back up and restore](https://docs.percona.com/percona-monitoring-and-management/3/backup/index.html) in the user documentation.
