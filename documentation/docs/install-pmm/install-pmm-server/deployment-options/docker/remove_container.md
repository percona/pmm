# Remove PMM Server Docker container

Completely remove PMM Server from your Docker environment when you want to uninstall PMM Server, free up resources, or prepare for a clean installation.


!!! danger "Warning: Data loss"
    These steps will permanently delete your PMM Server container, Docker image, all stored metrics data, and configuration. This action cannot be undone unless you have a backup.

    Consider [creating a backup](backup_container.md) first if you might need the data in the future.

To completely remove the container from your system:
{.power-number}

1. Stop the running PMM Server container:

    ```sh
    docker stop pmm-server
    ```

2. Remove the container (preserving data volume):

    ```sh
    docker rm pmm-server
    ```

3. Remove the data volume containing all metrics and configuration:

    ```sh
    docker volume rm pmm-data
    ```

4. Remove the PMM Server Docker image:

    ```sh
    docker rmi $(docker images | grep "percona/pmm-server" | awk '{print $3}')
    ```

## Verification

Verify that all PMM Server components have been removed. If you successfully removed everything, these commands should return no results:

```sh
# Check if the container is gone
docker ps -a | grep pmm-server

# Verify the volume is removed
docker volume ls | grep pmm-data

# Confirm the image is removed
docker images | grep percona/pmm-server
```

## Selective removal options

If you need to remove only specific components:

=== "Remove container but keep data"

    This allows for future reinstallation without losing historical data:

    ```sh
    docker stop pmm-server
    docker rm pmm-server
    # Do NOT remove the volume
    ```

=== "Remove container and image but keep data"

    ```sh
    docker stop pmm-server
    docker rm pmm-server
    docker rmi $(docker images | grep "percona/pmm-server" | awk '{print $3}')
    # Do NOT remove the volume
    ```

## Related topics

- [Backup PMM Server](../docker/backup_container.md) to create a backup before removal
- [Restore PMM Server](../docker/remove_container.md) to restore from a backup if needed later
- [Install PMM Server](index.md) to reinstall PMM Server if needed