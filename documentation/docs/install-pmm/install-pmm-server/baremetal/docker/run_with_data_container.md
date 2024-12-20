
# Run Docker with data container

To run Docker with data container:
{.power-number}

1. Create a persistent data container:

    ```sh
    docker create --volume /srv \
    --name pmm-data \
    percona/pmm-server:3 /bin/true
    ```

    !!! caution alert alert-warning "Important"
        PMM Server expects the data volume to be `/srv`. Using any other value will result in **data loss** when upgrading.

        To check server and data container mount points:

        ```sh
        docker inspect pmm-data | grep Destination && \
        docker inspect pmm-server | grep Destination
        ```

2. Create the Docker network:

    ```sh
    docker network create pmm_default
    ```

3. Run the image:

    ```sh
    docker run --detach --restart always \
    --publish 443:8443 \
    --env PMM_WATCHTOWER_HOST=your_watchtower_host \
    --env PMM_WATCHTOWER_TOKEN=your_watchtower_token \
    --volumes-from pmm-data \
    --network=pmm_default \
    --name pmm-server \
    percona/pmm-server:3
    ```

4. Change the password for the default `admin` user, replacing `your_secure_password123` with a strong, unique password:

    ```sh
    docker exec -t pmm-server change-admin-password your_secure_password123
    ```

5. Check the [WatchTower prerequisites](../docker/index.md|#prerequisites) and pass the following command to Docker Socket to start [Watchtower](https://containrrr.dev/watchtower/):

    ```sh
    docker run -v /var/run/docker.sock:/var/run/docker.sock -e WATCHTOWER_HTTP_API_UPDATE=1 -e WATCHTOWER_HTTP_API_TOKEN=your_watchtower_token --hostname=your_watchtower_host --network=pmm_default docker.io/perconalab/watchtower
    ```

6. Visit `https://localhost:443` to see the PMM user interface in a web browser. If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.
