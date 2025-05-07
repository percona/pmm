
# Run Docker with volume

To run Docker with volume:
{.power-number}

1. Pull the image:

    ```sh
    docker pull percona/pmm-server:3
    ```

2. Create a volume:

    ```sh
    docker volume create pmm-data
    ```

3. Run the image:

    ```sh
    docker run --detach --restart always \
    --publish 443:8443 \
    --env PMM_WATCHTOWER_HOST=your_watchtower_host:8080 \
    --env PMM_WATCHTOWER_TOKEN=your_watchtower_token \
    --volume pmm-data:/srv \
    --network=pmm-network \
    --name pmm-server \
    percona/pmm-server:3
    ```

4. Change the password for the default `admin` user, replacing `your_secure_password` with a strong, unique password:

    ```sh
    docker exec -t pmm-server change-admin-password your_secure_password
    ```

5. Visit `https://localhost:443` to see the PMM user interface in a web browser. If you are accessing the Docker host remotely, replace `localhost` with the IP or server name of the host.
