# Upgrade PMM Server on AWS

Keep your PMM Server up to date with the latest features, security patches, and performance improvements.

## Prerequisites

Before upgrading your PMM Server, ensure you have:

- a current backup of your PMM data volume
- scheduled maintenance window for potential downtime

## Check current version

Verify your current PMM Server version:

```bash
# Via web interface: PMM Configuration > Updates
# Or via command line:
docker exec pmm-server pmm-admin --version
```

## Upgrade methods

### Method 1: In-place upgrade (Recommended)

For minor version updates and patches:

1. Create a backup snapshot of your PMM data volume:

   ```bash
   aws ec2 create-snapshot --volume-id vol-xxxxxxxxx --description "Pre-upgrade backup $(date)"
   ```

2. Update the PMM Server container**:

   ```bash
   # SSH to your PMM instance
   ssh -i /path/to/your-key.pem admin@<pmm-server-ip>

   # Pull the latest PMM Server image
   sudo docker pull percona/pmm-server:latest

   # Stop the current PMM Server
   sudo docker stop pmm-server

   # Remove the old container (data is preserved in volumes)
   sudo docker rm pmm-server

   # Start the new container with the same configuration
   sudo docker run -d \
     -p 80:80 \
     -p 443:443 \
     --volumes-from pmm-data \
     --name pmm-server \
     --restart always \
     percona/pmm-server:latest
   ```

3. Verify the upgrade:

   ```bash
   # Check container status
   sudo docker ps
   
   # Verify PMM is accessible
   curl -k https://localhost/ping
   
   # Check version via web interface
   ```

### Method 2: Blue-green deployment

For major version upgrades or when minimizing downtime is critical:

1. Launch a new PMM instance with the latest version following the [deployment guide](../aws/deploy_aws.md).

2. Create a volume from your PMM data snapshot and attach it to the new instance to restore data from backup.

3. Update PMM clients to point to the new server:

   ```bash
   # On each monitored host
   pmm-admin config --server-url=https://new-pmm-server-ip:443
   ```

4. Decommission the old instance once verified.

## Upgrade to specific versions

To upgrade to a specific PMM version instead of latest:

```bash
# Example: Upgrade to PMM 3.0.0
sudo docker pull percona/pmm-server:3.0.0

# Follow the same container replacement steps but use the specific tag
sudo docker run -d \
  -p 80:80 \
  -p 443:443 \
  --volumes-from pmm-data \
  --name pmm-server \
  --restart always \
  percona/pmm-server:3.0.0
```

## Post-upgrade tasks

After upgrading PMM Server:

1. Verify all services are running:
   ```bash
   sudo docker exec pmm-server supervisorctl status
   ```

2. Check PMM client connectivity:
   ```bash
   # On monitored hosts
   pmm-admin status
   pmm-admin list
   ```

3. Update PMM clients if required for compatibility:
   ```bash
   # Download and install latest PMM Client
   wget https://www.percona.com/downloads
   ```

4. Review dashboards and alerting rules** for any changes or new features.

5. Test monitoring functionality to ensure data collection continues normally.

## Rollback procedure

If issues occur after upgrade:

1. Stop the new PMM container:
   ```bash
   sudo docker stop pmm-server
   sudo docker rm pmm-server
   ```

2. Restore from backup:

   - Create volume from pre-upgrade snapshot
   - Attach to instance
   - Start previous PMM version

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

1. Set up CloudWatch alarms to monitor PMM health
2. Use AWS Systems Manager for automated patching schedules
3. Implement backup automation before upgrades
4. Create upgrade runbooks for your team

## Next steps

After successful upgrade:

- [Configure new features](../aws/configure_aws.md) introduced in the latest version
- [Update monitoring alerts](../../../../alert/index.md) to use new capabilities
- [Review performance optimizations](../aws/configure_aws.md#optimize-memory-allocation) for the new version