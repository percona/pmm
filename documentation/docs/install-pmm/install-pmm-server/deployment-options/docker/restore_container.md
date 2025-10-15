# Restore PMM Server Docker container
You can restore PMM Server either from a manual backup or from an automated backup volume that was created during migration to PMM v3.

## Before you begin

Before proceeding with restoration, ensure you have one of the following:

- a manual backup you previously created. Make sure to verify its integrity using the verification procedures in the [back up guide](backup_container.md).
- an [automated backup volume](../../../../pmm-upgrade/migrating_from_pmm_2.md#step-2-migrate-pmm-2-server-to-pmm-3) created during migration from PMM V3

# Restore methods

Choose the restoration method that matches how your backup was created:

=== "Volume-to-volume backup"
    Restore from a backup volume created using the volume-to-volume method:
    {.power-number}

    1. Stop the current PMM Server container:
        ```sh
        docker stop pmm-server
        ```

    2. Remove the current container:
        ```sh
        docker rm pmm-server
        ```
    4. Choose a restauration option:
        - Replace current volume with backup volume:
            ```sh
            # Remove current volume (WARNING: This deletes current data)
            docker volume rm pmm-data

            # Restore from backup volume to new pmm-data volume
            docker volume create pmm-data
            sudo docker run --rm -v <backup-volume-name>:/from -v pmm-data:/to alpine ash -c 'cp -av /from/. /to'
            ```
        - Use backup volume directly:
            ```sh
            # Start PMM Server using backup volume directly
            docker run -d \
            --publish 443:8443 \
            --volume <backup-volume-name>:/srv \
            --name pmm-server \
            --restart always \
            percona/pmm-server:3
            ```

    4. Verify the restored PMM Server is working correctly:
        ```sh
        docker logs pmm-server
        ```
        
=== "Directory backup"
    Restore from a host directory backup:
    {.power-number}

    1. Stop the current PMM Server container:
        ```sh
        docker stop pmm-server
        ```

    2. Remove the current container:
        ```sh
        docker rm pmm-server
        ```

    3. Copy backup data to PMM volume:
        ```sh
        # Remove current volume (WARNING: This deletes current data)
        docker volume rm pmm-data
        
        # Create new pmm-data volume
        docker volume create pmm-data
        
        # Copy directory backup to volume
        docker run --rm -v $(pwd)/<backup-directory>:/backup -v pmm-data:/srv alpine sh -c 'cp -r /backup/* /srv/'
        ```

    4. Fix ownership of restored files:
        ```sh
        docker run --rm -v pmm-data:/srv -t percona/pmm-server:3 chown -R pmm:pmm /srv
        ```

    5. Start the restored PMM Server:
        ```sh
        docker run -d \
        --publish 443:8443 \
        --volume pmm-data:/srv \
        --name pmm-server \
        --restart always \
        percona/pmm-server:3
        ```

=== "Migration rollback"
    Rollback from PMM 3 to PMM 2 using automated migration backup:
    {.power-number}

    1. Stop the current PMM v3 container:
        ```sh
        docker stop pmm-server
        ```

    2. Remove the PMM v3 container:
        ```sh
        docker rm pmm-server
        ```

    3. Start a PMM v2 container using your backup volume:
        ```sh
        docker run -d \
        -p 443:443 \
        --volume <backup-volume-name>:/srv \
        --name pmm-server \
        --restart always \
        percona/pmm-server:2.44.0
        ```
        
        Replace `<backup-volume-name>` with your PMM v2 backup volume name (e.g., `pmm-data-2025-01-16-165135`).

    4. Verify that your PMM v2 instance is running correctly:
        ```sh
        docker logs pmm-server
        # Check that all your data is accessible via the web interface
        ```

=== "Universal container restore"
    Use this as a fallback method when:

    - you created a backup using `docker cp pmm-server-backup:/srv .`  
    - you have a backup directory with an `srv/` folder containing PMM data
    - you used the [**Universal container copy** backup option](../docker/backup_container.md)
    - other restore methods don't match your backup type

    To restore from a universal container:
    {.power-number}

    1. Stop the current PMM Server container:
        ```sh
        docker stop pmm-server
        ```

    2. Remove the container:
        ```sh
        docker rm pmm-server
        ```

    3. Restore the renamed backup container:
        ```sh
        docker rename pmm-server-backup pmm-server
        ```

    4. Navigate to the backup directory:
        ```sh
        cd pmm-data-backup-YYYYMMDD-HHMMSS
        ```

    5. Copy the backup data to the PMM data volume:
        ```sh
        docker run --rm -v $(pwd)/srv:/backup -v pmm-data:/srv -t percona/pmm-server:3 cp -r /backup/* /srv
        ```

    6. Fix ownership of the restored files:
        ```sh
        docker run --rm -v pmm-data:/srv -t percona/pmm-server:3 chown -R pmm:pmm /srv
        ```

    7. Start the restored PMM Server container:
        ```sh
        docker start pmm-server
        ```   

## Find your backup volume name

If you're restoring from an automated migration backup and don't know the volume name:

- your backup volume name was displayed during the [automated upgrade process](../../../../pmm-upgrade/migrating_from_pmm_2.md#step-2-migrate-pmm-2-server-to-pmm-3).
- to list all available Docker volumes, use the following command and look for volumes with names like `pmm-data-YYYY-MM-DD-HHMMSS`:

    ```sh
    docker volume ls       
    ```

## Next steps

- [Create a backup of your PMM Server](../docker/backup_container.md)
- [Upgrade your PMM Server](../docker/upgrade_container.md) to a newer version
- [Migrate from PMM v2 to v3](../../../../pmm-upgrade/migrating_from_pmm_2.md) if restoring to upgrade