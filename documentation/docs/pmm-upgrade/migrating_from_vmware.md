# Migrate from VMware to alternative platforms

Migrate your PMM Server deployment from VMware to supported platforms before VMware support is removed in PMM 3.6.0 (expected January 2026).

## Prerequisites and planning

Before starting the migration:

- Check if your VMware PMM instance is running PMM 3.x. [Upgrade from PMM2 if needed](../pmm-upgrade/migrating_from_pmm_2.md).
- Check for sufficient storage space for backups (at least 2x your current `/srv` directory size).
- Note your current PMM Server IP address and configuration.
- Plan a maintenance window for the migration (typically 2-4 hours depending on data size).

### Platform-specific prerequisites

=== " For Docker/Podman"
    - Confirm Docker/Podman installation and sufficient disk space
    - Ensure port `443` availability

=== "For Kubernetes"
    - Ensure kubectl access and sufficient cluster resources
    - Verify persistent volume availability

=== "For OpenShift"
    - Verify `cluster-admin`privileges
    - Confirm appropriate security context constraints (SCCs) are configured

## Migration steps 
To migrate away from VMware:
{.power-number}

1.  SSH into your VMware PMM Server:
    ```bash
    ssh admin@<your-pmm-ip>
    ```

2.  Check the size of your data to estimate backup time and storage needs:
    ```bash
    sudo du -sh /srv
    # Also check available space for backup
    df -h /tmp
    ```

3.  Create a full backup of the `/srv` directory so that all metrics, settings, and database files can be restored on the new platform:
    -   Stop PMM services to ensure data consistency:
        ```bash
        sudo supervisorctl stop all
        ```
    -   Create a backup of the PMM data directory:
        ```bash
        sudo tar -czf /tmp/pmm-backup-$(date +%Y%m%d-%H%M%S).tar.gz /srv
        ```
    -   Note the backup filename and verify it was created:
        ```bash
        ls -lh /tmp/pmm-backup*.tar.gz
        ```
    -   Restart services (if continuing to use the VMware instance temporarily):
        ```bash
        sudo supervisorctl start all
        ```

4.  Export custom configurations to save them before migration:

    - **Dashboards**: navigate to each custom dashboards, click the share icon and select **Export > Save to file**.
    - **Alert rules**: go to **Alerting > Alert rules** and copy the configuration details or take screenshots of each rule. Alternatively, export all rules via the PMM API:
        ```bash
        curl -k -u admin: https://<your-pmm-ip>/graph/api/ruler/grafana/api/v1/rules
        ```
    - **Service accounts**: note service names and roles from **Configuration > PMM Settings > Administration > Users and Access > Service Accounts**. 
    - **External database connections**: check **Configuration > PMM Settings** and custom configurations from **Advanced Settings**.

5. Choose your migration target and deploy new PMM:

=== "Migrate to Docker (recommended)"
    **Benefits:** simplified deployment and updates, better resource utilization and easy scaling and backup management. 

    To migrate to Docker:
    {.power-number}

    1. [Install PMM Server with Docker](../install-pmm/install-pmm-server/deployment-options/docker/index.md) to set up your environment.

    2. Create PMM data volume and restore backup:
        ```bash
        docker volume create pmm-data
        
        # Start temporary container
        docker run -d --name pmm-temp -v pmm-data:/srv busybox sleep 3600
        
        # Copy and extract backup
        scp admin@<vmware-pmm-ip>:/tmp/pmm-backup-*.tar.gz .
        docker cp pmm-backup-*.tar.gz pmm-temp:/tmp/
        docker exec pmm-temp sh -c "cd / && tar -xzf /tmp/pmm-backup-*.tar.gz"

        # Clean up temporary container
        docker rm -f pmm-temp
        ```
    
    3. Launch PMM Server with the restored data:
        ```bash
        docker run -d \
            --restart always \
            --publish 443:8443 \
            --volume pmm-data:/srv \
            --name pmm-server \
            percona/pmm-server:3
        ```
=== "Migrate to VirtualBox"

    **Benefits:** familiar VM management interface, easy snapshots and rollback capabilities, minimal learning curve from VMware.

    To migrate to VirtualBox:
    {.power-number}
    
    1. [Deploy PMM Server on VirtualBox](../install-pmm/install-pmm-server/deployment-options/virtual/virtualbox.md) to import and configure your new PMM instance.

    2. Transfer and restore backup:
        ```bash
        # Get the new VM IP
        VBoxManage guestproperty get "PMM Server" "/VirtualBox/GuestInfo/Net/0/V4/IP"

        # Copy backup to new VM
        scp /tmp/pmm-backup-*.tar.gz admin@<new-vm-ip>:/tmp/

        # SSH to new VM and restore
        ssh admin@<new-vm-ip>
        sudo supervisorctl stop all
        sudo mv /srv /srv.original
        cd / && sudo tar -xzf /tmp/pmm-backup-*.tar.gz
        sudo supervisorctl start all
        ```

