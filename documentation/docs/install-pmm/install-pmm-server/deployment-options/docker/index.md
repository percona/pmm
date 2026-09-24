# Install PMM Server with Docker

Deploy PMM Server as a Docker container for a fast, flexible and isolated setup.

## Prerequisites
Before installation, ensure you have:

- Docker version 17.03 or higher
- CPU with `x86-64-v2` support
- [Sufficient system resources](../../../plan-pmm-installation/hardware_and_system.md) (recommended: 2+ CPU cores, 4+ GB RAM, 100+ GB disk space)

## Installation options

### Container setup summary

??? info "Container setup at a glance"
    - **Pull the Docker image**: `docker pull percona/pmm-server:3`
    - **Choose storage**: Docker volumes (recommended) or host directory
    - **Run the container**: Using the appropriate `docker run` command
    - **Access the UI**: Navigate to `https://SERVER_IP_ADDRESS` in your browser
    - **Log in**: Default credentials `admin` / `admin`

### Install PMM Server

You can install PMM Server using one of two methods:

=== "Easy-install script (Recommended for simplicity)"

    The [Easy-install script](../docker/easy-install.md) provides a one-step installation of PMM Server. Run the following command:

      ```sh
      curl -fsSL https://www.percona.com/get/pmm | /bin/bash
      ```

=== "Manual installation (For customization)"
    For a more customizable setup, follow these steps:
    {.power-number}

    1. Run PMM Server with Docker based on your preferred data storage method:
         - [Run Docker with host directory](../docker/run_with_host_dir.md)
         - [Run Docker with volume](../docker/run_with_vol.md)

    ## Configuration options

    ### Storage configuration

    You can choose either of two storage options offered by PMM Server:

    | Option | Suitable for | Docker parameter |
    |--------|-------------|---------|
    | [Docker volumes](../docker/run_with_vol.md) (Recommended) | Production environments | `--volume pmm-data:/srv` |
    | [Host directory](../docker/run_with_host_dir.md) | Development/testing | `--volume /path/on/host:/srv` |


    ### Environment variables

    Configure PMM Server's behavior using environment variables:

    ```sh
    docker run -e PMM_DATA_RETENTION=720h -e PMM_DEBUG=true percona/pmm-server:3
    ```

    Common variables:

    | Variable | Default | Description |
    |----------|---------|-------------|
    | `PMM_DATA_RETENTION` | `30d` | Duration to retain metrics data |
    | `PMM_METRICS_RESOLUTION` | `1s` | Base metrics collection interval |
    | `PMM_ENABLE_UPDATES` | `true` | Allow version checks |
    | `PMM_ENABLE_TELEMETRY` | `true` | Send usage statistics |

    For a complete list, see the [environment variables](../docker/env_var.md).

## Access PMM Server

After installation:
{.power-number}

1. Access the PMM interface in your browser: `https://SERVER_IP_ADDRESS` (replace with your server's address)

2. Log in with default credentials: `admin` / `admin`. 

3. Change the default password on first login.

## Advanced configuration
After basic installation, you may want to customize your PMM Server setup:

### Security options
- Configure a [trusted SSL certificate](../../../../admin/security/ssl_encryption.md) to remove browser warnings.
- Disable updates if needed:

    - **via Docker**:  add `-e PMM_ENABLE_UPDATES=0` to the `docker run` command (for the life of the container)
    - **via UI**: go to **Configuration > Settings > Advanced settings** and disable **Check for Updates** (can be turned back on by any admin in the UI)

- Enable HTTP (insecure, NOT recommended): add `--publish 80:8080` to the `docker run` command.

!!! info "Warning"
    PMM Client requires a secure (TLS-encrypted) connection and will only communicate with PMM Server over HTTPS.

## Next steps
- [Install PMM Client on hosts you want to monitor](../../../install-pmm-client/index.md)
- [Connect databases for monitoring](../../../install-pmm-client/connect-database/index.md)
