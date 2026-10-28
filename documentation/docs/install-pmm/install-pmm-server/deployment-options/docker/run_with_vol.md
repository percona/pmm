
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

3. Run PMM Server with the volume configured:

    ```sh
    docker run --detach --restart always \
    --publish 443:8443 \
    --volume pmm-data:/srv \
    --network=pmm-network \
    --name pmm-server \
    percona/pmm-server:3
    ```

4. Set a secure password for the default `admin` user. You can do this either at startup using environment variables, or after startup using a helper script:

    - **At startup** (add to the `docker run` command in step 3):

        ```sh
        --env GF_SECURITY_ADMIN_PASSWORD="your_secure_password"
        ```

    - **After startup:**

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
        --env PMM_DATA_RETENTION=336h \
        --env PMM_METRICS_RESOLUTION=5s \
        --volume pmm-data:/srv \
        --name pmm-server \
        percona/pmm-server:3
    ```

    `PMM_DATA_RETENTION` must be a whole number of days, given in hours: at least `24h`, in multiples of `24h`, such as `720h` for 30 days. Values in days, such as `30d`, stop PMM Server from starting.

- **Port mapping** to expose PMM Server on a different port:

    ```sh 
    docker run ... --publish 8443:8443 ... percona/pmm-server:3
    ```

For a complete list of configuration options, see the [full list of environment variables](../docker/env_var.md).

## Next steps

- [Install PMM Client](../../../install-pmm-client/index.md) to start monitoring your database instances
- [Set up backups](../../../../backup/index.md) to protect your monitoring data
- [Configure SSL certificates](../../../../admin/security/ssl_encryption.md) for secure communications
