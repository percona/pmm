# Install PMM server with Docker container

This section provides instructions for running PMM Server with Docker based on our [Docker image](https://hub.docker.com/r/percona/pmm-server).

!!! note alert alert-primary ""
    The tags used here are for the current release. Other [tags](https://hub.docker.com/r/percona/pmm-server/tags) are available.

!!! seealso alert alert-info "See also"
    [Easy-install script](../easy-install.md)

**Prerequisites**

- Install [Docker](https://docs.docker.com/get-docker/) 1.12.6 or higher.
- For PMM 2.38.0 or greater, ensure your CPU (and any virtualization layer you may be using) supports `x86-64-v2`


## Run docker container

??? info "Summary"

    !!! summary alert alert-info ""
        - Pull the Docker image.
        - Copy it to create a persistent data container.
        - Run the image.
        - Open the PMM UI in a browser.

    ---

You can store data from PMM in:

1. [Docker volume](run_with_vol.md) (Preffered method)
2. [Data container](run_with_data_container.md)
3. [Host directory](run_with_host_dir.md)


??? info "Key points"

    - To Disable the Home Dashboard *PMM Upgrade* panel you can either add `-e DISABLE_UPDATES=true` to the `docker run` command (for the life of the container) or navigate to _PMM --> PMM Settings --> Advanced Settings_ and disable "Check for Updates" (can be turned back on by any admin in the UI).

    - Eliminate browser certificate warnings by configuring a [trusted certificate](https://docs.percona.com/percona-monitoring-and-management/how-to/secure.html#ssl-encryption).

    - You can optionally enable an (insecure) HTTP connection by adding `--publish 80:80` to the `docker run` command. However, running PMM insecure is not recommended. You should also note that PMM Client *requires* TLS to communicate with the server, only working on a secure port.
