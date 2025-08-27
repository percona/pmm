# Install PMM Client with Package Manager
Percona Monitoring and Management (PMM) Client can be installed using standard Linux package managers. You can choose between automated repository setup or manual package download options.

## Prerequisites

Complete these essential steps before installation:
{.power-number}

1. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

2. [Install and configure PMM Server](../install-pmm-server/index.md) as you'll its IP address or hostname to configure the Client.

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
    - Debian-based: 
        ```sh
        sudo apt update
        sudo apt install -y pmm-client
        ```
    - Red Hat-based: 
        ```sh
        yum install -y pmm-client
        ```

=== "From downloaded package"
    - Debian-based: 
        ```sh
        sudo dpkg -i pmm-client_*.deb
        ```
    - Red Hat-based: 
        ```sh
        sudo dnf localinstall pmm-client-*.rpm
        ```


### Step 3: Verify installation

Check that PMM Client installed correctly:

```sh
pmm-admin --version
```

### Step 4: Register the node

After installing PMM Client, register your node with PMM Server to begin monitoring. This enables PMM Server to collect metrics and provide monitoring dashboards for your database infrastructure.

Registration requires authentication to verify that your PMM Client has permission to connect and send data to the PMM Server. PMM supports two authentication methods for registering the node: secure service account tokens and standard username/password credentials.

=== "Using Service accounts (Recommended)"
    [Service accounts](../../api/authentication.md) provide secure, token-based authentication for registering nodes with PMM Server. Unlike standard user credentials, service account tokens can be easily rotated, revoked, or scoped to specific permissions without affecting user access to PMM.

    To register with service accounts, create a service account then generate an authentication token that you can use to register the PMM Client:
    {.power-number}

    1. Log into PMM web interface.
    2. Navigate to **Administration > Users and access > Service Accounts**.
    3. Click **Add Service account**.
    4. Enter a descriptive name (e.g.: `pmm-client-prod-db01`). PMM automatically shortens names exceeding 200 characters using a `{prefix}_{hash}` pattern.
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
        --server-password=YOUR_GLSA_TOKEN
    ```

    **Parameters explained:**

    - `--server-insecure-tls` - Skip certificate validation (remove for production with valid certificates)
    - `YOUR_PMM_SERVER` - Your PMM Server's IP address or hostname
    - `service_token` - Use this exact string as the username (not a placeholder!)
    - `YOUR_GLSA_TOKEN` - The token you copied (starts with `glsa_`)

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

### Step 5: Verify the connection

Check that PMM Client is properly connected and registered:

```sh
pmm-admin status
```

## Related topics

- [Install PMM Client using Docker](../install-pmm-client/docker.md) 
- [Connect database services](../install-pmm-client/connect-database/index.md) 
- [PMM Client command reference](../../use/commands/pmm-admin.md) 
- [Upgrade PMM Client](../../pmm-upgrade/upgrade_client.md) 
- [Uninstall PMM Client](../../uninstall-pmm/index.md)
- [Unregister PMM Client](../../uninstall-pmm/unregister_client.md)