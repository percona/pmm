# MongoDB backup prerequisites

Before creating MongoDB backups, make sure to:

1. Check that **Backup Management** is enabled and the <i class="uil uil-history"></i> Backup option is available on the side menu. If Backup Management has been disabled on your instance, go to <i class="uil uil-cog"></i> **Configuration > PMM Settings > Advanced Settings**, re-enable **Backup Management**  then click **Apply changes**.
2. [Prepare and create a storage location for your backups](../../get-started/backup/prepare_storage_location.md).
3. Check that [PMM Client](../../setting-up/client/index.md) is installed. For creating logical backups, the PMM client should run on at least one node of the replica set. Make sure this is the one that will be used for backup and restore jobs. For physical backups, make sure PMM client runs on all nodes instead.  
4. Check that [Percona Backup for MongoDB](https://docs.percona.com/percona-backup-mongodb/index.html) (PBM) is installed and `pbm-agent` is running on all MongoDB nodes in the replica set. Make sure to [configure the MongoDB connection URI for pbm-agent](https://docs.percona.com/percona-backup-mongodb/install/initial-setup.html#set-the-mongodb-connection-uri-for-pbm-agent) on all nodes. 

!!! caution alert alert-warning "Important"
       PMM 2.32 and later require PBM 2.0.1 or newer

5. Check that your MongoDB Services are managed as clusters in PMM. Go to **PMM Inventory > Services** page, expand the **Details** section <image src="../../_images/arrow-downward.ico" width="15px" aria-label="downward arrow"/> on the **Options** column, and make sure that all the services in the table specify a cluster name.
Services that do not specify a cluster name should be removed and re-added using command like the following:
   <pre><code>pmm-admin add mongodb \
   --username=pmm_mongodb --password=password \
   query-source=profiler <mark>--cluster=mycluster</mark></code></pre>

6. Check that MongoDB nodes are members of replica set.
7. Check that you set the [required permissions for creating and restoring MongoDB backups](../../setting-up/client/mongodb.md#create-pmm-account-and-set-permissions).
8. Verify the [MongoDB supported configurations and limitations](../../get-started/backup/mongodb_limitations.md).
   
!!! caution alert alert-warning "Important"
       Never use `pbm` in manual mode! PMM already takes care of the pbm configuration (with the exception of the connection URI required to start the agent). Any other manual intervention can break the state.
