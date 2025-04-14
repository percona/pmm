# Run Docker via the Easy-install script

The Easy-install script provides the simplest way to deploy PMM Server with Docker, handling all the necessary setup steps automatically.

## Security best practice
Before running the script:

- Download the installation script from the official Percona domain: [https://www.percona.com/get/pmm](https://www.percona.com/get/pmm)

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
    curl -fsSL https://www.percona.com/get/pmm | /bin/bash
    ```
=== "wget"
    ```sh
    wget -O - https://www.percona.com/get/pmm | /bin/bash
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

If you encounter Docker installation issues with the Easy-install script (such as `ERROR: Unsupported distribution 'rocky' on Rocky Linux`):
{.power-number}

 1. [Install Docker manually](https://docs.docker.com/engine/install/)
 2. Run the Easy-install script above again

This two-step approach resolves most installation issues, especially on Rocky Linux where automatic installation may fail.

### Next steps
After deploying PMM Server successfully, continue by setting up PMM Client:

[Install PMM Client :material-arrow-right:](../../../install-pmm-client/index.md){.md-button}
