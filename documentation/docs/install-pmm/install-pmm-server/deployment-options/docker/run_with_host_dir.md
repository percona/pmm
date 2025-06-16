
# Run Docker with the host directory

!!! danger alert alert-danger "Not recommended for production environments"
    Using a host directory for PMM data persistence is not recommended for production environments. This approach may lead to permission issues, inconsistent backup behavior, and potential data corruption during upgrades. 
    
    For production deployments, we strongly recommend using [Docker volumes](./run_with_vol.md) instead, which provide better isolation, portability, and compatibility with Docker's ecosystem.

## When to use host directories
Host directory mounting can be useful in specific scenarios:

- development and testing environments
- when you need direct filesystem access to PMM data
- integration with existing host-based backup solutions
- migration from other deployment methods

## Installation steps
To deploy PMM Server using a host directory: 
{.power-number}

1. Pull the latest PMM Server image:
   ```sh
   docker pull percona/pmm-server:3
   ```

2. Create and identify a directory on the host where to store PMM data. For example, `/home/user/srv`.

3. Run the PMM Server with the host image mounted, making sure to replace `your_watchtower_token` with the token created during [Watchtower setup](../docker/index.md#installation-options): 

    ```sh
    docker run --detach --restart always \
    --publish 443:8443 \
    --env PMM_WATCHTOWER_HOST=http://your_watchtower_host:8080 \
    --env PMM_WATCHTOWER_TOKEN=your_watchtower_token \
    --volume /home/user/srv:/srv \
    --network=pmm-network \
    --name pmm-server \
    percona/pmm-server:3
    ```

4. Set a secure password for the default `admin` user, replacing `your_secure_password` with a strong, unique password:

    ```sh
    docker exec -t pmm-server change-admin-password your_secure_password
    ```

5. Access the PMM web interface at `https://localhost:443` in a web browser. If you're connecting from a different machine, replace `localhost` with your server's IP address or hostname.

## Migrate from data container to host directory

To migrate data from a Docker volume to a host directory:

```sh
docker cp <container-id>:/srv /target/host/directory
```

## Migrate from host directory to Docker volume
To migrate from a host directory to a Docker volume (recommended for production):
{.power-number}

1. Create a new Docker volume:
```sh
docker volume create pmm-data
```
2. Copy data from host directory to the volume:
```sh
  docker run --rm -v /path/on/host:/source -v pmm-data:/target alpine cp -a /source/. /target/
```

3. Update your container to use the volume instead of the host directory. 

## Next steps

- [Install PMM Client](../../../install-pmm-client/index.md) to start monitoring your database instances
- [Consider migrating to Docker volumes](../docker/run_with_vol.md) for production environments
- [Learn how to back up your PMM Server](../../../../backup/index.md)
