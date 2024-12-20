
# Run Docker with the host directory

To run Docker with the host directory:
{.power-number}

1. Pull the image:

    ```sh
    docker pull percona/pmm-server:3
    ```

2. Run the image:

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

3. Change the password for the default `admin` user:

    ```sh
    docker exec -t pmm-server change-admin-password <new_password>
    ```

4. Check the [WatchTower prerequisites](../docker/index.md|#prerequisites) and pass the following command to Docker Socket to start [Watchtower](https://containrrr.dev/watchtower/):

    ```sh
    docker run -v /var/run/docker.sock:/var/run/docker.sock -e WATCHTOWER_HTTP_API_UPDATE=1 -e WATCHTOWER_HTTP_API_TOKEN=your_watchtower_token --hostname=your_watchtower_host --network=pmm_default docker.io/perconalab/watchtower
    ```

5. Visit `https://localhost:443` to see the PMM user interface in a web browser. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)

## Migrate from data container to host directory/volume

To migrate your PMM from data container to host directory or volume run the following command:

```sh
docker cp <containerId>:/srv /target/host/directory
```
