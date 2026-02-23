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

1. Install [Docker Engine](https://docs.docker.com/get-docker/).

2. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

3. [Install and configure PMM Server](../install-pmm-server/index.md) as you'll need its IP address or hostname to configure the Client.

4. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

5. [Create database monitoring users](prerequisites.md#database-monitoring-requirements) with appropriate permissions for the databases you plan to monitor.

6. Check that you have root or sudo privileges to install PMM Client. Alternatively, use [binary installation](binary_package.md) for non-root environments.

## Installation and setup

Register your node with PMM Server to begin monitoring. This enables PMM Server to collect metrics and provide monitoring dashboards for your database infrastructure.

Registration requires authentication to verify that your PMM Client has permission to connect and send data to the PMM Server. PMM supports two authentication methods for registering the node: secure service account tokens and standard username/password credentials.

### Deploy PMM Client 

Follow these steps to deploy PMM Client using Docker:
{.power-number}

1. Pull the PMM Client Docker image:

    ```sh
    docker pull percona/pmm-client:3
    ```

2. Start the PMM Client container and configure the [pmm-agent](../../use/commands/pmm-agent.md) in Setup mode to connect to PMM Server. Replace `X.X.X.X` with the external IP address of your PMM Server:
   
    === "Using Service accounts (Recommended)"
   
        [Service accounts](../../api/authentication.md) provide secure, token-based authentication for registering nodes with PMM Server. Unlike standard user credentials, service account tokens can be easily rotated, revoked, or scoped to specific permissions without affecting user access to PMM.
    
        To register with service accounts, create a service account then generate an authentication token that you can use to register the PMM Client:
        {.power-number}
    
        1. Log into PMM web interface.
        2. Navigate to **Administration > Users and access > Service Accounts**.
        3. Click **Add Service account**.
        4. Enter a descriptive name (e.g.: `pmm-client-prod-db01`). Keep in mind that PMM automatically shortens names exceeding 200 characters using a `{prefix}_{hash}` pattern.
        5. Select the **Admin** role from the drop-down. For detailed information about what each role can do, see [Role types in PMM](../../admin/roles/index.md).
        6. Click **Create > Add service account token**.
        7. (Optional) Name your token or leave blank for auto-generated name.
        8. (Optional) Set expiration date for enhanced security. Expired tokens require manual rotation. Permanent tokens remain valid until revoked.
        9. Click **Generate Token**.
        10. **Save your token immediately**. It starts with `glsa_` and won't be shown again!
        11. Run the container using the token:
    
            ```bash
             docker run \
             --name pmm-client \
             -e PMM_AGENT_SETUP_NODE_NAME=my_node_name \
             -e PMM_AGENT_SETUP_NODE_TYPE=container \
             -e PMM_AGENT_SERVER_ADDRESS=X.X.X.X:443 \
             -e PMM_AGENT_SERVER_USERNAME=service_token \
             -e PMM_AGENT_SERVER_PASSWORD=YOUR_GLSA_TOKEN \
             -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
             -e PMM_AGENT_SETUP=1 \
             -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
             -e PMM_AGENT_SETUP_FORCE=1 \
             -e PMM_AGENT_PRERUN_SCRIPT=/opt/percona/pmm-prerun.sh \
             percona/pmm-client:3
            ```
    
            **Parameters explained:**
    
            - `PMM_AGENT_SETUP_NODE_NAME` - (Optional) Descriptive name for the node
            - `PMM_AGENT_SETUP_NODE_TYPE` - (Optional) Node type: generic, container, etc.
            - `PMM_AGENT_SERVER_ADDRESS` - Your PMM Server’s IP address or hostname
            - `service_token` - Use this exact string as the username (not a placeholder!)
            - `YOUR_GLSA_TOKEN` - The token you copied (starts with `glsa_`)
            - `PMM_AGENT_SERVER_INSECURE_TLS` - Skip certificate validation (remove for production with valid certificates)
            - `PMM_AGENT_PRERUN_SCRIPT` - (Optional) See [Monitoring services](#add-monitoring-services)
    
    === "Standard authentication (Not recommended)"
   
        This method exposes credentials in command history, process lists, and logs! Use only for testing or migration scenarios:
    
        ```sh
         docker run \
         --name pmm-client \
         -e PMM_AGENT_SETUP_NODE_NAME=my_node_name \
         -e PMM_AGENT_SETUP_NODE_TYPE=container \
         -e PMM_AGENT_SERVER_ADDRESS=X.X.X.X:443 \
         -e PMM_AGENT_SERVER_USERNAME=admin \
         -e PMM_AGENT_SERVER_PASSWORD=admin \
         -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
         -e PMM_AGENT_SETUP=1 \
         -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
         -e PMM_AGENT_SETUP_FORCE=1 \
         -e PMM_AGENT_PRERUN_SCRIPT=/opt/percona/pmm-prerun.sh \
         percona/pmm-client:3
        ```
    
        **Parameters explained:**
   
        - `PMM_AGENT_SETUP_NODE_NAME` - (Optional) Descriptive name for the node
        - `PMM_AGENT_SETUP_NODE_TYPE` - (Optional) Node type: generic, container, etc.
        - `PMM_AGENT_SERVER_ADDRESS` - Your PMM Server’s IP address or hostname
        - `admin`/`admin` - Default PMM Server username and password (change this immediately after first login)
        - `PMM_AGENT_PRERUN_SCRIPT` - (Optional) See [Monitoring services](#add-monitoring-services)

        To migrate to [service accounts](../../api/authentication.md):
        {.power-number}
    
        1. Create service accounts while still using standard authentication.
        2. Test service account tokens on non-critical nodes.
        3. Gradually migrate all nodes to token authentication.
        4. Change the admin password from default.
        5. Consider restricting or disabling direct admin account usage for node registration.

    !!! danger alert alert-danger "Danger"
   
        `pmm-agent.yaml` contains sensitive credentials and should not be shared.
       
    !!! hint alert-success "Important"
   
         - Do not use the `docker --detach` option with this command. The pmm-agent outputs logs directly to the console, and detaching would prevent you from seeing important setup information and potential errors.
         - You can find a complete list of compatible environment variables [here](../../use/commands/pmm-agent.md).
         - If you get `Failed to register pmm-agent on PMM Server: connection refused`, this typically means that the IP address is incorrect or the PMM Server is unreachable.
    
## Verify the connection

Check that PMM Client is properly connected and registered. If the connection is successful, you should also see an increased number of monitored nodes in the PMM user interface:

```bash
docker exec -t pmm-client pmm-admin status
```

### View your monitored node

To confirm your node is being monitored:
{.power-number}

  1. Go to the main menu and select **Operating System (OS) > Overview**.

  2. In the **Node Names** drop-down menu, select the node you recently registered.

  3. Modify the time range to view the relevant data for your selected node.

## Add monitoring services

After installing PMM Client, you add database services to monitor with the [`pmm-admin`](../../use/commands/pmm-admin.md) command. 

When running PMM Client in Docker, you can put `pmm-admin` commands in a script and provide the location via the `PMM_AGENT_PRERUN_SCRIPT` argument. The `pmm-agent` will run those commands after registering with the PMM Server. Example:

```bash
 docker run \
 --name pmm-client \
 -e PMM_AGENT_SETUP_NODE_NAME=my_node_name \
 -e PMM_AGENT_SETUP_NODE_TYPE=container \
 -e PMM_AGENT_SERVER_ADDRESS=X.X.X.X:443 \
 -e PMM_AGENT_SERVER_USERNAME=service_token \
 -e PMM_AGENT_SERVER_PASSWORD=YOUR_GLSA_TOKEN \
 -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
 -e PMM_AGENT_SETUP=1 \
 -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
 -e PMM_AGENT_SETUP_FORCE=1 \
 -e PMM_AGENT_PRERUN_SCRIPT=/opt/percona/pmm-prerun.sh
 percona/pmm-client:3
```

!!! hint alert alert-success "Tips for Docker configuration"

    - Ensure your host's firewall and routing rules are configured to allow Docker communications. This is crucial for Docker containers to communicate properly. For more details, see to the [troubleshooting checklist](../../troubleshoot/checklist.md).
    - If you need assistance with PMM Client, run: `docker run --rm percona/pmm-client:3 --help`.

