
# Backup container

??? info "Summary"

    !!! summary alert alert-info ""
        - Stop and rename the `pmm-server` container.
        - Take a local copy of the `pmm-server` container's `/srv` directory.

    ---

!!! caution alert alert-warning "Important"
    Grafana plugins have been moved to the `/srv` directory since the 2.23.0 version. So if you are upgrading PMM from any version before 2.23.0 and have installed additional plugins then plugins should be installed again after the upgrade.
    
    To check used Grafana plugins:

    ```sh
    docker exec -t pmm-server ls -l /var/lib/grafana/plugins
    ```

To back up the container:
{.power-number}

1. Stop the container:

    ```sh
    docker stop pmm-server
    ```

2. Rename the image:

    ```sh
    docker rename pmm-server pmm-server-backup
    ```

3. Create a subdirectory (e.g., `pmm-data-backup`) and change directory to it:

    ```sh
    mkdir pmm-data-backup && cd pmm-data-backup
    ```

4. Back up the data:

    ```sh
    docker cp pmm-server-backup:/srv .
    ```
