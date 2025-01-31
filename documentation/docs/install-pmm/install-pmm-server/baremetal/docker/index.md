# Install PMM Server with Docker container

This section explains how to install PMM Server as a Docker container. To enable PMM Server upgrades via the **Upgrade page** and the **Upgrade Now** button on the **Home** dashboard, you must configure Watchtower during the PMM Server installation.

Watchtower is a container-updating tool that enables [PMM Server upgrades](../../../../pmm-upgrade/ui_upgrade.md) through the UI. Without it, the **Upgrade** page and **Upgrade Now** button will not be available.

### Prerequisites

Before starting the installation:

* Install Docker version 17.03 or higher
* Ensure your CPU supports `x86-64-v2`
* For manual installation, consider these Watchtower security requirements:
  - Restrict Watchtower access to Docker network or localhost
  - Configure network to expose only PMM Server externally
  - Secure Docker socket access for Watchtower
  - Place both Watchtower and PMM Server on the same network

### Installation options

You can install PMM Server with Watchtower in two ways:

#### Easy-install script 

The [Easy-install script](../docker/easy-install.md) implifies setup by including Watchtower commands, enabling a one-step installation of PMM with Watchtower. Run the following command:
     ```sh
     curl -fsSL https://www.percona.com/get/pmm | /bin/bash
     ```

#### Manual installation

For a more customizable setup, follow these steps:
{.power-number}

1.  Create a Docker network for PMM and Watchtower:
   ```sh
   docker network create pmm-network
   ```

2. Install Watchtower using the command below. The <your_token> value must match the `WATCHTOWER_HTTP_API_TOKEN` value used in your PMM Server container configuration:

   ```sh
   docker run --detach \
   --restart always \
   --network=<your_network> \
   -e WATCHTOWER_HTTP_API_TOKEN=<your_token> \
   -e WATCHTOWER_HTTP_API_UPDATE=1 \
   --volume /var/run/docker.sock:/var/run/docker.sock \
   --name watchtower \
   percona/watchtower:latest
   ```

3 Install PMM Server (choose one storage option):

   - [Running Docker with host directory](../docker/run_with_host_dir.md)
   - [Running Docker with volume](../docker/run_with_vol.md)