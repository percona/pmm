# Back up and restore

Losing your data can destroy your business. This is why backing up data is critical for all database operations.
Even more important than backing up data, is the ability to restore it in the event of data loss.
PMM enables you to do all this with zero downtime and minimal performance impact.

Currently, PMM provides Backup and Restore functionality to work with:

- MongoDB (Generally Available)
- MySQL (in Technical Preview)

Enable the **Backup Management** option in PMM's Advanced Settings to activate the **Backup** page from where you can:

- Create and restore MongoDB and MySQL backups
- Automate backup scheduling
- Set retention policies
- Monitor your backup and restore activity

## Supported setups

For MySQL databases, you can create and restore on-demand and scheduled physical backups. 
For MongoDB, you can create and restore physical, logical and Point-in-Time-Recovery (PITR) backups, both on-demand and scheduled.

### Sharded MongoDB cluster configurations
PMM 2.38 added support for creating backups of sharded MongoDB clusters. However, the restoring process is not handled end-to-end, and requires you to manually restore the artifacts using the CLI in Percona Backup for MongoDB.

For information on restoring sharded backups, check the [PBM documentation](https://docs.percona.com/percona-backup-mongodb/usage/restore.html)
  

## Start here
To learn how to create and restore backups, check out subtopics below:

- [Prepare a storage location](prepare_storage_location.md)
- [MongoDB  backups](../../get-started/backup/backup_mongo.md)
- [MySQL backups](backup_mysql.md) 


## Additional resources
Here are some external resources for learning more about databases backups:

- [Amazon Web Services S3](https://aws.amazon.com/s3/)
- [Percona Backup for MongoDB](https://www.percona.com/doc/percona-backup-mongodb/installation.html)
- [PERCONA_QPRESS](https://docs.percona.com/percona-xtrabackup/2.4/backup_scenarios/compressed_backup.html)
- [PERCONA_XBCLOUD](https://www.percona.com/doc/percona-xtrabackup/2.3/xbcloud/xbcloud.html)
- [PERCONA_XBSTREAM](https://www.percona.com/doc/percona-xtrabackup/2.3/xbstream/xbstream.html)
- [PERCONA_XTRABACKUP](https://www.percona.com/software/mysql-database/percona-xtrabackup)
- [oplog slices](https://www.percona.com/doc/percona-backup-mongodb/glossary.html#term-oplog-slice)
- [Percona Server for MongoDB](https://www.percona.com/software/mongo-database/percona-server-for-mongodb)
- [MongoDB Replication](https://docs.mongodb.com/manual/replication/)
