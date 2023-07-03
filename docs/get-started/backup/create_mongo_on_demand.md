# Create MongoDB on-demand and scheduled backups

Before creating a backup, make sure to check the [MongoDB backup prerequisites](../backup/mongo-prerequisites.md).

To schedule or create an on-demand backup, check the instructions below. If you want to create a Point-in-time-recovery (PITR) backup instead, see [Create MongoDB PITR backups](create_PITR_mongo.md).

1. Go to <i class="uil uil-history"></i> **Backup > All Backups**.
2. Click <i class="uil uil-plus-square"></i> **Create Backup**.
3. In the **Create Scheduled backup** window, select whether you want to create an **On Demand** or a **Schedule Backup**.
4. Enter a unique name for the backup.
5. Choose the service to back up from the **Service name** drop-down menu. This automatically populates the **DB Technology** field.
6. Select whether you want to create a **Physical** or **Logical** backup of your data, depending on your use case and requirements.
7. Choose a storage location for the backup. MongoDB supports both Amazon S3-compatible and local storage. If no options are available here, see the [Create a storage location](prepare_storage_location.md) topic.
8. Specify the backup type, the schedule, and a retention policy for your backup:
    - **Backup Type**: select **Full**. If you want to create a PITR backup instead, see the [Create MongoDB PITR backups topic](../backup/create_PITR_mongo.md)
    - **Schedule**: if you're creating a scheduled backup, configure its frequency and start time.
    !!! caution alert alert-warning "Important"
    Make sure that the schedule you specify here does not create overlapping jobs or overhead on the production environment. Also, check that your specified schedule does not overlap with production hours.
    
    - **Retention**: this option is only available for snapshot backups stored on S3-compatible storage. If you want to keep an unlimited number of backup artifacts, type `0`.
9. Expand **Advanced Settings** to specify the settings for retrying the backup in case of any issues. You can either let PMM retry the backup again (**Auto**), or do it again yourself (**Manual**). Auto-retry mode enables you to select up to ten retries and an interval of up to eight hours between retries. <a id="folder-field"></a>
10. In the **Folder** field, check the target directory available for the specified service and location. By default, this field is prefilled with the cluster label to ensure that all the backups for a cluster are stored in the same directory. If the field is not automatically populated, the service you have specified is not member of a cluster and should be re-added using the following set of commands:
   <pre><code>pmm-admin add mongodb \
   --username=pmm_mongodb --password=password \
   query-source=profiler <mark>--cluster=mycluster</mark></code></pre>
    !!! caution alert alert-warning "Important"
        Unless you are using verified custom workflows, make sure to keep the default **Folder** value coming from the cluster name. Editing this field will impact PMM-PBM integration workflows.
11. To start creating the backup artifact, click **Backup** or **Schedule** at the top of the window, depending on whether you are creating a scheduled or an on-demand backup.
12. Go to the **All Backups** tab, and check the **Status** column. An animated ellipsis indicator {{icon.bouncingellipsis}} shows that a backup is currently being created.
