# Install PMM Client manually using binaries
This method allows you to install PMM Client using pre-compiled binary packages on a wide range of Linux distributions, for both x86_64 and ARM64 architectures.

Installing from binaries offers these advantages:

- Supports Linux distributions not covered by package managers
- Doesn't require package managers
- Allows installation without root permissions (unique to this method)
- Provides complete control over the installation location


!!! note "Version information"
The commands shown below are for the latest PMM release. If you want to install a different release, make sure to update the commands with your required version number.

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
        ./install_tarball
        ```

    7. Update your PATH:

        ```sh
        PATH=$PATH:$PMM_DIR/bin
        ```

    8. Set up the agent:

        ```sh
        pmm-agent setup --config-file=/usr/local/percona/pmm/config/pmm-agent.yaml --server-address=192.168.1.123 --server-insecure-tls --server-username=admin --server-password=admin
        ```

    9. Run the agent:

        ```sh
        pmm-agent --config-file=${PMM_DIR}/config/pmm-agent.yaml
        ```

    10. Open a new terminal and verify the installation:

        ```sh
        pmm-admin status
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

    10. Open a new terminal and verify the installation:

        ```sh
        pmm-admin status
        ```
        
!!! hint alert alert-success "Tip for quick installation"
    For a quick installation:

    - Download the PMM Client tar.gz file
    - Extract it
    - Run `./install_tarbal`l (or with `-u` flag to preserve existing config during upgrades)

## Related topics

- [Prerequisites for PMM Client](prerequisites.md)
- [Register client node](../register-client-node/index.md) 
- [Connect databases for monitoring](connect-database/index.md)
- [Uninstall PMM Client](../../uninstall-pmm/unregister_client.md)
- [Docker installation option](../install-pmm-client/docker.md) 
- [Package manager installation](package_manager.md) 