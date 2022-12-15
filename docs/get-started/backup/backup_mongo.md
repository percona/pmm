# Supported setups for MongoDB backups

PMM supports MongoDB replica set setups for:

  - Storing backups on Amazon S3-compatible object storage, and on mounted filesystem
  - Creating and restoring Logical snapshot backups
  - Creating and restoring Physical snapshot backups
  - Creating logical PITR backups both locally and on S3-compatible object storage. Restoring logical PITR backups from S3-compatible object storage.
  
For a detailed overview of the supported setups for MongoDB, check out the [Support matrix](../backup/mongodb_limitations.md).
