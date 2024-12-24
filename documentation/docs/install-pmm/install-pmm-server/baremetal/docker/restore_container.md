# Restore container

??? info "Summary"

    !!! summary alert alert-info ""
        - Stop and remove the container.
        - Restore (rename) the backup container.
        - Restore saved data to the data volume.
        - Restore permissions to the directories.
        - Start the container.

    ---

!!! caution alert alert-warning "Important"
    You must have a [backup](backup_container.md) to restore from.

To restore the container:
{.power-number}

1. Stop the container.

    ```sh
    docker stop pmm-server
    ```

2. Remove it.

    ```sh
    docker rm pmm-server
    ```

3. Revert to the saved image.

    ```sh
    docker rename pmm-server-backup pmm-server
    ```

4. Change directory to the backup directory (e.g. `pmm-data-backup`).

    ```sh
    cd pmm-data-backup
    ```

5. Copy the data.

    ```sh
    docker run --rm -v $(pwd)/srv:/backup -v pmm-data:/srv -t perconalab/pmm-server:3.0.0-beta cp -r /backup/* /srv
    ```

6. Restore permissions.

    ```sh
    docker run --rm -v pmm-data:/srv -t perconalab/pmm-server:3.0.0-beta chown -R pmm:pmm /srv
    ```

7. Start the image.

    ```sh
    docker start pmm-server
    ```
