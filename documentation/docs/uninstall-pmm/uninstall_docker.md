# Uninstall PMM client using Docker container

This completely removes the PMM Client Docker container, image, and all associated data from your system.

To remove (uninstall) PMM Client, do the following steps in Docker:

!!! caution alert alert-warning "Caution"
    These steps delete the PMM Client Docker image and client services configuration data.

## Prerequisites

- [Unregister PMM Client](unregister_client.md) from PMM Server
- Docker access on the system

To uninstall PMM Client with the Docker container:
{.power-number}

1. Stop the pmm-client container:

    ```sh
    docker stop pmm-client
    ```

2. Remove the container:

    ```sh
    docker rm pmm-client
    ```

3. Remove the PMM Client image:

    ```sh
    docker rmi $(docker images | grep "percona/pmm-client" | awk {'print $3'})
    ```

4. Remove the data volume:

    ```sh
    docker volume rm pmm-client-data
    ```