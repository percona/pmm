# Install PMM Client with Package Manager
Percona Monitoring and Management (PMM) Client can be installed using standard Linux package managers. You can choose between automated repository setup or manual package download options.

## Prerequisites

Complete these essential steps before installation:
{.power-number}

1. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

2. [Install and configure PMM Server](../install-pmm-server/index.md)as you'll its IP address or hostname to configure the Client.

3. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

4. [Create database monitoring users](prerequisites.md#database-monitoring-requirements) with appropriate permissions for the databases you plan to monitor.

5. Check that you have root or sudo privileges to install PMM Client. Alternatively, use [binary installation](binary_package.md) for non-root environments.

## Supported architectures and platforms
PMM Client supports:

- Architectures: x86_64 (AMD64) and ARM64 (aarch64)
- Operating systems:

    - Red Hat/CentOS/Oracle Linux 8 and 9
    - Debian 11 (Bullseye) and 12 (Bookworm)
    - Ubuntu 22.04 (Jammy) and 24.04 (Noble)
    - Amazon Linux 2023

The package manager will automatically select the appropriate version for your system architecture.

## Installation process

### Step 1: Configure repositories

Choose your preferred method to configure the Percona repositories:

=== "Automatic (Recommended)"
    Use the `percona-release` utility to automatically configure repositories:

    !!! hint alert alert-success "Tip"
        If you have used `percona-release` before, disable and re-enable the repository:
        ```sh
        percona-release disable all
        percona-release enable pmm3-client
        ```

    === "Debian-based"
        ```sh
        wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
        sudo dpkg -i percona-release_latest.generic_all.deb
        sudo percona-release enable pmm3-client
        ```

    === "Red Hat-based"
        ```sh
        yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
        percona-release enable pmm3-client
        ```

=== "Manual download"
    Download packages directly without configuring repositories:
    {.power-number}

    1. Visit the [PMM download page](https://www.percona.com/downloads/).
    2. Select PMM 3 and choose specific version (usually the latest).
    3. Under **Select Platform**, select the item matching your software platform and architecture (x86_64 or ARM64).
    4. Download the package file or copy the link and use `wget` to download it.

### Step 2: Install PMM Client

!!! hint "Root permissions required"
    The installation commands below require root privileges. Use `sudo` if you're not running as root.

=== "From repository"
    === "Debian-based"
        ```sh
        sudo apt update
        sudo apt install -y pmm-client
        ```

    === "Red Hat-based"
        ```sh
        yum install -y pmm-client
        ```

=== "From downloaded package"
    === "Debian-based"
        ```sh
        sudo dpkg -i pmm-client_*.deb
        ```

    === "Red Hat-based"
        ```sh
        sudo dnf localinstall pmm-client-*.rpm
        ```

### Step 3: Verify installation

Check that PMM Client installed correctly:

```sh
pmm-admin --version
```

### Step 4: Register the node

Register your nodes to be monitored by PMM Server using the PMM Client:

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
```

where: 

- `X.X.X.X` is the address of your PMM Server
- `443` is the default port number
- `admin`/`admin` is the default PMM username and password. This is the same account you use to log into the PMM user interface, which you had the option to change when first logging in.

!!! caution alert alert-warning "HTTPS connection required"
    Nodes *must* be registered with the PMM Server using a secure HTTPS connection. If you try to use HTTP in your server URL, PMM will automatically attempt to establish an HTTPS connection on port 443. If a TLS connection cannot be established, you will receive an error message and must explicitly use HTTPS with the appropriate secure port.

??? info "Registration example"

    Register a node with IP address 192.168.33.23, type generic, and name mynode on a PMM Server with IP address 192.168.33.14:

    ```sh
    pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.33.14:443 192.168.33.23 generic mynode
    ```

## Related topics

- [Register a PMM Client](../register-client-node/index.md) 
- [Install PMM Client using Docker](../install-pmm-client/docker.md) 
- [Connect database services](../install-pmm-client/connect-database/index.md) 
- [PMM Client command reference](../../use/commands/pmm-admin.md) 
- [Upgrade PMM Client](../../pmm-upgrade/upgrade_client.md) 
- [Uninstall PMM Client](../../uninstall-pmm/unregister_client.md)
- [Unregister PMM Client](../../uninstall-pmm/unregister_client.md)
