
# Run Docker with the host directory

To run Docker with the host directory:
{.power-number}

1. Pull the image:

    ```sh
    docker pull percona/pmm-server:3
    ```

2. Identify a directory on the host that you want to use to persist PMM data. For example, `/home/user/srv`.

3. Run the image:

    ```sh
    docker run --detach --restart always \
    --publish 443:8443 \
    --env PMM_WATCHTOWER_HOST=your_watchtower_host \
    --env PMM_WATCHTOWER_TOKEN=your_watchtower_token \
    --volume /home/user/srv:/srv \
    --network=pmm_default \
    --name pmm-server \
    percona/pmm-server:3
    ```

4. Change the password for the default `admin` user, replacing `your_secure_password` with a strong, unique password:

    ```sh
    docker exec -t pmm-server change-admin-password your_secure_password
    ```

5. Visit `https://localhost:443` to see the PMM user interface in a web browser. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)

## Migrate from data container to host directory

To migrate your PMM from data container to host directory, run the following command:

```sh
docker cp <container-id>:/srv /target/host/directory
```
