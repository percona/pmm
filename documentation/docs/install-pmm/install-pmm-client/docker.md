# Run PMM Client as a Docker container

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) is a convenient way to run PMM Client as a preconfigured [Docker](https://docs.docker.com/get-docker/) container. 

The PMM Client Docker image is available for both x86_64 and ARM64 architectures. Docker will automatically pull the correct image for your system architecture.
{.power-number}

1. Pull the PMM Client Docker image:

    ```sh
      docker pull percona/pmm-client:3
    ```

2. Create a Docker volume to store persistent data:
   ```sh
    docker volume create pmm-client-data
   ```

3. Execute the following command to start the [pmm-agent](../../use/commands/pmm-agent.md) in Setup mode. Replace `X.X.X.X` with the IP address of your PMM Server:

    ```sh
      PMM_SERVER=X.X.X.X:443
      docker run \
      --rm \
      --name pmm-client \
      -e PMM_AGENT_SERVER_ADDRESS=${PMM_SERVER} \
      -e PMM_AGENT_SERVER_USERNAME=admin \
      -e PMM_AGENT_SERVER_PASSWORD=admin \
      -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
      -e PMM_AGENT_SETUP=1 \
      -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
      -v pmm-client-data:/srv \
      percona/pmm-client:3
    ```
    !!! hint alert-success "Important"
       - Do not use the `docker --detach` option with Docker, as the pmm-agent logs output directly to the console, and detaching the container will prevent you from seeing these logs:
      - You can find a complete list of compatible environment variables [here](../../use/commands/pmm-agent.md).

3. Run the following command to verify the PMM client status. You should also see an increase in the number of monitored nodes in the PMM user interface:
    ```sh
      docker exec -t pmm-client pmm-admin status
    ```

You can now add services with [`pmm-admin`](../../use/commands/pmm-admin.md) by prefixing commands with `docker exec pmm-client`.

!!! hint alert alert-success "Tips for Docker configuration"

   - Firewall and routing rules: Ensure your host's firewall and routing rules are configured to allow Docker communications. This is crucial for Docker containers to communicate properly. For more details, see to the [troubleshooting checklist](../../troubleshoot/checklist.md).

   - Help command: If you need assistance with PMM Client, you can run the following command to display help information: `docker run --rm percona/pmm-client:3 --help`.

View your monitored node:
{.power-number}

    1. Go to the main menu and select **Operating System (OS) > Overview**.

    2. In the **Node Names** drop-down menu, select the node you recently registered.

    3. Modify the time range to view the relevant data for your selected node.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.
