# Install PMM Client manually using binaries
This method allows you to install PMM Client using pre-compiled binary packages on a wide range of Linux distributions, for both x86_64 and ARM64 architectures.

Installing from binaries offers these advantages:

- supports Linux distributions not covered by package managers
- doesn't require package managers
- allows installation without root permissions (unique to this method)
- provides complete control over the installation location

## Prerequisites

Complete these essential steps before installation:
{.power-number}

1. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

2. [Install and configure PMM Server](../install-pmm-server/index.md)as you'll its IP address or hostname to configure the Client.

3. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

4. [Create database monitoring users](prerequisites.md#database-monitoring-requirements) with appropriate permissions for the databases you plan to monitor.

5. Check that you have root or sudo privileges to install PMM Client. Alternatively, use [binary installation](binary_package.md) for non-root environments.

!!! note "Version information"
    The commands below are for the latest PMM release. If you want to install a different release, make sure to update the commands with your required version number.

## Choose your installation path

Select the appropriate instructions based on your access level:
=== "With root permissions"
    To install with root/administrator privileges:
    {.power-number}

    1. Download the PMM Client package for your architecture:

        === "For x86_64 (AMD64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm3/{{release}}/binary/tarball/pmm-client-{{release}}-x86_64.tar.gz
            ```

        === "For ARM64 (aarch64)"
            ```sh 
            wget https://downloads.percona.com/downloads/pmm3/{{release}}/binary/tarball/pmm-client-{{release}}-aarch64.tar.gz
            ```

    2. Download the corresponding checksum file to verify integrity:

        === "For x86_64 (AMD64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm3/{{release}}/binary/tarball/pmm-client-{{release}}-x86_64.tar.gz.sha256sum
            ```

        === "For ARM64 (aarch64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm3/{{release}}/binary/tarball/pmm-client-{{release}}-aarch64.tar.gz.sha256sum
            ```

    3. Verify the download:

        === "For x86_64 (AMD64)"
            ```sh
            sha256sum -c pmm-client-{{release}}-x86_64.tar.gz.sha256sum
            ```

        === "For ARM64 (aarch64)"
            ```sh
            sha256sum -c pmm-client-{{release}}-aarch64.tar.gz.sha256sum
            ```

    4. Unpack the package and move into the directory:

        === "For x86_64 (AMD64)"
            ```sh
            tar xfz pmm-client-{{release}}-x86_64.tar.gz && cd pmm-client-{{release}}
            ```

        === "For ARM64 (aarch64)"
            ```sh
            tar xfz pmm-client-{{release}}-aarch64.tar.gz && cd pmm-client-{{release}}
            ```

    5. Set the installation directory:

        ```sh
        export PMM_DIR=/usr/local/percona/pmm
        ```

    6. Run the installer:

        ```sh
        sudo ./install_tarball
        ```

    7. Update your PATH:

        ```sh
        PATH=$PATH:$PMM_DIR/bin
        ```
    8. Create symbolic links to make PMM commands available system-wide:

        ```sh
        sudo ln -s /usr/local/percona/pmm/bin/pmm-agent /usr/local/bin/pmm-agent
        sudo ln -s /usr/local/percona/pmm/bin/pmm-admin /usr/local/bin/pmm-admin
        ```
    9. Set up the agent:

        ```sh
        sudo pmm-agent setup --config-file=/usr/local/percona/pmm/config/pmm-agent.yaml --server-address=192.168.1.123 --server-insecure-tls --server-username=admin --server-password=admin
        ```

    10. Run the agent:

        ```sh
        sudo pmm-agent --config-file=${PMM_DIR}/config/pmm-agent.yaml
        ```
    11. Register your nodes to be monitored by PMM Server using the PMM Client:

        ```sh
        sudo pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
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
            sudo pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.33.14:443 192.168.33.23 generic mynode
            ```
    12. Verify the installation in a new terminal:

        ```sh
        sudo pmm-admin status
        ```

=== "Without root permissions"

    Follow these steps for environments where you don't have root access:
    {.power-number}

    1. Download the PMM Client package for your architecture:

        === "For x86_64 (AMD64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm3/{{release}}/binary/tarball/pmm-client-{{release}}-x86_64.tar.gz
            ```

        === "For ARM64 (aarch64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm3/{{release}}/binary/tarball/pmm-client-{{release}}-aarch64.tar.gz
            ```

    2. Download the corresponding checksum file to verify integrity:

        === "For x86_64 (AMD64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm3/{{release}}/binary/tarball/pmm-client-{{release}}-x86_64.tar.gz.sha256sum
            ```

        === "For ARM64 (aarch64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm3/{{release}}/binary/tarball/pmm-client-{{release}}-aarch64.tar.gz.sha256sum
            ```

    3. Verify the download:

        === "For x86_64 (AMD64)"
            ```sh
            sha256sum -c pmm-client-{{release}}-x86_64.tar.gz.sha256sum
            ```
            
        === "For ARM64 (aarch64)"
            ```sh
            sha256sum -c pmm-client-{{release}}-aarch64.tar.gz.sha256sum
            ```

    4. Unpack the package and move into the directory:

        === "For x86_64 (AMD64)"
            ```sh
            tar xfz pmm-client-{{release}}-x86_64.tar.gz && cd pmm-client-{{release}}
            ```

        === "For ARM64 (aarch64)"
            ```sh
            tar xfz pmm-client-{{release}}-aarch64.tar.gz && cd pmm-client-{{release}}
            ```
    
    5. Set the installation directory:

        ```sh
        export PMM_DIR=YOURPATH
        ```

        Replace YOURPATH with a path where you have required access.

    6. Run the installer:

        ```sh
        ./install_tarball
        ```

    7. Update your PATH:

        ```sh
        PATH=$PATH:$PMM_DIR/bin
        ```

    8. Set up the agent:

        ```sh
        pmm-agent setup --config-file=${PMM_DIR}/config/pmm-agent.yaml --server-address=192.168.1.123 --server-insecure-tls --server-username=admin --server-password=admin --paths-tempdir=${PMM_DIR}/tmp --paths-base=${PMM_DIR}
        ```

    9. Run the agent:

        ```sh
        pmm-agent --config-file=${PMM_DIR}/config/pmm-agent.yaml
        ```
    10. Register your nodes to be monitored by PMM Server using the PMM Client:

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

    11. Open a new terminal and verify the installation:

        ```sh
        pmm-admin status
        ```
        
!!! hint alert alert-success "Tip for quick installation"
    For a quick installation:

    - Download the PMM Client tar.gz file
    - Extract it
    - Run `./install_tarball` (or with `-u` flag to preserve existing config during upgrades)

## Related topics

- [Prerequisites for PMM Client](prerequisites.md)
- [Connect databases for monitoring](connect-database/index.md)
- [Uninstall PMM Client](../../uninstall-pmm/unregister_client.md)
- [Docker installation option](../install-pmm-client/docker.md) 
- [Package manager installation](package_manager.md) 