# Install PMM Client manually using binaries
This method allows you to install PMM Client using pre-compiled binary packages on a wide range of Linux distributions, for both x86_64 and ARM64 architectures.

Installing from binaries offers these advantages:

- supports Linux distributions not covered by package managers
- doesn't require package managers
- allows installation without root permissions (unique to this method)
- provides complete control over the installation location

!!! hint alert alert-success "Tip for quick installation"
    For a quick installation:

    - Download the PMM Client tar.gz file
    - Extract it
    - Run `./install_tarball` (or with `-u` flag to preserve existing config during upgrades)

## Prerequisites

Complete these essential steps before installation:
{.power-number}

1. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

2. [Install and configure PMM Server](../install-pmm-server/index.md) as you'll its IP address or hostname to configure the Client.

3. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

4. [Create database monitoring users](prerequisites.md#database-monitoring-requirements) with appropriate permissions for the databases you plan to monitor.

5. Check that you have root or sudo privileges to install PMM Client. Alternatively, use [binary installation](binary_package.md) for non-root environments.

!!! note "Version information"
    The commands below are for the latest PMM release. If you want to install a different release, make sure to update the commands with your required version number.

## Installation and setup
Binary installation adapts to your environment's permission model. Complete the installation first, then register your node for monitoring.

### Install PMM Client
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

### Register the node

After installing PMM Client, register your node with PMM Server to begin monitoring. This enables PMM Server to collect metrics and provide monitoring dashboards for your database infrastructure.

Registration requires authentication to verify that your PMM Client has permission to connect and send data to the PMM Server. PMM supports two authentication methods for registering the node: secure service account tokens and standard username/password credentials.

=== "Using Service accounts (Recommended)"
    [Service accounts](../../api/authentication.md) provide secure, token-based authentication for registering nodes with PMM Server. Unlike standard user credentials, service account tokens can be easily rotated, revoked, or scoped to specific permissions without affecting user access to PMM.

    To register with service accounts, create a service account then generate an authentication token that you can use to register the PMM Client:
    {.power-number}

    1. Log into PMM web interface.
    2. Navigate to **Administration > Users and access > Service Accounts**.
    3. Click **Add Service account**.
    4. Enter a descriptive name (e.g.: `pmm-client-prod-db01`). Keep in mind that PMM automatically shortens names exceeding 200 characters using a `{prefix}_{hash}` pattern.
    5. Select the **Editor** role from the drop-down. For detailed information about what each role can do, see [Role types in PMM](../../admin/roles/index.md).
    6. Click **Create > Add service account token**.
    7. (Optional) Name your token or leave blank for auto-generated name.
    8. (Optional) Set expiration date for enhanced security. Expired tokens require manual rotation. Permanent tokens remain valid until revoked.
    9. Click **Generate Token**.
    10. **Save your token immediately**. It starts with `glsa_` and won't be shown again!
    11. Register using the token:

        ```bash
        pmm-admin config --server-insecure-tls \
            --server-url=https://YOUR_PMM_SERVER:443 \
            --server-username=service_token \
            --server-password=YOUR_GLSA_TOKEN \
            [NODE_ADDRESS] [NODE_TYPE] [NODE_NAME]

        ```

        **Parameters explained:**

        - `--server-insecure-tls` - Skip certificate validation (remove for production with valid certificates)
        - `YOUR_PMM_SERVER` - Your PMM Server's IP address or hostname
        - `service_token` - Use this exact string as the username (not a placeholder!)
        - `YOUR_GLSA_TOKEN` - The token you copied (starts with `glsa_`)
        - `[NODE_ADDRESS]` - (Optional) IP address of the node being registered
        - `[NODE_TYPE]` - (Optional) Node type: `generic`, `container`, etc.
        - `[NODE_NAME]` - (Optional) Descriptive name for the node.


        ??? example "Full example with node details"
            ```bash
            pmm-admin config --server-insecure-tls \
                --server-url=https://192.168.33.14:443 \
                --server-username=service_token \
                --server-password=glsa_aBc123XyZ456... \
                192.168.33.23 generic prod-db01
            ```
            This registers node `192.168.33.23` with type `generic` and name `prod-db01`.

=== "Standard authentication (Not recommended)"

    This method exposes credentials in command history, process lists, and logs. Use only for testing or migration scenarios.

    ```bash
    pmm-admin config --server-insecure-tls \
    --server-url=https://admin:admin@YOUR_PMM_SERVER:443
    ```

    **Parameters explained:**

       - `YOUR_PMM_SERVER`- Your PMM Server's IP address or hostname
       - `443` - Default HTTPS port
       - `admin`/`admin` - Default PMM username and password (change this immediately after first login)

    ??? example "Registration with node details"
        Register a node with IP address `192.168.33.23`, type `generic`, and name `mynode`:

        ```bash
        pmm-admin config --server-insecure-tls \
        --server-url=https://admin:admin@192.168.33.14:443 \
        192.168.33.23 generic mynode
        ```
    To migrate to [service accounts](../../api/authentication.md):
    {.power-number}

    1. Create service accounts while still using standard authentication.
    2. Test service account tokens on non-critical nodes.
    3. Gradually migrate all nodes to token authentication.
    4. Change the admin password from default.
    5. Consider restricting or disabling direct admin account usage for node registration.

    !!! info "HTTPS requirement"
        PMM requires HTTPS connections (port `443` by default). HTTP URLs automatically redirect to HTTPS. For connection errors, verify:

        - Port `443` is accessible
        - Firewall rules allow HTTPS traffic
        - TLS certificates are valid (or use `--server-insecure-tls`)

## Verify the connection

Check that PMM Client is properly connected and registered:

```sh
pmm-admin status
```

## Related topics

- [Prerequisites for PMM Client](prerequisites.md)
- [Connect databases for monitoring](connect-database/index.md)
- [Uninstall PMM Client](../../uninstall-pmm/unregister_client.md)
- [Docker installation option](../install-pmm-client/docker.md) 
- [Package manager installation](package_manager.md) 