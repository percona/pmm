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

1. Verify all services are running:
   ```bash
   docker exec pmm-server supervisorctl status
   ```

2. Check PMM client connectivity:
   ```bash
   # On monitored hosts
   pmm-admin status
   pmm-admin list
   ```
3. Test monitoring functionality to ensure data collection continues normally.


## Rollback procedure

If issues occur after upgrade:
{.power-number}

1. Stop the new PMM container:
   ```bash
   sudo docker stop pmm-server
   sudo docker rm pmm-server
   ```

2. Restore from backup by creating a volume from your pre-upgrade snapshot, attaching it to the instance, and starting the previous PMM version.

3. Revert PMM Clients if they were updated:

   ```bash
   # Reinstall previous client version
   pmm-admin config --server-url=https://original-pmm-server:443
   ```

## Troubleshooting upgrades

### Container won't start after upgrade

```bash
# Check logs
sudo docker logs pmm-server

# Verify volume mounts
sudo docker inspect pmm-server
```

### Database migration issues

```bash
# Access PMM container
sudo docker exec -it pmm-server bash

# Check database status
pmm-admin status
```
## Automated upgrade scheduling

Consider implementing automated upgrade workflows:

- set up CloudWatch alarms to monitor PMM health
- use AWS Systems Manager for automated patching schedules
- implement backup automation before upgrades
- create upgrade runbooks for your team
