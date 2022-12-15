# MongoDB backup prerequisites

Before creating MongoDB backups, make sure to:

1. Enable Backup Management from <i class="uil uil-cog"></i> **Configuration > PMM Settings > Advanced Settings** and activate the **Backup Management** option. This adds the <i class="uil uil-history"></i> Backup option on the side menu.
2. [Prepare and create a storage location for your backups](../../get-started/backup/prepare_storage_location.md).
3. For setups where PMM Server runs as a Docker container, enable backup features at container creation time by adding `-e ENABLE_BACKUP_MANAGEMENT=1` to your `docker run` command.
4. Check that [PMM Client](../../setting-up/client/index.md) is installed. For creating logical backups, the PMM client should run on at least one node of the replica set. Make sure this is the one that will be used for backup and restore jobs. For physical backups, make sure PMM client runs on all nodes instead.  
5. Check that [Percona Backup for MongoDB](https://docs.percona.com/percona-backup-mongodb/index.html) (PBM) is installed and `pbm-agent` is running on all MongoDB nodes in the replica set. PMM 2.32 and later require PBM 2.0.1 or newer.
6. Check that MongoDB is a member of a replica set.
7. Check that you set the [required permissions for creating and restoring MongoDB backups](../../setting-up/client/mongodb.md#create-pmm-account-and-set-permissions).
8. Verify the [MongoDB supported configurations and limitations](../../get-started/backup/mongodb_limitations.md).
   

!!! caution alert alert-warning "Important"
       Never use `pbm`  in manual mode! PMM already takes care of the pbm configuration and any manual intervention can break the state.