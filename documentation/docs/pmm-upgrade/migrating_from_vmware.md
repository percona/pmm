# Migrate from VMware to alternative platforms

Migrate your PMM Server deployment from VMware to supported platforms before VMware support is removed in PMM 3.6.0 (expected January 2026).

## Prerequisites and planning

Before starting the migration:

- check if your VMware PMM instance is running PMM 3.x. [Upgrade from PMM2 if needed](../pmm-upgrade/migrating_from_pmm_2.md).
- check for sufficient storage space for backups (at least 2x your current `/srv` directory size).
- note your current PMM Server IP address and configuration.
- for OpenShift migrations, verify you have cluster-admin privileges and appropriate security context constraints (SCCs).
- plan a maintenance window for the migration (typically 2-4 hours depending on data size).

To migrate from VMware:
{.power-number}

1. SSH into your VMware PMM Server:
   ```bash
   ssh admin@<your-pmm-ip>
   ```

2. Check the size of your data to estimate backup time and storage needs:
   ```bash
    sudo du -sh /srv
    # Also check available space for backup
    df -h /tmp
    ```

3. Create a full of the `/srv` directory so that all metrics, settings, and database files can be restored on the new platform:

    -  Stop PMM services to ensure data consistency:
        ```bash
        sudo supervisorctl stop all
        ```
    - Create a backup of the PMM data directory:
        ```bash
        sudo tar -czf /tmp/pmm-backup-$(date +%Y%m%d-%H%M%S).tar.gz /srv
        ```
    - Note the backup filename and verify it was created:
        ```bash
        ls -lh /tmp/pmm-backup*.tar.gz
        ```
    - Restart services (if continuing to use the VMware instance temporarily):
        ```bash
        sudo supervisorctl start all
        ```

4.  Export custom configurations: 
    - Dashboards: go to custom dashboard, click the share icon and select **Export > Save to file**.
    - Alert rules: go to **Alerting > Alert rules** and copy the configuration or take screenshots of each rule. Alternatively, export all rules via the PMM API:
        ```bash
        curl -k -u admin: https:///graph/api/ruler/grafana/api/v1/rules > alert-rules-backup.json
        ```
    - Service accounts: record service names and roles under **Configuration > PMM Settings > Administration > Users and Access > Service Accounts**. 
    - External database connections: check **Configuration > PMM Settings** and custom configurations under **Advanced Settings**.
    - Other settings: Go to PMM Configuration and record **SSH Key**, data retention **Advanced Settings > Data retention**, **Advanced Settings > Telemetry**, and backup locations (**Backup > Storage Locations**).

5. Choose your migration target:

=== "Migrate to VirtualBox"

    Best if you prefer virtual machine deployments with minimal infrastructure changes.
    {.power-number}

    1. Export VM configuration from VMware via **File > Export to OVF**.
    2. Download PMM OVA: `wget https://downloads.percona.com/downloads/pmm/3.3.1/ova/pmm-server-3.3.1.ova`
    3. Import PMM OVA to VirtualBox: 
        ```bash
        VBoxManage import pmm-server-3.3.1.ova \
                --vsys 0 \
                --vmname "PMM Server" \
                --cpus 4 \
                --memory 8192
        ```
    4. Configure network:
        ```bash
        # Set bridged networking for direct access
        VBoxManage modifyvm "PMM Server" --nic1 bridged --bridgeadapter1 eth0
        ```
    5. Start the VM:
        ```bash
        VBoxManage startvm "PMM Server" --type headless
        ```
    6. Transfer and restore backup:
        ```bash
        # Get the new VM's IP
        VBoxManage guestproperty get "PMM Server" "/VirtualBox/GuestInfo/Net/0/V4/IP"

        # Copy backup to new VM
        scp /tmp/pmm-backup-*.tar.gz admin@<new-vm-ip>:/tmp/

        # SSH to new VM
        ssh admin@<new-vm-ip>

        # Stop services
        sudo supervisorctl stop all

        # Backup existing data
        sudo mv /srv /srv.original

        # Extract backup
        cd /
        sudo tar -xzf /tmp/pmm-backup-*.tar.gz

        # Start services
        sudo supervisorctl start all
        ```

