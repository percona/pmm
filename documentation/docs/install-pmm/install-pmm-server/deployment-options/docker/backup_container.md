
# Back up PMM Server Docker container

Regular backups of your PMM Server are essential for protecting your monitoring configuration and historical data.

## Backup overview
??? info "Summary"

    - Stop and rename the `pmm-server` container.
    - Take a local copy of the `pmm-server` container's `/srv` directory.
    - Copy the dat`a directory (`/srv`) to your host
    - Resume normal operations
    

## Backing up Grafana plugins 
Grafana plugins have been moved to the `/srv` directory since PMM 2.23.0. So if you are upgrading PMM from a version before 2.23.0 and have installed additional plugins, you'll need to reinstall them after the upgrade.
    
To check used Grafana plugins:

```sh
docker exec -t pmm-server ls -l /var/lib/grafana/plugins
```

## Back up procedure

To back up your PMM Server container:
{.power-number}

1. Stop the running PMM Server container:

    ```sh
    docker stop pmm-server
    ```

2. Rename the container to preserve it as a backup source:

    ```sh
    docker rename pmm-server pmm-server-backup
    ```

3. Create a backup subdirectory (e.g., `pmm-data-backup`) and navigate to it:

    ```sh
    mkdir pmm-data-backup && cd pmm-data-backup
    ```

4. Back up the data:

    ```sh
    docker cp pmm-server-backup:/srv .
    ```

5. Verify the backup was created successfully:
    ```sh
    ls -la srv/
    ```

## Next steps after backup  

After creating your backup, you have two options:
{.power-number}

1. Resume normal operations if you were creating a routine backup, restart your original container.
2. [Upgrade](../docker/upgrade_container.md) or [restore the container](../docker/restore_container.md) if you were backing up before an upgrade or restoration.

## Backup storage recommendations

- Store backups in a location separate from the PMM Server host
- Implement automated rotation of backups to manage disk space
- Consider encrypting backups containing sensitive monitoring data
- Test restores periodically to verify backup integrity