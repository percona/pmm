# Upgrade PMM Server Docker container
Upgrade your PMM Server Docker container to the latest version, ensuring you benefit from new features, improvements, and bug fixes while preserving your monitoring data and configuration.

!!! caution alert alert-warning "Important"
    Downgrades are not possible. To go back to using a previous version you must have [created a backup](../docker/backup_container.md) of it before upgrading.

## Prerequisite: check current version

Before you start upgrading, check current PMM Server version:

- **via UI**: use the **PMM Upgrade** panel on the **Home Dashboard**, or run the following command. For remote access, make sure to replace `localhost` with your PMM Server's address:
- **via CLI**:

    ```sh
    docker exec -it pmm-server \
    curl -ku admin:admin https://localhost/v1/version
    ```

## Upgrade procedure

To upgrade the container:
{.power-number}

1. Stop the current PMM Server container:

    ```sh
    docker stop pmm-server
    ```

2. Create a [backup](../docker/backup_container.md) of your current installation:

3. Pull the latest PMM Server image:

    ```sh
    docker pull percona/pmm-server:3
    ```

4. Rename the original container to keep it as a fallback: 

    ```sh
    docker rename pmm-server pmm-server-old
    ```

5. Run a new container with the latest image, connecting to your existing data. Make sure to adjust the volume parameter based on your setup (using `--volumes-from` for container data, `--volume pmm-data:/srv` for Docker volumes, or `--volume /path/on/host:/srv` for host directories):

    ```sh
    docker run \
    --detach \
    --restart always \
    --publish 443:8443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:3
    ```

6. Verify the upgrade was successful:
    ```sh
    docker exec -it pmm-server \
    curl -ku admin:admin https://localhost/v1/version
    ```

7. Access the PMM web interface and confirm your dashboards and monitoring are working correctly.

## Troubleshooting
If you encounter issues after upgrading:
{.power-number}

1. Check the PMM Server logs:
    ```sh
    docker logs pmm-server
    ```
2. If the upgrade fails, revert to your previous version:
    ```sh
    # Stop and remove the problematic container
    docker stop pmm-server
    docker rm pmm-server
    ```
    # Restore the backup
    docker rename pmm-server-backup pmm-server
    docker start pmm-server
    ```

## Automated Upgrades with Watchtower
If you installed [PMM Server with Watchtower](../docker/index.md#install-pmm-server--watchtower), you can u[pgrade directly from the PMM UI](../../../../pmm-upgrade/ui_upgrade.md). This method handles the entire upgrade process automatically, including pulling the new image and restarting the container.

## Related topics

- [Create a backup](../docker/backup_container.md) before upgrading
- [Restore from backup](../docker/restore_container.md) if needed