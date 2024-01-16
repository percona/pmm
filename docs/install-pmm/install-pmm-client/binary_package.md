# Install PMM client manually

To install PMM client with **binary** package, do the following:
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
    sha256sum -c pmm2-client-{{release}}.tar.gz.sha256sum
    ```

4. Unpack the package and move into the directory.

    ```sh
    tar xfz pmm2-client-{{release}}.tar.gz && cd pmm2-client-{{release}}
    ```

5. Choose one of these two commands (depends on your permissions):

    !!! caution alert alert-warning "Without root permissions"
        ```sh
        export PMM_DIR=YOURPATH
        ```
        where YOURPATH replace with you real path, where you have required access.

    !!! caution alert alert-warning "With root permissions"
        ```sh
        export PMM_DIR=/usr/local/percona/pmm2
        ```

6. Run the installer.

    !!! hint "Root permissions (if you skipped step 5 for non root users)"
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

    !!! caution alert alert-warning "Non root users"
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
    - Download tar.gz with pmm2-client.
    - Extract it.
    - Run `./install_tarball script `with the `-u` flag.

The configuration file will be overwritten if you do not provide the -`u` flag while the pmm-agent is updated.
