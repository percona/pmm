## Run Docker via the Easy-install script

!!! caution alert alert-warning "Caution"
    You can download and check `get-pmm.sh` before running it from our [github]:

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

[github]: https://github.com/percona/pmm/blob/main/get-pmm.sh

### Next steps

Start by installing PMM client:

[Install PMM client :material-arrow-right:](../../../install-pmm-client/index.md){.md-button}