# Install PMM Client manually using binaries

Choose your installation instructions based on whether you have root permissions:

=== "With root permissions"
    To install PMM Client with **binary** package with root permissions:
    {.power-number}

1. Download the PMM Client package:

    ```sh
    wget https://downloads.percona.com/downloads/pmm2/{{release}}/binary/tarball/pmm2-client-{{release}}.tar.gz
    ```

2. Download the PMM Client package checksum file:

    ```sh
    wget https://downloads.percona.com/downloads/pmm2/{{release}}/binary/tarball/pmm2-client-{{release}}.tar.gz.sha256sum
    ```

3. Verify the download.

        ```sh
        pmm-agent --config-file=${PMM_DIR}/config/pmm-agent.yaml
        ```

4. Unpack the package and move into the directory.

        ```sh
        pmm-admin status
        ```

=== "Without root permissions"

    To install PMM Client with **binary** package without root permissions:
    {.power-number}

    1. Download the PMM Client package for your architecture:

        === "For x86_64 (AMD64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm/{{release}}/binary/tarball/pmm-client-{{release}}-x86_64.tar.gz
            ```

        === "For ARM64 (aarch64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm/{{release}}/binary/tarball/pmm-client-{{release}}-aarch64.tar.gz
            ```

    2. Download the corresponding checksum file:

        === "For x86_64 (AMD64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm/{{release}}/binary/tarball/pmm-client-{{release}}-x86_64.tar.gz.sha256sum
            ```

        === "For ARM64 (aarch64)"
            ```sh
            wget https://downloads.percona.com/downloads/pmm/{{release}}/binary/tarball/pmm-client-{{release}}-aarch64.tar.gz.sha256sum
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

6. Run the installer.

        ```sh
        ./install_tarball
        ```

7. Change the path.

        ```sh
        PATH=$PATH:$PMM_DIR/bin
        ```

8. Set up the agent (pick the command for you depending on permissions)

    !!! hint "Root permissions"
    ```sh
    pmm-agent setup --config-file=/usr/local/percona/pmm2/config/pmm-agent.yaml --server-address=192.168.1.123 --server-insecure-tls --server-username=admin --server-password=admin
    ```

        ```sh
        pmm-agent setup --config-file=${PMM_DIR}/config/pmm-agent.yaml --server-address=192.168.1.123 --server-insecure-tls --server-username=admin --server-password=admin --paths-tempdir=${PMM_DIR}/tmp --paths-base=${PMM_DIR}
        ```

9. Run the agent.

        ```sh
        pmm-agent --config-file=${PMM_DIR}/config/pmm-agent.yaml
        ```

10. Open a new terminal and check.

    ```sh
    pmm-admin status
    ```
    
!!! hint alert alert-success "Tips"
    - Download tar.gz with pmm-client.
    - Extract it.
    - Run `./install_tarball script `with the `-u` flag.

The configuration file will be overwritten if you do not provide the -`u` flag while the pmm-agent is updated.
