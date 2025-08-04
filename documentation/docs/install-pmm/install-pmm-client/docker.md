# Run PMM Client as a Docker container

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) provides a convenient way to run PMM Client as a pre-configured container without installing software directly on your host system.

Using the Docker container approach offers several advantages:

- no need to install PMM Client directly on your host system
- consistent environment across different operating systems
- simplified setup and configuration process
- automatic architecture detection (x86_64/ARM64)
- [centralized configuration management](../install-pmm-server/deployment-options/docker/env_var.md#configure-vmagent-variables) through PMM Server environment variables

## Prerequisites
Complete these essential steps before installation:
{.power-number}

1. Install [Docker Engine](https://docs.docker.com/get-docker/)

2. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

3. [Install and configure PMM Server](../install-pmm-server/index.md) as you'll need its IP address or hostname to configure the Client.

4. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

5. [Create database monitoring users](prerequisites.md#database-monitoring-requirements) with appropriate permissions for the databases you plan to monitor.

6. Check that you have root or sudo privileges to install PMM Client. Alternatively, use [binary installation](binary_package.md) for non-root environments.

## Installation and setup

Follow these steps to deploy PMM Client using Docker:
{.power-number}

1. Pull the PMM Client Docker image:

    ```sh
    docker pull percona/pmm-client:3
    ```

2. Create a persistent Docker volume to store PMM Client data between container restarts:

    ```sh
    docker volume create pmm-client-data
    ```

3. Start the PMM Client container and configure the [pmm-agent](../../use/commands/pmm-agent.md) in Setup mode to connect to PMM Server. Replace `X.X.X.X` with the IP address of your PMM Server and update `PMM_AGENT_SERVER_PASSWORD` value if you changed the default `admin` password during setup:

    ```sh
     docker run \
     --name pmm-client \
     -e PMM_AGENT_SERVER_ADDRESS=X.X.X.X:443 \
     -e PMM_AGENT_SERVER_USERNAME=admin \
     -e PMM_AGENT_SERVER_PASSWORD=admin \
     -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
     -e PMM_AGENT_SETUP=1 \
     -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
     -e PMM_AGENT_SETUP_FORCE=1 \
     -v pmm-client-data:/usr/local/percona/pmm/tmp \
     percona/pmm-client:3
    ```

    !!! hint alert-success "Important"
         - Do not use the `docker --detach` option with this command. The pmm-agent outputs logs directly to the console, and detaching would prevent you from seeing important setup information and potential errors.
         - You can find a complete list of compatible environment variables [here](../../use/commands/pmm-agent.md).
         -   If you get `Failed to register pmm-agent on PMM Server: connection refused`, this typically means that the IP address is incorrect or the PMM Server is unreachable.

4. After the setup is complete, start the [pmm-agent](../../use/commands/pmm-agent.md) in normal mode:

    ```bash
    docker run \
      --detach \
      --name pmm-client \
      -e PMM_AGENT_SETUP=0 \
      -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
      -v pmm-client-data:/usr/local/percona/pmm/tmp \
      percona/pmm-client:3
    ```
    
5. Register your nodes to be monitored by PMM Server using the PMM Client:

    ```sh
    docker exec pmm-client pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
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
        docker exec pmm-client pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.33.14:443 192.168.33.23 generic mynode
        ```

6. Verify the PMM Client status. If the connection is successful, you should also see an increased number of monitored nodes in the PMM user interface:

    ```bash
    docker exec -t pmm-client pmm-admin status
    ```

## Add monitoring services

After installing PMM Client, you can add database services to monitor with [`pmm-admin`](../../use/commands/pmm-admin.md). 

When running PMM in Docker, prefix all pmm-admin commands with `docker exec pmm-client`.

!!! hint alert alert-success "Tips for Docker configuration"

    - Ensure your host's firewall and routing rules are configured to allow Docker communications. This is crucial for Docker containers to communicate properly. For more details, see to the [troubleshooting checklist](../../troubleshoot/checklist.md).
    - If you need assistance with PMM Client, run: `docker run --rm percona/pmm-client:3 --help`.

## View your monitored node
To confirm your node is being monitored:
{.power-number}

  1. Go to the main menu and select **Operating System (OS) > Overview**.

  2. In the **Node Names** drop-down menu, select the node you recently registered.

  3. Modify the time range to view the relevant data for your selected node.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.
