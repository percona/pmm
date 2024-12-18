# Run PMM client as a Docker container

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) is a convenient way to run PMM Client as a preconfigured [Docker](https://docs.docker.com/get-docker/) container. 

The PMM Client Docker image is available for both x86_64 and ARM64 architectures. Docker will automatically pull the correct image for your system architecture.
{.power-number}

1. Pull the PMM Client Docker image:

    ```sh
    docker pull \
    percona/pmm-client:2
    ```

2. Use the image as a template to create a persistent data store that preserves local data when the image is updated:

    ```sh
    docker create \
    --volume /srv \
    --name pmm-client-data \
    percona/pmm-client:2 /bin/true
    ```

3. Run the container to start [pmm-agent](../../use/commands/pmm-agent.md) in setup mode. Set `X.X.X.X` to the IP address of your PMM Server. (Do not use the `docker --detach` option as PMM agent only logs to the console.)

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
    percona/pmm-client:2
    ```
!!! hint alert-success "Tips"
    You can find a complete list of compatible environment variables [here](../../use/commands/pmm-agent.md).

3. Check status.

    ```sh
    docker exec pmm-client \
    pmm-admin status
    ```

    In the PMM user interface you will also see an increase in the number of monitored nodes.

You can now add services with [`pmm-admin`](../../use/commands/pmm-admin.md) by prefixing commands with `docker exec pmm-client`.

!!! hint alert alert-success "Tips"
    - Adjust host firewall and routing rules to allow Docker communications. ([Read more](../../troubleshoot/checklist.md)
    - For help: `docker run --rm percona/pmm-client:2 --help`

    In the GUI:

    - Select {{icon.dashboards}} *PMM Dashboards* → {{icon.node}} *System (Node)* → {{icon.node}} *Node Overview*.
    - In the *Node Names* menu, select the new node.
    - Change the time range to see data.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.