=== "Migrate to Podman"

    **Benefits**: rootless container execution, no daemon dependency and enhanced security model.

    To migrate to Podman:
    {.power-number}

    1. [Install PMM Server with Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md) to set up your new environment.
    2. Create storage and restore data:
        ```bash
        # Create volume (if not already created during installation)
        podman volume create pmm-data

        # Start temporary container
        podman run -d --name pmm-temp -v pmm-data:/srv busybox sleep 3600

        # Copy and extract backup
        podman cp pmm-backup-*.tar.gz pmm-temp:/tmp/
        podman exec pmm-temp sh -c "cd / && tar -xzf /tmp/pmm-backup-*.tar.gz"
        podman rm -f pmm-temp
        ```
    3. Start PMM Server with your restored data:

        - For systemd integration (with UI updates):
            ```bash
            # Ensure systemd service files exist, then:
            systemctl --user enable --now pmm-server
            ```
        
        - For basic container (no UI updates):
            ```bash
            podman run -d \
                --restart always \
                --publish 443:8443 \
                --volume pmm-data:/srv \
                --name pmm-server \
                percona/pmm-server:3
            ```

=== "Migrate to Kubernetes/OpenShift with Helm"
    **Benefits:** simplified deployment and updates, better resource utilization, easy scaling and backup management.

    To migrate to Kubernetes/Openshift:
    {.power-number}

    1. [Deploy PMM Server on your Kubernetes or Openshift cluster](../install-pmm/install-pmm-server/deployment-options/helm/index.md).
    2. Restore backup to Kubernetes:
        ```bash
        # Get the PMM pod name
        PMM_POD=$(kubectl get pods -n pmm -l app.kubernetes.io/name=pmm -o jsonpath='{.items[0].metadata.name}')

        # Copy backup to pod
        kubectl cp pmm-backup-*.tar.gz pmm/$PMM_POD:/tmp/

        # Stop services, backup existing /srv, restore data, and restart services
        kubectl exec -n pmm $PMM_POD -- supervisorctl stop all
        kubectl exec -n pmm $PMM_POD -- mv /srv /srv.original
        kubectl exec -n pmm $PMM_POD -- sh -c "cd / && tar -xzf /tmp/pmm-backup-*.tar.gz"
        kubectl exec -n pmm $PMM_POD -- supervisorctl start all
        ```

## Post-migration checks
Verify that all components are functioning correctly before decommissioning the old VMware instance. This restores custom configurations and updates all connected systems to use the new PMM server.
{.power-number}

1. Access new PMM Server from `https://<new-pmm-ip>` and check PMM version under **Configuration > Updates**. 
2. Update all PMM Clients to connect the new server.
    - **If clients use IP addresses:** Update each client individually:
      ```bash
      # Edit the PMM Agent configuration file
      sudo nano /usr/local/percona/pmm2/config/pmm-agent.yaml

      # Find the "server_address" line and update it to your new PMM Server IP.
      # Example: server_address: "192.168.1.100:443"

      # Save the file and exit the editor
      
      # Restart the PMM Agent service to apply changes
      sudo systemctl restart pmm-agent

      # Verify the connection status
      pmm-admin status
      ```
    - **If clients use DNS hostnames**: skip updating individual clients and update DNS records instead. See step 4.

3. Verify data integrity:

    - **Historical metrics**: Open any dashboard, set time range to **Last 7 days**, and verify graphs show continuous data without gaps.
    - **Service inventory**: Check **Configuration > PMM Inventory > Services** for all monitored services.
    - **QAN**: Go to **Query Analytics**, set time range to **Last 7 days**, and verify past SQL queries appear with execution times.
    - **Custom dashboards**: Verify imported dashboards display correctly.
    - **Alert functionality**: Test alert rule triggers and notifications.

4. Update DNS/network configuration: 

    - **Update DNS**: point your PMM DNS name to the new server IP.
    - **IP Swap**: assign the old VMware IP to the new server.
    - **Load Balancer**: update backend pool to point to new server.

5. (DNS-based only) If you skipped step 2 because your clients use DNS hostnames, restart pmm-agent after updating DNS in step 4:

    ```bash
    # On each DNS-configured client after DNS update
    sudo systemctl restart pmm-agent
    ```

6. Restore custom configurations: 

    - **Import dashboards**: Go to **Dashboards > Browse > Import** and upload previously exported dashboard JSON files. 
    - **Recreate custom alert rules**: Go to **Alerting > Alert Rules** and recreate custom rules.
    - **Reconfigure services**: Restore backup schedules and custom settings, then update service account configurations. 

7. Decommission VMware instance once you've verified the migration is successful:

    - Keep the VMware instance offline but available for 1-2 weeks as a fallback.
    - Take a final VM snapshot/backup for archiving.

8. Select the VM in VMWare and Power off the instance then click **Remove > Delete** all files.
