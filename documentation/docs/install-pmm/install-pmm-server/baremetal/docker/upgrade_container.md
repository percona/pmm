# Upgrade container

!!! caution alert alert-warning "Important"
    Downgrades are not possible. To go back to using a previous version you must have created a backup of it before upgrading.

!!! hint alert alert-success "Tip"
    To see what release you are running, use the **PMM Upgrade** panel on the **Home Dashboard**, or run the following command (replace localhost with your PMM server's address for remote access):

    ```sh
    docker exec -it pmm-server \
    curl -ku admin:admin https://localhost/v1/version
    ```

To upgrade the container:
{.power-number}


1. Stop the container:

    ```sh
    docker stop pmm-server
    ```

2. Perform a [backup](../docker/backup_container.md).


3. Pull the latest image:

    ```sh
    docker pull percona/pmm-server:3
    ```

4. Rename the original container:

    ```sh
    docker rename pmm-server pmm-server-old
    ```


5. Run it:

    ```sh
    docker run \
    --detach \
    --restart always \
    --publish 443:443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:3
    ```