=== "To Docker"
    **Recommended for most users:** Replace VM complexity with containerized simplicity for automatic updates, better performance and easier scaling.    
    {.power-number}
    
    1. Install Docker on target host:
        ```bash
        # Install Docker if not already installed
        curl -fsSL [https://get.docker.com](https://get.docker.com) | bash
        ```
    2. Create PMM data volume: 
        ```bash
        docker volume create pmm-data
        ```
    3. Start temporary container to restore data: 
        ```bash
        # Start a temporary container with the volume mounted
        docker run -d --name pmm-temp \
        -v pmm-data:/srv \
        busybox sleep 3600
        ```
    4. Copy and extract backup: 
        ```bash
        # Copy backup from VMware instance to Docker host
        scp admin@<vmware-pmm-ip>:/tmp/pmm-backup-*.tar.gz .

        # Copy backup into container volume
        docker cp pmm-backup-*.tar.gz pmm-temp:/tmp/

        # Extract backup in the volume
        docker exec pmm-temp sh -c "cd / && tar -xzf /tmp/pmm-backup-*.tar.gz"

        # Remove temporary container
        docker rm -f pmm-temp
        ```
    5. Start PMM Server container: 
        ```bash
        docker run -d \
        --restart always \
        --publish 443:8443 \
        --volume pmm-data:/srv \
        --name pmm-server \
        percona/pmm-server:3
        ```

=== "To Podman"

    Best for environments that need rootless containers and enhanced security without Docker dependencies.
    {.power-number}

    1. Install Podman:
        ```bash
        # On RHEL/CentOS/Fedora
        sudo dnf install podman

        # On Ubuntu/Debian
        sudo apt-get install podman
        ```
    2. Create storage and restore data:
        ```bash
        # Create volume
        podman volume create pmm-data

        # Start temporary container
        podman run -d --name pmm-temp \
        -v pmm-data:/srv \
        busybox sleep 3600

        # Copy and extract backup (similar to Docker steps)
        podman cp pmm-backup-*.tar.gz pmm-temp:/tmp/
        podman exec pmm-temp sh -c "cd / && tar -xzf /tmp/pmm-backup-*.tar.gz"
        podman rm -f pmm-temp
        ```
    3. Run PMM with Podman (rootless): 
        ```bash
        podman run -d \
        --restart always \
        --publish 443:8443 \
        --volume pmm-data:/srv \
        --name pmm-server \
        percona/pmm-server:3
        ```

=== "To Kubernetes with Helm"

    Best for cloud-native environments with orchestration requirements and standard Kubernetes clusters.
    {.power-number}

    1. Create Kubernetes namespace and secret
        ```bash
        kubectl create namespace pmm

        # Create secret with admin password
        kubectl create secret generic pmm-secret \
        --from-literal=PMM_ADMIN_PASSWORD=<your-password> \
        -n pmm
        ```
    2. Add Percona Helm Repository: 
        ```bash
        helm repo add percona [https://percona.github.io/percona-helm-charts/](https://percona.github.io/percona-helm-charts/)
        helm repo update
        ```
    3. Deploy PMM with persistent storage: 
        ```yaml
        # values.yaml
        storage:
          size: 100Gi
          storageClass: "standard"  # Adjust to your storage class

        service:
          type: LoadBalancer  # or NodePort/ClusterIP based on your needs

        secret:
          create: false
          name: pmm-secret
        ```
        ```bash
        helm install pmm percona/pmm \
        -f values.yaml \
        -n pmm
        ```
    4. Restore backup to Kubernetes: 
        ```bash
        # Get the PMM pod name
        PMM_POD=$(kubectl get pods -n pmm -l app.kubernetes.io/name=pmm -o jsonpath='{.items[0].metadata.name}')

        # Copy backup to pod
        kubectl cp pmm-backup-*.tar.gz pmm/$PMM_POD:/tmp/

        # Stop services and restore
        kubectl exec -n pmm $PMM_POD -- supervisorctl stop all
        kubectl exec -n pmm $PMM_POD -- sh -c "cd / && tar -xzf /tmp/pmm-backup-*.tar.gz"
        kubectl exec -n pmm $PMM_POD -- supervisorctl start all
        ```
        
