# MongoDB backup prerequisites

Before creating MongoDB backups, make sure to:
{.power-number}

1. Check that **Backup Management** is enabled and the <i class="uil uil-history"></i> Backup option is available on the side menu. If Backup Management has been disabled on your instance, go to :material-cog: **Configuration > PMM Settings > Advanced Settings**, re-enable **Backup Management**  then click **Apply changes**.
2. [Prepare and create a storage location for your backups](../prepare_storage_location.md).
3. Check that [PMM Client](../../install-pmm/install-pmm-client/index.md) is installed and running on all MongoDB nodes in the cluster.
4. Check that [Percona Backup for MongoDB](https://docs.percona.com/percona-backup-mongodb/index.html) (PBM) is installed and `pbm-agent` is running on all MongoDB nodes in the replica set. Make sure to [configure the MongoDB connection URI for pbm-agent](https://docs.percona.com/percona-backup-mongodb/install/initial-setup.html#set-the-mongodb-connection-uri-for-pbm-agent) on all nodes.
5. Check that installed **mongod** binary is added to **PATH** variable of the user under which PMM client is running, and that **mongod** is controlled as a service by **systemctl**. PMM only works with a single **mongod** installed on a node.
6. Check that your MongoDB Services are managed as clusters in PMM. Go to **PMM Inventory > Services** page, expand the **Details** section <image src="../../images/arrow-downward.ico" width="15px" aria-label="downward arrow"/> on the **Options** column, and make sure that all the services in the table specify a cluster name.
Services that do not specify a cluster name should be removed and re-added using commands like the following:
   <pre><code>pmm-admin add mongodb \
   --username=pmm_mongodb --password=password \
   query-source=profiler <mark>--cluster=mycluster</mark></code></pre>

7. Check that MongoDB nodes are members of replica set.
8. Check that you set the [required permissions for creating and restoring MongoDB backups](../../install-pmm/install-pmm-client/connect-database/mongodb.md#create-user-and-assign-created-role).
9. Verify the [MongoDB supported configurations and limitations](mongodb_limitations.md).

!!! caution alert alert-warning "Important"

      Use `pbm` in manual mode only for restoring sharded cluster backups or other operations that can only be completed via the PBM CLI! Since PMM takes care of the PBM configuration, any unnecessary manual intervention can break the state.

       PMM 3 and later require PBM 2.0.1 or newer.