## Run Docker via the Easy-install script
!!! caution alert alert-warning "Caution"
    You can download and check the installation script before running it from our [Github](https://www.percona.com/get/pmm):


### Docker installation on Rocky Linux
When using the Easy-install script on Rocky Linux, you may encounter `ERROR: Unsupported distribution 'rocky'`. This occurs because the Docker installation script doesn't explicitly support Rocky Linux. In this case, you'll need to [install Docker manually](../docker/index.md#installation-options) before running the Easy-install script.

## Linux or macOS
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
### Next steps
Start by installing PMM Client:

[Install PMM Client :material-arrow-right:](../../../install-pmm-client/index.md){.md-button}
