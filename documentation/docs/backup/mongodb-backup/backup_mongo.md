# Supported setups for MongoDB backups

PMM supports the following actions for MongoDB backups: 

!!! note alert alert-primary "Docker limitations"
    MongoDB instances running in Docker containers are not supported for backup operations.

## Replica set setups
The following backup operations are supported for replica sets:

  - Storing backups on Amazon S3-compatible object storage, and on mounted filesystem
  - Creating and restoring Logical snapshot backups
  - Creating and restoring Physical snapshot backups
  - Creating logical PITR backups both locally and on S3-compatible object storage. Restoring logical PITR backups from S3-compatible object storage.

  
## Sharded clusters

PMM 3 supports backing up sharded clusters. However, restoring for sharded cluster configurations is only supported from the CLI, and is handled via [Percona Backup for MongoDB](https://docs.percona.com/percona-backup-mongodb/usage/restore.html).

  - Storing backups on Amazon S3-compatible object storage, and on mounted filesystem
  - Creating Logical snapshot backups
  - Creating Physical snapshot backups
  - Creating logical PITR backups both locally and on S3-compatible object storage
 
For a detailed overview of the supported setups for MongoDB, check out the [Support matrix](mongodb_limitations.md).
