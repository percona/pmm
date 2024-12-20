## Run Docker via the Easy-install script

!!! caution alert alert-warning "Caution"
    You can download and check `get-pmm.sh` before running it from our [github]:

## Linux or macOS

Download and install PMM Server using `cURL` or `wget`:

=== "cURL"

    ```sh
    export PMM_REPO=perconalab/pmm-server PMM_TAG=3.0.0-beta
    curl -fsSL https://raw.githubusercontent.com/percona/pmm/refs/heads/v3/get-pmm.sh | /bin/bash
    ```

=== "wget"

    ```sh
    export PMM_REPO=perconalab/pmm-server PMM_TAG=3.0.0-beta
    wget -O - https://raw.githubusercontent.com/percona/pmm/refs/heads/v3/get-pmm.sh | /bin/bash
    ```


??? info "What does the script do?"
     This script does the following:

    - Installs Docker if it is not already installed on your system.
    - Stops and renames any currently running PMM Server Docker container from `pmm-server` to `pmm-server-{timestamp}`. This old pmm-server container is not a recoverable backup.
    - Pulls and runs the latest PMM Server Docker image.
    - Can run in Interactive mode to change the default settings:

        ```sh
        curl -fsSLO https://raw.githubusercontent.com/percona/pmm/refs/heads/v3/get-pmm.sh (or wget https://raw.githubusercontent.com/percona/pmm/refs/heads/v3/get-pmm.sh)
        chmod +x pmm
        ./pmm --interactive
        ```

[github]: https://github.com/percona/pmm/blob/v3/get-pmm.sh

### Next steps

Start by installing PMM client:

[Install PMM client :material-arrow-right:](../../../install-pmm-client/index.md){.md-button}
