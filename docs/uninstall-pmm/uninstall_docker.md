# Uninstall PMM client using Docker container

To remove (uninstall) PMM Client, do the following steps in Docker:

!!! caution alert alert-warning "Caution"
    These steps delete the PMM Client Docker image and client services configuration data.

To uninstall PMM client with the Docker container:
{.power-number}

1. Stop pmm-client container.

    ```sh
    docker stop pmm-client
    ```

2. Remove containers.

    ```sh
    docker rm pmm-client
    ```

3. Remove the image.

    ```sh
    docker rmi $(docker images | grep "percona/pmm-client" | awk {'print $3'})
    ```

4. Remove the volume.

    ```sh
    docker volume rm pmm-client-data
    ```