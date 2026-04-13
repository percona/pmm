# Run Docker via the Easy-install script

The Easy-install script provides the simplest way to deploy PMM Server with Docker, handling all the necessary setup steps automatically.

## Security best practice
Before running the script:

- Download the installation script from the [official Percona domain](https://www.percona.com/get/pmm).

- Review the script content to understand its actions.

- Consider running the script with the `--interactive` flag to customize:
   
    - port mappings (default: 443 for HTTPS)
    - location where PMM Server stores its data
    - PMM Server version (specific version or latest)
    - additional configuration parameters (environment variables, resource limits)


## Installation instructions

### Linux or macOS
Download and install PMM Server using `cURL` or `wget`:
=== "cURL"
    ```sh
    curl -fsSL https://www.percona.com/get/pmm | bash
    ```
=== "wget"
    ```sh
    wget -O - https://www.percona.com/get/pmm | bash
    ```
??? info "What does the script do?"
     This script does the following:

    - Installs Docker if it is not already installed on your system.
    - Stops and renames any currently running PMM Server Docker container from `pmm-server` to `pmm-server-{timestamp}`. This old pmm-server container is not a recoverable backup.
    - Pulls and runs the latest PMM Server Docker image.
    - Can run in Interactive mode to change the default settings:

    ```sh
       curl -fsSLO https://www.percona.com/get/pmm (or wget https://www.percona.com/get/pmm) 
       chmod +x pmm
       ./pmm --interactive
    ```

## Docker installation issues

### Script fails on Rocky Linux or other unsupported distributions

If you encounter Docker installation issues with the Easy-install script (such as `ERROR: Unsupported distribution 'rocky' on Rocky Linux`):
{.power-number}

 1. [Install Docker manually](https://docs.docker.com/engine/install/)
 2. Run the Easy-install script above again

This two-step approach resolves most installation issues, especially on Rocky Linux where automatic installation may fail.

### Container keeps restarting with `/srv is not writable` error

If the PMM Server container keeps restarting and `docker logs pmm-server` shows:

```
FATAL: /srv is not writable for pmm user.
Please make sure that /srv is owned by uid 1000 and gid 0 and try again.
You can change ownership by running: sudo chown -R 1000:0 /srv
```

This is typically caused by incorrect ownership on the Docker volume used by PMM. To resolve this:
{.power-number}

1. Fix the ownership of the volume data directory:

    ```sh
    sudo chown -R 1000:0 /srv
    ```

    Then restart the container:

    ```sh
    docker restart pmm-server
    ```

    If the container starts successfully, no further steps are needed.

2. If the problem persists, stop and remove the PMM Server container:

    ```sh
    docker stop pmm-server && docker rm pmm-server
    ```

3. Remove the orphaned PMM data volume:

    ```sh
    docker volume rm pmm-data
    ```

    !!! warning
        This permanently deletes all PMM data stored in the volume, including dashboards, metrics history, and configuration. Only proceed if you intend to start fresh.

4. Run the Easy-install script again:

    ```sh
    curl -fsSL https://www.percona.com/get/pmm | bash
    ```

If the problem still persists, remove all unused Docker objects (containers, images, volumes, and networks) and try again:

```sh
docker system prune -a
```

!!! warning
    This removes **all** unused Docker resources on the host, not just those related to PMM. Only run this command if you are sure no other Docker workloads are affected.

### Next steps
After deploying PMM Server successfully, continue by setting up PMM Client:

[Install PMM Client :material-arrow-right:](../../../install-pmm-client/index.md){.md-button}
