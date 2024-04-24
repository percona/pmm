
# Run Docker with volume

To run Docker with volume:
{.power-number}

1. Pull the image:

    ```sh
    docker pull percona/pmm-server:2
    ```

2. Create a volume:

    ```sh
    docker volume create pmm-data
    ```

3. Run the image:

    ```sh
    docker run --detach --restart always \
    --publish 443:443 \
    -v pmm-data:/srv \
    --name pmm-server \
    percona/pmm-server:2
    ```

4. Change the password for the default `admin` user. This command is only compatible with PMM 2.27.0 and later. If you are using an earlier version of PMM, upgrade to a supported version before running this command:

    ```sh
    docker exec -t pmm-server change-admin-password <new_password>
    ```

5. Check the [WatchTower prerequisites](../docker/index.md|#prerequisites) and pass the following command to Docker Socket to start [Watchtower](https://containrrr.dev/watchtower/):

    ```sh
    docker run -v /var/run/docker.sock:/var/run/docker.sock -e WATCHTOWER_HTTP_API_UPDATE=1 -e WATCHTOWER_HTTP_API_TOKEN=123 --hostname=watchtower --network=pmm_default docker.io/perconalab/watchtower
    ```

6. Visit `https://localhost:443` to see the PMM user interface in a web browser. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)