=== "To OpenShift with Helm"

    Best for enterprise OpenShift environments with enhanced security policies and integrated developer tools.
    {.power-number}

    1. Create OpenShift project and secret: 
        ```bash
        oc new-project pmm

        # Create secret with admin password
        oc create secret generic pmm-secret \
        --from-literal=PMM_ADMIN_PASSWORD=<your-password> \
        -n pmm
        ```
    2. Add Percona Helm repository:
        ```bash
        helm repo add percona [https://percona.github.io/percona-helm-charts/](https://percona.github.io/percona-helm-charts/)
        helm repo update
        ```
    3. Create OpenShift-specific values file:
        ```yaml    
        # openshift-values.yaml
        storage:
          size: 100Gi
          storageClass: "gp2"  # Adjust to your OpenShift storage class

        service:
          type: ClusterIP  # Use Routes instead of LoadBalancer

        secret:
          create: false
          name: pmm-secret
        # OpenShift-specific pod security settings
        podSecurityContext:
          runAsNonRoot: true
          seccompProfile:
            type: RuntimeDefault
        ```
    4. Deploy PMM with OpenShift configuration:
        ```bash
        helm install pmm percona/pmm \
        -f openshift-values.yaml \
        -n pmm
        ```
    5. Create Route to expose PMM:
        ```bash
        oc expose svc/pmm-service --port=443
        ```
    6. Restore backup to OpenShift:
        ```bash
        # Get the PMM pod name
        PMM_POD=$(oc get pods -n pmm -l app.kubernetes.io/name=pmm -o jsonpath='{.items[0].metadata.name}')

        # Copy backup to pod
        oc cp pmm-backup-*.tar.gz pmm/$PMM_POD:/tmp/

        # Stop services and restore
        oc exec -n pmm $PMM_POD -- supervisorctl stop all
        oc exec -n pmm $PMM_POD -- sh -c "cd / && tar -xzf /tmp/pmm-backup-*.tar.gz"
        oc exec -n pmm $PMM_POD -- supervisorctl start all
        ```

## Post-migration checks
Verify that all components are functioning correctly before decommissioning the old VMware instance. This ensures data integrity, restores custom configurations, and updates all connected systems to use the new PMM server.
{.power-number}

1. Verify PMM Server access from `https://<new-pmm-ip>` and check PMM version under **Configuration > Updates**. 
2. Update all PMM Clients to point to the new server:

    ```bash
    # On each client node
    pmm-admin config --server-url=https://admin:<password>@<new-pmm-ip>:443 --force
    ```

3. Verify data integrity:

    - Check that historical metrics are present.
    - Verify all services appear in **Configuration > PMM Inventory > Services**.
    - Confirm Query Analytics (QAN) data is available.
    - Test alerting functionality.

4. Update DNS/network configuration: 

    - **Update DNS**: point your PMM DNS name to the new server IP.
    - **IP Swap**: assign the old VMware IP to the new server.
    - **Load Balancer**: update backend pool to point to new server.

5. Go to **Dashboards > Browse > Import** and upload previously exported dashboard JSON files. 

6. Recreate any custom alert rules in **Alerting > Alert Rules** and reconfigure backup schedules.

7. Decommission VMware instance once you've verified the migration is successful:

 - keep the VMware instance offline but available for 1-2 weeks as a fallback.
 - take a final VM snapshot/backup for archival purposes.

8. Select the VM in VMWare and Power Off the instance then click **Remove > Delete** all files.
