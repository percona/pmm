# Remove container

!!! caution alert alert-warning "Caution"
    These steps delete the PMM Server Docker image and any accumulated PMM metrics data.

To remove the container:
{.power-number}

1. Stop pmm-server container:

    ```sh
    docker stop pmm-server
    ```

2. Remove the container:

    ```sh
    docker rm pmm-server
    ```

3. Remove the data volume:

    ```sh
    docker volume rm pmm-data
    ```

4. Remove the image:

    ```sh
    docker rmi $(docker images | grep "percona/pmm-server" | awk '{print $3}')
    ```
