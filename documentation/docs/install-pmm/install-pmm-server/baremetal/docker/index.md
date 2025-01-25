# Install PMM Server with Docker container

This section provides instructions for running PMM Server with Docker based on the [PMM Docker image](https://hub.docker.com/r/percona/pmm-server).

## Running PMM Server with Watchtower

To enable PMM Server upgrades via the **Upgrade page** and the **Upgrade Now** button on the Home dashboard, you must configure Watchtower during the PMM Server installation. Watchtower is a container monitoring tool that helps update Docker containers to their latest version when triggered.

The [Easy-install script](../easy-install.md) script includes Watchtower commands, allowing for a one-step setup of PMM alongside Watchtower.

You can also install PMM 3 manually, following the instructions below.

## Installing PMM Server manually

Before starting the installation, review the installation prerequisites below and choose a method to run PMM Server with Docker based on your preferred data storage option:

- [Running Docker with host directory](../docker/run_with_host_dir.md)
- [Running Docker with volume](../docker/run_with_vol.md)

### Manual installation prerequisites

- Install [Docker](https://docs.docker.com/get-docker/) version 17.03 or higher.
- Ensure your CPU (and any virtualization layer you may be using) supports `x86-64-v2`.
- Install Watchtower to automatically update your containers with the following considerations:

      - Ensure Watchtower is only accessible from within the Docker network or local host to prevent unauthorized access and enhance container security.
      - Configure network settings to expose only the PMM Server container to the external network, keeping Watchtower isolated within the Docker network.
      - Grant Watchtower access to the Docker socket to monitor and manage containers effectively, ensuring proper security measures are in place to protect the Docker socket.
      - Verify that both Watchtower and PMM Server are on the same network, or ensure PMM Server can connect to Watchtower for communication. This network setup is essential for PMM Server to initiate updates through Watchtower.

## Run Docker container

??? info "Summary"

    !!! summary alert alert-info ""
        - Pull the Docker image.
        - Choose how you want to store data.
        - Run the image.
        - Open the PMM UI in a browser.

    ---
??? info "Key points"

    - To disable the Home Dashboard **PMM Upgrade** panel you can either add `-e PMM_ENABLE_UPDATES=false` to the `docker run` command (for the life of the container) or navigate to _PMM --> PMM Settings --> Advanced Settings_ and disable "Check for Updates" (can be turned back on by any admin in the UI).

    - Eliminate browser certificate warnings by configuring a [trusted certificate](https://docs.percona.com/percona-monitoring-and-management/how-to/secure.html#ssl-encryption).

    - You can optionally enable an (insecure) HTTP connection by adding `--publish 80:80` to the `docker run` command. However, running PMM insecure is not recommended. You should also note that PMM Client *requires* TLS to communicate with the server, only working on a secure port.
