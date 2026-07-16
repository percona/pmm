# Get started with PMM

To get up and running with Percona Monitoring and Management (PMM) in no time, install PMM on Bare Metal using the Easy-install script for Docker.

This is the simplest and most efficient way to install PMM with Docker.

??? info "Alternative installation options"
     For alternative setups or if you're not using Docker, explore the additional installation options detailed in the **Setting up** chapter:

    - [Deploy on Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md)
    - [Deploy based on a Docker image](../install-pmm/install-pmm-server/deployment-options/docker/index.md)
    - [Deploy on Virtual Appliance](../install-pmm/install-pmm-server/deployment-options/virtual/index.md)
    - [Deploy on Kubernetes/OpenShift via Helm](../install-pmm/install-pmm-server/deployment-options/helm/index.md)
    - [Run a PMM instance hosted at AWS Marketplace](../install-pmm/install-pmm-server/deployment-options/aws/deploy_aws.md)

#### Prerequisites

Before you start installing PMM, verify that your system meets the compatibility requirements:

??? info "Verify system compatibility"
    - System: Linux-compatible system with `sudo` privileges or `root` access
    - Network: Internet connectivity to download PMM components
    - Ports: Your system's firewall should allow TCP traffic on port `443`

## Install PMM

The Easy-install script only runs on Linux-compatible systems. To use it, run the command with `sudo` privileges or as `root`:
{ .power-number }

1. Download and install PMM using `cURL` or `wget`:

    === "cURL"

        ```sh
        curl -fsSL https://raw.githubusercontent.com/percona/pmm/refs/heads/main/get-pmm.sh | /bin/bash
        ```

    === "wget"

        ```sh
        wget -qO - https://raw.githubusercontent.com/percona/pmm/refs/heads/main/get-pmm.sh | /bin/bash    
        ```

2. After the installation is complete, log into PMM with the default `admin:admin` credentials.

??? info "What's happening under the hood?"
     This script does the following:

     * Installs Docker if it is not installed on your system.
     * Stops and renames any currently running PMM Docker container from `pmm-server` to `pmm-server-{timestamp}`. This old `pmm-server` container is not a recoverable backup.
     * Pulls and runs the latest PMM Docker image.

## Install PMM Client

Once PMM Server is running, install PMM Client on your database host and connect your first database using the **Install PMM Client (One-step)** page in the PMM UI.

{.power-number}

1. In the PMM sidebar, under **Inventory**, click **Install PMM Client**.

2. Select your database technology: **MySQL**, **PostgreSQL**, **MongoDB**, or **Valkey**.

3. Click **Generate short-lived token**.

4. Copy the generated command and run it on your database host with `sudo`. The script installs PMM Client, registers the node with PMM Server, and adds the first monitored service in one step.

For full details, including credential options and advanced settings, see [Install PMM Client (One-step)](../install-pmm/install-pmm-client/one-click-install.md).

For all other installation methods or to connect additional database types, see [PMM Client installation overview](../install-pmm/install-pmm-client/index.md).

## Next steps

- [Connect database services](../install-pmm/install-pmm-client/connect-database/index.md)
- [Configure PMM via the interface](../configure-pmm/configure.md)
- [Manage users in PMM](../admin/manage-users/index.md)
- [Set up roles and permissions](../admin/roles/index.md)
- [Back up and restore data in PMM](../backup/index.md)
