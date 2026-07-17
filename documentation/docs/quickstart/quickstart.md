# Get started with PMM

Get a full PMM setup running with a database under monitoring using just two commands: one to install PMM Server, one to install PMM Client and connect your first database.

The steps below use the PMM Server easy-install script and the Quick-Install PMM Client feature (Technical Preview).

??? info "Alternative installation options"
    For more control over your setup (different deployment methods, non-Docker environments, or production configurations), see [Install PMM Server](../install-pmm/install-pmm-server/index.md) and [Install PMM Client](../install-pmm/install-pmm-client/index.md).

    If you're not using Docker, you can also deploy PMM Server on:

    - [Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md)
    - [Docker (manual setup)](../install-pmm/install-pmm-server/deployment-options/docker/index.md)
    - [Virtual Appliance](../install-pmm/install-pmm-server/deployment-options/virtual/index.md)
    - [Kubernetes/OpenShift via Helm](../install-pmm/install-pmm-server/deployment-options/helm/index.md)
    - [AWS Marketplace](../install-pmm/install-pmm-server/deployment-options/aws/deploy_aws.md)

#### Prerequisites

Before you start installing PMM, verify that your system meets the compatibility requirements:

??? info "Verify system compatibility"
    - System: Linux-compatible system with `sudo` privileges or `root` access
    - Network: Internet connectivity to download PMM components
    - Ports: Your system's firewall should allow TCP traffic on port `443`

## Step 1: Install PMM Server

The easy-install script only runs on Linux-compatible systems. To use it, run the command with `sudo` privileges or as `root`:
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

## Step 2: Install PMM Client

Once PMM Server is running, install PMM Client on your database host and connect your first database using the **Quick-Install PMM Client** page in the PMM UI.
{.power-number}

1. In the PMM sidebar, under **Inventory**, click **Install PMM Client**.

2. Select your database technology: **MySQL**, **PostgreSQL**, **MongoDB**, or **Valkey**.

3. Click **Generate token**.

4. Copy the generated command and run it on your database host with `sudo`. PMM Client is installed, the node registered, and your database added for monitoring in one step.

For full details, including credential options and advanced settings, see [Quick-Install PMM Client](../install-pmm/install-pmm-client/one-click-install.md).

For all other installation methods or to connect additional database types, see [PMM Client installation overview](../install-pmm/install-pmm-client/index.md).

## Next steps

- [Connect database services](../install-pmm/install-pmm-client/connect-database/index.md)
- [Configure PMM via the interface](../configure-pmm/configure.md)
- [Manage users in PMM](../admin/manage-users/index.md)
- [Set up roles and permissions](../admin/roles/index.md)
- [Back up and restore data in PMM](../backup/index.md)
