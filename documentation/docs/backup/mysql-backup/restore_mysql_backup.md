# Restore a MySQL backup

## Restore compatibility

MySQL backups can be restored to the same service it was created from, or to a compatible one. 

To restore a backup:
{.power-number}

1. Go to <i class="uil uil-history"></i> **Backup > All backups** and find the backup that you want to restore.
2. Click the three dots ![](../images/dots-three-vertical.png) in the **Actions** column to check all the information for the backup, then click ![](../images/dots-three-vertical.png) **Restore from backup**.
3. In the **Restore from backup** dialog, select **Same service** to restore to a service with identical properties or **Compatible services** to restore to a compatible service.
4. Select one of the available service names from the drop-down menu.
5. Check the values, then click **Restore**.
6. Go to the **Restores** tab to check the status of the restored backup.

During restoring, PMM disables all the scheduled backup tasks for the current service. Remember to re-enable them manually after the restore.
