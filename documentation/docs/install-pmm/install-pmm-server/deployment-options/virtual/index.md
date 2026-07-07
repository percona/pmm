# Deploy PMM Server as a Virtual Appliance (Deprecated)

## End of support for OVF deployment

OVF virtual appliance deployment is deprecated starting with PMM 3.7.0 and will be removed in PMM 3.9.0 (expected July 2026). If you currently run PMM on a virtual appliance, migrate to a supported deployment method before that date.

### Before you migrate

Your PMM Server stores monitoring data, dashboards, alert configurations, and user settings. To preserve this data during migration:
{.power-number}

1. Note your current PMM Server version. Run `pmm-admin status` on any connected client or check **Configuration > Updates** in the PMM UI.
2. Document your connected databases. Go to **Configuration > Inventory** in the PMM UI and record all monitored services, their connection parameters, and any custom labels.
3. Export custom dashboards. If you have created or modified dashboards, export them as JSON from the Grafana UI (**Dashboard > Share > Export**).
4. Back up alert rules and contact points. Note any custom alert templates, notification channels, and silences you have configured.

### Choose your target deployment

- **[Docker](../docker/index.md) (recommended)**: simplest migration path with minimal operational change
- **[Podman](../podman/index.md)**: rootless containers for security-sensitive environments
- **[Helm](../helm/index.md)**: Kubernetes-native deployment with high availability support

### After deploying the new PMM Server
Once your new PMM Server is running, complete these steps to finish the migration:
{.power-number}

1. [Configure each PMM Client](../../../install-pmm-client/package_manager.md#step-2-install-pmm-client) to point to the new server using service accounts:
```bash
pmm-admin config --server-insecure-tls --server-url=https://service_token:<YOUR_GLSA_TOKEN>@<NEW_PMM_SERVER_IP>:443
```

2. Verify data is flowing by logging into the new PMM Server UI and confirming that all monitored services appear in **Configuration > Inventory** with current metrics on dashboards.

3. Restore any custom dashboards by importing the exported JSON files via **Dashboards > New > Import** in the Grafana UI.

4. Recreate alert rules and contact points on the new server to match your previous setup.

5. Decommission the OVF instance once everything is confirmed working.

!!! note
    Historical metrics from the OVF deployment are not automatically transferred to the new server. If you need to preserve historical data, consider running both instances in parallel until the old data ages out of your retention window.

## Terminology

When working with the PMM Server virtual appliance, it's helpful to understand these terms:

- **Host**: The desktop or server machine running the hypervisor
- **Hypervisor**: Software (e.g., VirtualBox) that runs the guest OS as a virtual machine
- **Guest VM**: Virtual machine running PMM Server (Oracle Linux 9.3)

## OVA file details

| Item | Value |
|------|-------|
| Download page | https://downloads.percona.com/downloads/pmm3/3.7.0/ova/pmm-server-3.7.0.ova |
| File name | `pmm-server-{{release}}.ova` |
| VM name | `pmm-Server-{{release_date}}-N` (`N`=build number) |

## VM specifications

The PMM Server virtual appliance comes pre-configured with the following specifications. You can adjust CPU and memory resources after deployment to match your monitoring needs.

| Component | Value |
|-----------|-------|
| OS | Oracle Linux 9.3 |
| CPU | 1 |
| Base memory | 4096 MB |
| Disks | LVM, 2 physical volumes |
| Disk 1 (`sda`) | VMDK (SCSI, 40 GB) |
| Disk 2 (`sdb`) | VMDK (SCSI, 400 GB) |


## System requirements

For optimal performance, we recommend:

=== "Minimum (1-30 nodes)"
    - **CPU**: 4 cores
    - **Memory**: 8 GB
    - **Disk**: 100 GB

=== "Recommended (31-100 nodes)"
    - **CPU**: 8 cores
    - **Memory**: 16 GB
    - **Disk**: 200 GB

=== "Large (100+ nodes)"
    - **CPU**: 16+ cores
    - **Memory**: 32+ GB
    - **Disk**: 500+ GB

## Hypervisor compatibility

The PMM Server OVA is compatible with VirtualBox 6.0 and later.

## Network requirements

Ensure your network environment allows:

- Outbound internet access for updates (optional)
- Access to monitored database instances
- Access from client browsers to the PMM Server web interface
- Standard ports: 443 (HTTPS), 80 (HTTP, redirects to HTTPS)

See [Network and firewall requirements](../../../plan-pmm-installation/network_and_firewall.md) for full details.

## Default users

PMM Server comes with two pre-configured user accounts that you must secure immediately after installation:

- **admin** (default password: `admin`)
- **root** (default password: `percona`)

Change these default passwords to strong, unique passwords during your first login to prevent unauthorized access.
