# Migrate from the virtual appliance (OVF)

PMM 3.9.0 removed the OVF virtual appliance. Existing appliances keep running, but they don't get new PMM versions. To keep upgrading PMM, move your PMM Server to a supported deployment, such as [Docker](../install-pmm/install-pmm-server/deployment-options/docker/index.md), [Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md), or [Helm](../install-pmm/install-pmm-server/deployment-options/helm/index.md).

The virtual appliance runs PMM Server as a `pmm-server` systemd user service that launches a Podman container, and keeps PMM data in `/home/admin/volume/srv`. To migrate, you copy this directory to the new PMM Server.

## Before you migrate

Before you start, prepare so that you can go back to the appliance if something goes wrong:
{.power-number}

1. Take a snapshot or backup of the appliance VM. Keep the appliance until you've confirmed that the migration worked.
2. Note your current PMM Server version. Check **Configuration > Updates** in the PMM UI, or run `pmm-admin status` on a connected PMM Client. Deploy the same version or a newer one on the new server.

## Migrate to Docker

To move your PMM data from the appliance to PMM Server on Docker:
{.power-number}

1. On the appliance, log in as the `admin` user and stop PMM Server:

    ```sh
    systemctl --user stop pmm-server
    ```

2. On the new host, copy the PMM data directory from the appliance, replacing `<APPLIANCE_IP>` with the appliance address:

    ```sh
    scp -r admin@<APPLIANCE_IP>:/home/admin/volume/srv /home/user/
    ```

    This creates `/home/user/srv` on the new host.

3. Give the directory the owner that PMM Server expects, user ID `1000` and group ID `0`:

    ```sh
    sudo chown -R 1000:0 /home/user/srv
    ```

4. Start PMM Server with this directory mounted, as described in [Run Docker with the host directory](../install-pmm/install-pmm-server/deployment-options/docker/run_with_host_dir.md):

    ```sh
    docker run --detach --restart always \
    --publish 443:8443 \
    --volume /home/user/srv:/srv \
    --name pmm-server \
    percona/pmm-server:3
    ```

5. Log in to the new PMM Server and check that your dashboards, alert rules, and **Inventory** look as they did on the appliance.

6. For production, [move the data from the host directory to a Docker volume](../install-pmm/install-pmm-server/deployment-options/docker/run_with_host_dir.md#migrate-from-host-directory-to-docker-volume).

## After you migrate

When the new PMM Server is running:
{.power-number}

1. Point each PMM Client to the new server, replacing `<YOUR_GLSA_TOKEN>` with a service account token and `<NEW_PMM_SERVER_IP>` with the new server address:

    ```sh
    pmm-admin config --server-insecure-tls --server-url=https://service_token:<YOUR_GLSA_TOKEN>@<NEW_PMM_SERVER_IP>:443
    ```

2. Check that all monitored services appear in **Inventory** and show current metrics on the dashboards.
3. Shut down the appliance.
