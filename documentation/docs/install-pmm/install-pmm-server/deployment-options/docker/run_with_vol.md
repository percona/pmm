
# Run PMM Server with Docker volumes (Recommended)
Docker volumes provide the recommended storage configuration for PMM Server in production environments. 

Volumes offer better isolation, portability, and compatibility with Docker's ecosystem compared to [host directory storage](../docker/run_with_host_dir.md).

## Installation steps

To deploy PMM Server using Docker volumes:
{.power-number}

1. Pull the latest PMM Server image:

    ```sh
    docker pull percona/pmm-server:3
    ```

2. Create a dedicated Docker volume:

    ```sh
    docker volume create pmm-data
    ```

3. Run PMM Server with the volume configured, making sure to replace `your_watchtower_token` with the token created during [Watchtower setup](../docker/index.md#installation-options): 

    ```sh
    docker run --detach --restart always \
    --publish 443:8443 \
    --env PMM_WATCHTOWER_HOST=your_watchtower_host \
    --env PMM_WATCHTOWER_TOKEN=your_watchtower_token \
    --volume pmm-data:/srv \
    --network=pmm-network \
    --name pmm-server \
    percona/pmm-server:3
    ```

4. Set a secure password for the default `admin` user, replacing `your_secure_password` with a strong, unique password:

    ```sh
    docker exec -t pmm-server change-admin-password your_secure_password
    ```

5. Access the PMM web interface at `https://localhost` in a web browser. 
If you are accessing the Docker host remotely, replace `localhost` with your server's IP address or hostname.

## Additional configuration options
You can further customize your PMM Server deployment with:

- **Environment variables** to configure metrics retention, resolution, and features: 

    ```sh
    docker run --detach --restart always \
        --publish 443:8443 \
        --env PMM_DATA_RETENTION=14d \
        --env PMM_METRICS_RESOLUTION=5s \
        --volume pmm-data:/srv \
        --name pmm-server \
        percona/pmm-server:3
    ```

- **Port mapping** to expose PMM Server on a different port:

    ```sh 
    docker run ... --publish 8443:8443 ... percona/pmm-server:3
    ```

For a complete list of configuration options, see the [full list of environment variables](../docker/env_var.md).

## Next steps

- [Install PMM Client](../../../install-pmm-client/index.md) to start monitoring your database instances
- [Set up backups](../../../../backup/index.md) to protect your monitoring data
- [Configure SSL certificates](../../../../admin/security/ssl_encryption.md) for secure communications