# Install PMM Client with Percona repositories
Percona Monitoring and Management (PMM) Client can be installed using standard Linux package managers. You can choose between automated repository setup or manual package download options.

## Supported architectures and platforms
PMM Client supports:

- **Architectures**: x86_64 (AMD64) and ARM64 (aarch64)
- **Operating systems**:
    - Red Hat/CentOS/Oracle Linux 8 and 9
    - Debian 11 (Bullseye) and 12 (Bookworm)
    - Ubuntu 22.04 (Jammy) and 24.04 (Noble)
    - Amazon Linux 2023

The package manager will automatically select the appropriate version for your system architecture.

## Installation options
Choose one of these installation methods:

- [Quick installation](#quick-installation-using-percona-repositories) (recommended): Using the percona-release utility to configure repositories
- [Manual package download](#manual-package-download): Direct download from Percona website

## Quick installation using Percona repositories
This method configures the Percona repository on your system and installs PMM Client using your distribution's package manager.


!!! hint alert alert-success "Tip"
    If you have used `percona-release` before, disable and re-enable the repository:
    ```sh
    percona-release disable all
    percona-release enable pmm3-client
    ```

On Debian or Red Hat Linux, install `percona-release` and use a Linux package manager (`apt`/`dnf`) to install PMM Client.

=== "Debian-based"
    To install PMM Client:
    {.power-number}

    1. Configure repositories:
        ```sh
        wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
        dpkg -i percona-release_latest.generic_all.deb
        ```
    2. Enable pmm3-client repository:
        ```sh
        percona-release enable pmm3-client
        ```
    3. Install the PMM Client package:
         !!! hint "Root permissions required"
            The installation commands below require root privileges. Use `sudo` if you're not running as root.
        
        ```sh
        apt update
        apt install -y pmm-client
        ```
    4. Verify the installation by checking the PMM Client version:
        ```sh
        pmm-admin --version
        ```
    5. [Register the node](..//register-client-node/index.md).

=== "Red Hat-based"
    To install PMM Client:
    {.power-number}

    1. Configure repositories:
        ```sh
        yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
        ```
    2. Enable pmm3-client repository:
        ```sh
        percona-release enable pmm3-client
        ```
    3. Install the PMM Client package:
        ```sh
        yum install -y pmm-client
        ```
    4. Verify the installation by checking the PMM Client version:
        ```sh
        pmm-admin --version
        ```
    5. [Register the node](../register-client-node/index.md).

## Manual package download

If you prefer to download and install the packages manually without configuring repositories:
{.power-number}

1. Visit the [PMM download page](https://www.percona.com/downloads/) page.
2. Select PMM 3 and choose specific version (usually the latest), under **Select Platform**, select the item matching your software platform and architecture (x86_64 or ARM64).
4. Click the link in the Package Download Option section and download the package file or copy the link and use `wget` to download it.

    === "Debian-based"
        ```sh
        dpkg -i *.deb
        ``` 

    === "Red Hat-based"
        ```sh
        dnf localinstall *.rpm
        ```

## Related topics

- [Register a PMM Client](../register-client-node/index.md) 
- [Install PMM Client using Docker](../install-pmm-client/docker.md) 
- [Connect database services](../install-pmm-client/connect-database/index.md) 
- [PMM Client command reference](../../use/commands/pmm-admin.md) 
- [Upgrade PMM Client](../../pmm-upgrade/upgrade_client.md) 
- [Uninstall PMM Client](../../uninstall-pmm/unregister_client.md)