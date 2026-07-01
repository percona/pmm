# Migrate from Virtual Appliance (OVF)

OVF virtual appliance support was removed in PMM 3.9.0. No new OVA images are published. If you are still running PMM on a virtual appliance, migrate to a supported deployment method.

## Before you migrate

Your PMM Server stores monitoring data, dashboards, alert configurations, and user settings. To preserve this data during migration:
{.power-number}

1. Note your current PMM Server version. Run `pmm-admin status` on any connected client or check **Configuration > Updates** in the PMM UI.
2. Document your connected databases. Go to **Configuration > Inventory** in the PMM UI and record all monitored services, their connection parameters, and any custom labels.
3. Export custom dashboards. If you have created or modified dashboards, export them as JSON from the Grafana UI (**Dashboard > Share > Export**).
4. Back up alert rules and contact points. Note any custom alert templates, notification channels, and silences you have configured.

## Deploy your new PMM Server

Keep your OVA instance running while setting up the new server — you will need it for client reconfiguration and parallel validation.

Deploy on a **different host or IP address** from your current OVA, then follow the guide for your chosen method:

- **[Docker](../docker/index.md) (recommended)**: simplest migration path with minimal operational change
- **[Podman](../podman/index.md)**: rootless containers for security-sensitive environments
- **[Helm](../helm/index.md)**: Kubernetes-native deployment with high availability support

Once your new server is up, continue with the steps below.

## After deploying the new PMM Server
{.power-number}

1. [Configure each PMM Client](../../../install-pmm-client/package_manager.md#step-2-install-pmm-client) to point to the new server using service accounts:
```bash
pmm-admin config --server-insecure-tls \
  --server-url=https://service_token:<YOUR_GLSA_TOKEN>@<NEW_PMM_SERVER_IP>:443
```

2. Verify data is flowing by logging into the new PMM Server UI and confirming that all monitored services appear in **Configuration > Inventory** with current metrics on dashboards.

3. Restore any custom dashboards by importing the exported JSON files via **Dashboards > New > Import** in the Grafana UI.

4. Recreate alert rules and contact points on the new server to match your previous setup.

5. Decommission the OVF instance once everything is confirmed working.

!!! note
    Historical metrics from the OVF deployment are not automatically transferred to the new server. If you need to preserve historical data, consider running both instances in parallel until the old data ages out of your retention window.
