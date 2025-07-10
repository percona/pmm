# Install PMM Client with Package Manager
Percona Monitoring and Management (PMM) Client can be installed using standard Linux package managers. You can choose between automated repository setup or manual package download options.

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

Register your PMM Client node with your PMM Server:

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
```

For more information, see [Register your PMM Client node](../register-client-node/index.md).

## Related topics

- [Register a PMM Client](../register-client-node/index.md) 
- [Install PMM Client using Docker](../install-pmm-client/docker.md) 
- [Connect database services](../install-pmm-client/connect-database/index.md) 
- [PMM Client command reference](../../use/commands/pmm-admin.md) 
- [Upgrade PMM Client](../../pmm-upgrade/upgrade_client.md) 
- [Uninstall PMM Client](../../uninstall-pmm/unregister_client.md)