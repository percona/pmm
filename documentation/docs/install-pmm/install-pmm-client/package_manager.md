# Install PMM Client with Percona repositories
PMM Client can be installed using standard Linux package managers. This page covers both automated repository setup and manual package download options.

## Supported architectures and platforms
PMM Client supports:

- **Architectures**: x86_64 (AMD64) and ARM64 (aarch64)
- **Operating systems**:
    - Red Hat/CentOS/Oracle Linux 8 and 9
    - Debian 11 (Bullseye) and 12 (Bookworm)
    - Ubuntu 20.04 (Focal), 22.04 (Jammy), and 24.04 (Noble)

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
        !!! hint "Root permissions"
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

## Package manager -- manual download

To manually download package manager:
{.power-number}

1. Visit the [Percona Monitoring and Management 3 download](https://www.percona.com/downloads/) page.
2. Under **Select Product Version** select the one you want (usually the latest).
3. Under **Select Platform**, select the item matching your software platform and architecture (x86_64 or ARM64).
4. Click to download the package file:
    - For Debian, Ubuntu: `.deb`
    - For Red Hat, CentOS, Oracle Linux: `.rpm`
(Alternatively, copy the link and use `wget` to download it.)

=== "Debian-based"
    ```sh
    dpkg -i *.deb
    ```

=== "Red Hat-based"
    ```sh
    dnf localinstall *.rpm
    ```

??? info "Download page links"
    Here are the download page links for each supported platform:

    - [Red Hat/CentOS/Oracle 8](https://www.percona.com/downloads/pmm3/{{release}}/binary/redhat/8/)
    - [Red Hat/CentOS/Oracle 9](https://www.percona.com/downloads/pmm3/{{release}}/binary/redhat/9/)
    - [Debian 11 (Bullseye)](https://www.percona.com/downloads/pmm3/{{release}}/binary/debian/bullseye/)
    - [Debian 12 (Bookworm)](https://www.percona.com/downloads/pmm3/{{release}}/binary/debian/bookworm/)
    - [Ubuntu 20.04 (Focal Fossa)](https://www.percona.com/downloads/pmm3/{{release}}/binary/debian/focal/)
    - [Ubuntu 22.04 (Jammy Jellyfish)](https://www.percona.com/downloads/pmm3/{{release}}/binary/debian/jammy/)
    - [Ubuntu 24.04 (Noble Numbat)](https://www.percona.com/downloads/pmm3/{{release}}/binary/debian/noble/)
