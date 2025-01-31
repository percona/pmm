# Install PMM Server with Docker container

This section provides instructions for running PMM Server with Docker based on the PMM Docker image.

## Running PMM Server with Watchtower

To enable PMM Server upgrades via the **Upgrade page** and the **Upgrade Now** button on the Home dashboard, you must configure Watchtower during the PMM Server installation. Watchtower is a container monitoring tool that helps update Docker containers to their latest version when triggered.

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

2.  Install Watchtower:
   ```sh
   docker run -d \
     --name watchtower \
     --restart unless-stopped \
     --network pmm-network \
     -v /var/run/docker.sock:/var/run/docker.sock \
     containrrr/watchtower \
     --cleanup \
     --no-startup-message \
     --http-api-update \
     --http-api-token your_api_token
   ```

3Ö Install PMM Server (choose one storage option):
   - [Running Docker with host directory](../docker/run_with_host_dir.md)
   - [Running Docker with volume](../docker/run_with_vol.md)