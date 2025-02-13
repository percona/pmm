# Run PMM Client as a Docker container

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) is a convenient way to run PMM Client as a preconfigured [Docker](https://docs.docker.com/get-docker/) container. 

The PMM Client Docker image is available for both x86_64 and ARM64 architectures. Docker will automatically pull the correct image for your system architecture.
{.power-number}

1. Pull the PMM Client Docker image:

    ```sh
      docker pull percona/pmm-client:3
    ```

2. Use the image as a template to create a persistent data store that preserves local data when the image is updated:

    ```sh
      docker create \
      --volume /srv \
      --name pmm-client-data \
      percona/pmm-client:3 /bin/true
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
      --volumes-from pmm-client-data \
      percona/pmm-client:3
    ```
    !!! hint alert-success "Important"
       - Do not use the `docker --detach` option with Docker, as the pmm-agent logs output directly to the console, and detaching the container will prevent you from seeing these logs:
      - You can find a complete list of compatible environment variables [here](../../use/commands/pmm-agent.md).

3. Check status:

    ```sh
      docker exec -t pmm-client pmm-admin status
    ```

    In the PMM user interface you will also see an increase in the number of monitored nodes.

You can now add services with [`pmm-admin`](../../use/commands/pmm-admin.md) by prefixing commands with `docker exec pmm-client`.

!!! hint alert alert-success "Tips for Docker configuration"
   - Firewall and routing rules: Ensure your host's firewall and routing rules are configured to allow Docker communications. This is crucial for Docker containers to communicate properly. For more details, see to the [troubleshooting checklist](https://chat.deepseek.com/a/troubleshoot/checklist.md).

   - Help command: If you need assistance with PMM Client, you can run the following command to display help information: `docker run --rm percona/pmm-client:3 --help`.

  Steps in the UI:
  {.power-number}

    1. Go to the main menu and select **Operating System (OS) > Overview**.

    2. In the **Node Names** drop-down menu, choose the new node you want to monitor.

    3. Modify the time range to view the relevant data for your selected node.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.
