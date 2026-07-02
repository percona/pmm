# Migrate from Virtual Appliance (OVF)

OVF virtual appliance support was removed in PMM 3.9.0 and no new OVA images are published. If you are still running an older PMM version on a virtual appliance, migrate to a supported deployment method.

The OVA runs PMM Server inside a Docker container, which means you can use the standard PMM backup and restore process to migrate. This preserves dashboards, alert rules, contact points, monitored services, and historical data without manual re-configuration.

## Step 1: Back up PMM Server
Back up the PMM Server data from your OVA instance so you can restore it on the new server:
{.power-number}

1. SSH into the virtual machine.

2. Stop PMM Server:

    ```bash
    docker stop pmm-server
    ```

3. Follow the [Back up PMM Server Docker container](../docker/backup_container.md) guide to create a backup archive of the `/srv` directory.

## Step 2: Deploy your new PMM Server

Deploy PMM Server on a **different host or IP address** from your current OVA, using one of the supported methods:

- **[Docker](../docker/index.md) (recommended)**: simplest migration path
- **[Podman](../podman/index.md)**: rootless containers for security-sensitive environments
- **[Helm](../helm/index.md)**: Kubernetes-native deployment with high availability support

Keep your OVA instance stopped while you set up the new server.

## Step 3: Restore data on the new PMM Server

Before starting the new PMM Server for the first time, restore the backup you created in Step 1. Follow the [Restore PMM Server Docker container](../docker/restore_container.md) guide to extract the backup data into the new Docker volume.

This restores all monitored services, dashboards, alert rules, and historical data from the OVA.

## Step 4: Reconfigure PMM Clients

Point each PMM Client to the new server so it sends monitoring data to the correct endpoint: `
pmm-admin config --server-insecure-tls \
  --server-url=https://service_token:<YOUR_GLSA_TOKEN>@<NEW_PMM_SERVER_IP>:443
`

## Step 5: Verify and decommission
Confirm the migration is complete, then shut down the old instance:
{.power-number}

1. Log into the new PMM Server UI and confirm that all monitored services appear in **Configuration > Inventory** with current metrics on dashboards.
2. Decommission the OVA instance once all services are reporting correctly.