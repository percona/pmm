
# Remove PMM Server Virtual Appliance

When you no longer need your PMM Server virtual appliance or want to perform a clean reinstallation, follow these steps to completely remove the virtual machine.

## Remove virtual machine from VMware
To remove a PMM Server virtual machine from VMware:
{.power-number}

1. Select the PMM Server VM in your inventory and select **Close > Power Off**.
2. With the VM selected, choose **Remove > Delete all files** and confirm the deletion when prompted.

!!! caution "Data loss warning"

This action permanently deletes all monitoring data, dashboards, and configurations. If you need to preserve your PMM data, create a backup before removing the virtual machine.


## Remove virtual machine from VirtualBox
To remove a PMM Server virtual machine from VirtualBox:
{.power-number}

1. Select the PMM Server VM in the VirtualBox Manager and right-click and select **Close > Power Off**. 
2. Right-click on the powered-off VM and select **Remove** .
3. Choose **Delete all files** to remove the VM and its disk images, then click **Remove** to confirm.

## Verify removal
After removing the virtual machine, verify that all associated files have been deleted:

1. Check that the VM no longer appears in your virtualization software's inventory. 
2. Verify that disk space has been reclaimed on your host system.
3. If you used custom storage locations, check those locations for any remaining files.

## Next steps

After removing the virtual machine, you can:
- [Download the latest PMM Server OVA](download_ova.md) to install a newer version
- [Deploy PMM Server using an alternative method](../../index.md), such as Docker or Kubernetes
- [Set up a new PMM Server virtual appliance](vmware.md) with a fresh configuration