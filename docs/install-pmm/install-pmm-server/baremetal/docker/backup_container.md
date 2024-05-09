
# Backup container

??? info "Summary"

    !!! summary alert alert-info ""
        - Stop and rename the `pmm-server` container.
        - Take a local copy of the `pmm-data` container's `/srv` directory.

    ---

!!! caution alert alert-warning "Important"
    Grafana plugins have been moved to the data volume `/srv` since the 2.23.0 version. So if you are upgrading PMM from any version before 2.23.0 and have installed additional plugins then plugins should be installed again after the upgrade.
    
    To check used Grafana plugins:

    ```sh
    docker exec -it pmm-server ls /var/lib/grafana/plugins
    ```
To backup container:
{.power-number}

1. Stop the container:

    ```sh
    docker stop pmm-server
    ```

2. Move the image:

    ```sh
    docker rename pmm-server pmm-server-backup
    ```

3. Create a subdirectory (e.g., `pmm-data-backup`) and move to it:

    ```sh
    mkdir pmm-data-backup && cd pmm-data-backup
    ```

4. Back up the data:

    ```sh
    docker cp pmm-data:/srv .
    ```
