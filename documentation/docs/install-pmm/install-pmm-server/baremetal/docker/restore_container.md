# Restore container
You can restore PMM Server either from a manual backup or from an automated backup volume that was created during migration to PMM v3.

!!! caution alert alert-warning "Important"
    You must have either a [manual backup](backup_container.md) or an [automated backup volume](../../../../pmm-upgrade/migrating_from_pmm_2.md#step-2-migrate-pmm-2-server-to-pmm-3) to restore from.

=== "Restore from manual backup"
    To restore the container from a manual backup:
    {.power-number}

    1. Stop the container:

        ```sh
        docker stop pmm-server
        ```

    2. Remove the container:

        ```sh
        docker rm pmm-server
        ```

    3. Revert to the saved image:

        ```sh
        docker rename pmm-server-backup pmm-server
        ```

    4. Change directory to the backup directory (e.g. `pmm-data-backup`):

        ```shc
        cd pmm-data-backup
        ```

    5. Copy the data:

        ```sh
        docker run --rm -v $(pwd)/srv:/backup -v pmm-data:/srv -t percona/pmm-server:3 cp -r /backup/* /srv
        ```

    6. Restore permissions:

        ```sh
        docker run --rm -v pmm-data:/srv -t percona/pmm-server:3 chown -R pmm:pmm /srv
        ```

    7. Start the image:

        ```sh
        docker start pmm-server
        ```

=== "Restore from automated backup"

    If you need to restore from an automated backup volume created during [migration to PMM3](../../../../pmm-upgrade/migrating_from_pmm_2.md#step-2-migrate-pmm-2-server-to-pmm-3):
    {.power-number}

    1. Stop the current PMM3 container:
        ```sh
        docker stop pmm-server
        ```
    2. Remove the container (optional):
        ```sh
        docker rm pmm-server
        ```
    3. Start a PMM2 container using your backup volume, replacing   `<backup-volume-name>` with your PMM2 backup volume name (e.g., `pmm-data-2025-01-16-165135`):

        ```sh
        docker run -d \
        -p 443:443 \
        --volume <backup-volume-name>:/srv \
        --name pmm-server \
        --restart always \
        percona/pmm-server:2.44.0
        ```

    4. Verify that your PMM2 instance is running correctly and all your data is accessible.

    !!! note alert alert-primary "Finding your backup volume name"
    - If you used the [automated upgrade script](../../../../pmm-upgrade/migrating_from_pmm_2.md#step-2-migrate-pmm-2-server-to-pmm-3) (`get-pmm.sh -b`), the backup volume name was displayed during the upgrade process.
    - To list all available Docker volumes, use:
        ```sh
        docker volume ls       
        ```