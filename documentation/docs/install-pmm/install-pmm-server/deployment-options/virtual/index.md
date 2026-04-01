# Deploy PMM Server as a Virtual Appliance (deprecated)

OVF virtual appliance deployment is deprecated starting with PMM 3.7.0 and will be removed in PMM 3.9.0 (expected July 2026). If you currently run PMM on a virtual appliance, migrate to a supported deployment method before that date.

## Migrate from OVF

=== "Docker (recommended)"
    The simplest way to migrate is to export your existing PMM Server container and reimport it in your new Docker environment. This preserves all your monitoring data, dashboards, alert configurations, and user settings.
    {.power-number}

    1. SSH into your OVF instance.

    2. Stop the PMM Server container:
    ```sh
    docker stop pmm-server
    ```

    3. Export the container to a file. Save it to a directory that is accessible from your host machine:
    ```sh
    docker export -o pmm-server.docker pmm-server
    ```

    4. Copy the exported file to the host where you want to run PMM Server going forward.

    5. On the new host, import the image from the file:
    ```sh
    docker import pmm-server.docker --platform=linux/amd64 percona/pmm-server:backup
    ```

    6. Launch the container using the imported image:
    ```sh
    docker run -d --name pmm-server -p 443:8443 -v pmm-data:/srv percona/pmm-server:backup
    ```

    7. Verify that PMM Server is running and all your data is intact by logging into the PMM UI.

    8. [Configure each PMM Client](../../../install-pmm-client/package_manager.md#step-4-register-the-node) to point to the new server using your service account token and new server address:
    ```bash
    pmm-admin config --server-insecure-tls --server-url=https://service_token:<YOUR_GLSA_TOKEN>@<NEW_PMM_SERVER_IP>:443
    ```

    9. Decommission the OVF instance once everything is confirmed working.

=== "Podman or Kubernetes"
    If Docker is not an option, you can migrate to [Podman](../podman/index.md) or [Kubernetes/Helm](../helm/index.md). Since this method does not carry over your data automatically, document your current setup before starting:
    {.power-number}

    1. Note your current PMM Server version by running `pmm-admin status` on any connected client or checking **Configuration > Updates** in the PMM UI.
    2. Document your connected databases by going to **Configuration > Inventory** in the PMM UI and recording all monitored services, their connection parameters, and any custom labels.
    3. Export custom dashboards as JSON from the Grafana UI (**Dashboard > Share > Export**).
    4. Back up alert rules and contact points, including any custom alert templates, notification channels, and silences.
    5. Deploy a new PMM Server using [Podman](../podman/index.md) or [Helm](../helm/index.md).
    6. [Configure each PMM Client](../../../install-pmm-client/package_manager.md#configure-pmm-client) to point to the new server using your service account token and new server address:
    ```bash
    pmm-admin config --server-insecure-tls --server-url=https://service_token:<YOUR_GLSA_TOKEN>@<NEW_PMM_SERVER_IP>:443
    ```
    7. Verify data is flowing by logging into the new PMM Server UI and confirming that all monitored services appear in **Configuration > Inventory** with current metrics on dashboards.
    8. Restore any custom dashboards by importing the exported JSON files via **Dashboards > New > Import** in the Grafana UI.
    9. Recreate alert rules and contact points on the new server to match your previous setup.
    10. Decommission the OVF instance once everything is confirmed working.

    !!! note
        Historical metrics from the OVF deployment are not automatically transferred. If you need to preserve historical data, consider running both instances in parallel until the old data ages out of your retention window.