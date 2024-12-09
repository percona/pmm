# Restore PMM Server from a backup

To restore your PMM Server from a backup:
{.power-number}

1. Create a new volume by using the latest snapshot of the PMM data volume.

    ![!image](../../../images/aws-marketplace.pmm.ec2.backup2.png)

2. Stop the PMM Server instance.

3. Detach the current PMM data volume.

    ![!image](../../../images/aws-marketplace.pmm.ec2.backup3.png)

4. Attach the new volume.

    ![!image](../../../images/aws-marketplace.pmm.ec2.backup4.png)

5. Start the PMM Server instance.