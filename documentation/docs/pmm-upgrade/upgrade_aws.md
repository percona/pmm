# Upgrade PMM Server on AWS

Keep your PMM Server up to date with the latest features, security patches, and performance improvements.

## Prerequisites

Before upgrading your PMM Server, ensure you have:

- a current backup of your PMM data volume
- scheduled maintenance window for potential downtime

## Upgrade process

To upgrade PMM Server on AWS: 
{.power-number}

1. Create a backup snapshot of your PMM data volume:

    ```sh
    aws ec2 create-snapshot --volume-id vol-xxxxxxxxx --description "Pre-upgrade backup $(date)"
    ```

2. Go to **PMM Configuration > Updates**  and click **Update now** if a newer version is available.

## Post-upgrade tasks

After upgrading PMM Server:
{.power-number}

1. Go to **Dashboards > Experimental > PMM Health** and check that all services are running. 

2. Go to **PMM Configuration > Inventory > Services** and verify that all monitored nodes and services are listed, their status is **Up**.

3. Test monitoring functionality to ensure data collection continues normally.

## Rollback procedure

If issues occur after upgrade:
{.power-number}

1. Stop the new PMM container:
   ```bash
   systemctl --user stop pmm-server
   ```

2. Restore using your pre-upgrade snapshot. Create a volume from the snapshot, attach it to the instance, and start the previous PMM version.

## Troubleshooting upgrades

If the container won't start after the upgrade:

```bash
# Check container logs
podman logs pmm-server

# Check systemd service logs
journalctl -u pmm-server.service

# Verify volume mounts
podman inspect pmm-server
```