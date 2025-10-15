# Create a MySQL backup

Before creating a backup, make sure to check the [MySQL backup prerequisites](mysql_prerequisites.md).

To create a backup:
{.power-number}

1. Go to  <i class="uil uil-history"></i> **Backup > All Backups**.
2. Click <i class="uil uil-plus-square"></i> **Create Backup**.
3. Specify the type of backup that you want to create: **On Demand** or **Schedule Backup**.
4. Enter a unique name for this backup.
5. Choose the service to back up from the Service name drop-down menu. This automatically populates the **DB Technology** field and selects the **Physical** data model, as this is the only model available for MySQL backups.
6. Choose a storage location for the backup. MySQL currently only supports storing backups to Amazon S3. If no options are available here, see the [Create a storage location topic](../prepare_storage_location.md) section above.
7. If you're creating scheduled backups, also specify the backup type, the schedule, and a retention policy for your backup:
    - **Backup Type**: currently, PMM only supports **Full** backup types for MySQL.
    - **Schedule**: configure the frequency and the start time for this backup.
    !!! caution alert alert-warning "Important"
        Make sure that the schedule you specify here does not create overlapping jobs or overhead on the production environment. Also check that your specified schedule does not overlap with production hours.
    - **Retention**: this option is only available for Snapshot backups stored on S3-compatible object storage. If you want to keep an unlimited number of backup artifacts, type `0`.<a id="folder-field"></a>
8.  Leave the **Folder** field as is. This field is relevant for MongoDB backups to ensure compatibility with PBM wokflows and comes prefilled with the cluster label.
9. Expand **Advanced Settings** to specify the settings for retrying the backup in case of any issues. You can either let PMM retry the backup again (**Auto**), or do it again yourself (**Manual**). Auto retry mode enables you to select up to ten retries and an interval of up to eight hours between retries.
10. To start creating the backup artifact, click **Backup** or **Schedule** at the top of the window, depending on whether you are creating a scheduled or an on-demand backup.
11. Go to the **All Backups** tab, and check the **Status** column. An animated ellipsis icon :material-dots-horizontal: shows that a backup is currently being created.
