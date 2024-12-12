# Remove container

??? info "Summary"

    !!! summary alert alert-info ""
        - Stop the container.
        - Remove (delete) both the server and data containers.
        - Remove (delete) both images.

    ---

!!! caution alert alert-warning "Caution"
    These steps delete the PMM Server Docker image and any accumulated PMM metrics data.

To remove the container:
{.power-number}

1. Stop pmm-server container.

    ```sh
    docker stop pmm-server
    ```

2. Remove containers.

    ```sh
    docker rm pmm-server pmm-data
    ```

3. Remove the image.

    ```sh
    docker rmi $(docker images | grep "percona/pmm-server" | awk {'print $3'})
    ```