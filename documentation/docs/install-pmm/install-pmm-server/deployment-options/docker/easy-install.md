# Run Docker via the Easy-install script

## Security best practice
You can download and check the installation script before running it from our [Github](https://www.percona.com/get/pmm).

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

 1. [Install Docker manually](../docker/index.md#installation-options).
 2. Run the Easy-install script above again.

This two-step approach resolves most installation issues, especially on Rocky Linux where automatic installation may fail.

### Next steps
Start by installing PMM Client:

[Install PMM Client :material-arrow-right:](../../../install-pmm-client/index.md){.md-button}
