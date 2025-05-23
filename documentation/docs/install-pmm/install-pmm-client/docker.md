# Run PMM Client as a Docker container

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) provides a convenient way to run PMM Client as a pre-configured container without installing software directly on your host system.

Using the Docker container approach offers several advantages:

- No need to install PMM Client directly on your host system
- Consistent environment across different operating systems
- Simplified setup and configuration process
- Automatic architecture detection (x86_64/ARM64)

## Prerequisites
Before you begin, make sure you have:

- [Docker Engine](https://docs.docker.com/get-docker/) installed and running
- Network connectivity to your PMM Server
- Basic familiarity with Docker commands

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

3. Start the [pmm-agent](../../use/commands/pmm-agent.md) in Setup mode to connect to PMM Server. Replace `X.X.X.X` with the IP address of your PMM Server:

    ```sh
     docker run \
     --rm \
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

4. After the setup is complete, start the [pmm-agent](../../use/commands/pmm-agent.md) in normal mode:

    ```sh
      docker run \
      --detach \
      --name pmm-client \
      -e PMM_AGENT_SETUP=0 \
      -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
      -v pmm-client-data:/usr/local/percona/pmm/tmp \
      percona/pmm-client:3
    ```
5. Verify the PMM Client status. If the connection is successful, you should also see an increased number of monitored nodes in the PMM user interface:

    ```sh
    docker exec -t pmm-client pmm-admin status
    ```

## Add monitoring services

After installing PMM Client, you can add database services to monitor with [`pmm-admin`](../../use/commands/pmm-admin.md). 

When running PMM in Docker, prefix all pmm-admin commands with `docker exec pmm-client`.

!!! hint alert alert-success "Tips for Docker configuration"

    - Firewall and routing rules: Ensure your host's firewall and routing rules are configured to allow Docker communications. This is crucial for Docker containers to communicate properly. For more details, see to the [troubleshooting checklist](../../troubleshoot/checklist.md).
    - Help command: If you need assistance with PMM Client, you can run the following command to display help information: `docker run --rm percona/pmm-client:3 --help`.

## View your monitored node
To confirm your node is being monitored:
{.power-number}

  1. Go to the main menu and select **Operating System (OS) > Overview**.

  2. In the **Node Names** drop-down menu, select the node you recently registered.

  3. Modify the time range to view the relevant data for your selected node.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.
