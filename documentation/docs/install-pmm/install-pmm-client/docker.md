# Run PMM Client as a Docker container

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) provides a convenient way to run PMM Client as a pre-configured container without installing software directly on your host system.

Using the Docker container approach offers several advantages:

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

## Installation and setup

Set up PMM Client by deploying it as a Docker container and registering it with PMM Server.

### Deploy and register PMM Client

Deploy and register PMM Client to start monitoring your node. 

PMM identifies a Node by name, and the identity that PMM Server issues to the pmm-agent is stored in its configuration file. Keep both stable, a fixed `PMM_AGENT_SETUP_NODE_NAME` and a volume for the configuration, and the Node together with the Services you add survives a restart, a re-created container, or an image upgrade.

!!! caution alert alert-warning "Requires PMM Client 3.10.0 or later"
    Earlier versions register the Node again on every container start, and PMM Server answers a re-registration by removing the Node together with every Service configured on it. On those versions the Services you add to a restarted container are lost regardless of the volume you give it.

Registration gives PMM Server permission to collect metrics from your infrastructure and display them in monitoring dashboards. PMM supports two authentication methods: service account tokens (recommended) and username/password credentials. 

To deploy and register PMM Client using Docker:
{.power-number}

1. Pull the PMM Client Docker image:

    ```sh
    docker pull percona/pmm-client:3
    ```

2. Start the PMM Client container and register it with PMM Server using the [pmm-agent](../../use/commands/pmm-agent.md) Setup mode. Replace `X.X.X.X` with the external IP address of your PMM Server:

    !!! hint alert-success "Important"
        Do not use the `--detach` option with this command. The pmm-agent outputs logs directly to the console, and detaching would prevent you from seeing important setup information and potential errors.
   
    === "Using Service accounts (Recommended)"
   
        [Service accounts](../../api/authentication.md) provide secure, token-based authentication for registering nodes with PMM Server. Unlike standard user credentials, service account tokens can be easily rotated, revoked, or scoped to specific permissions without affecting user access to PMM.
    
        To register with service accounts, create a service account then generate an authentication token that you can use to register the PMM Client:
        {.power-number}
    
        1. Log into PMM web interface.
        2. Navigate to **Users and access > Service accounts**.
        3. Click **Add service account**.
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
            -v pmm-client-config:/usr/local/percona/pmm/config \
            percona/pmm-client:3
            ```
    
            **Parameters explained:**
    
            - `PMM_AGENT_SETUP_NODE_NAME` - Name of the Node in PMM. Keep it fixed: it defaults to the hostname, which a re-created container does not keep, and `pmm-agent setup` stops when PMM Server has the pmm-agent registered under another name
            - `pmm-client-config` - Named volume holding `pmm-agent.yaml` with the identity PMM Server issued to the pmm-agent. Without it a re-created container has nothing to keep, and registering the same Node name again fails
            - `PMM_AGENT_SETUP_NODE_TYPE` - (Optional) Node type: generic, container, etc.
            - `PMM_AGENT_SERVER_ADDRESS` - Your PMM Server’s IP address or hostname
            - `service_token` - Use this exact string as the username (not a placeholder!)
            - `YOUR_GLSA_TOKEN` - The token you copied (starts with `glsa_`)
            - `PMM_AGENT_SERVER_INSECURE_TLS` - Skip certificate validation (remove for production with valid certificates)

            You can find a complete list of compatible environment variables [here](../../use/commands/pmm-agent.md).
    
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
        -v pmm-client-config:/usr/local/percona/pmm/config \
        percona/pmm-client:3
        ```
    
        **Parameters explained:**
   
        - `PMM_AGENT_SETUP_NODE_NAME` - Name of the Node in PMM. Keep it fixed: it defaults to the hostname, which a re-created container does not keep, and `pmm-agent setup` stops when PMM Server has the pmm-agent registered under another name
        - `pmm-client-config` - Named volume holding `pmm-agent.yaml` with the identity PMM Server issued to the pmm-agent. Without it a re-created container has nothing to keep, and registering the same Node name again fails
        - `PMM_AGENT_SETUP_NODE_TYPE` - (Optional) Node type: generic, container, etc.
        - `PMM_AGENT_SERVER_ADDRESS` - Your PMM Server’s IP address or hostname
        - `admin`/`admin` - Default PMM Server username and password (change this immediately after first login)
        
        You can find a complete list of compatible environment variables [here](../../use/commands/pmm-agent.md).

        To migrate to [service accounts](../../api/authentication.md):
        {.power-number}
    
        1. Create service accounts while still using standard authentication.
        2. Test service account tokens on non-critical nodes.
        3. Gradually migrate all nodes to token authentication.
        4. Change the admin password from default.
        5. Consider restricting or disabling direct admin account usage for node registration.

!!! hint alert-success "Important"
    If you get `Failed to register pmm-agent on PMM Server: connection refused`, this typically means that the IP address is incorrect or the PMM Server is unreachable.

!!! caution alert alert-warning "Recovering a Node after losing the volume"
    If the volume is deleted while the Node still exists in PMM, the Client cannot register that Node name again and the container exits. Add `-e PMM_AGENT_SETUP_FORCE=1` for one start to take the name over. PMM Server then removes the old Node together with every Service on it, so drop the variable again afterwards.
    
## Verify the connection

Run the following command to check that PMM Client is properly connected and registered:

```bash
docker exec -t pmm-client pmm-admin status
```

If the connection is successful, you should also see an increased number of monitored nodes in the PMM user interface.

### View your monitored node
To confirm your node is being monitored:
{.power-number}

  1. Go to the main menu and select **Operating System > Overview**.

  2. In the **Node Names** drop-down menu, select the node you recently registered.

  3. Modify the time range to view the relevant data for your selected node.

## Add monitoring services

After installing PMM Client, you add database services to monitor with the [`pmm-admin`](../../use/commands/pmm-admin/pmm-admin.md) command. Run it in the container once; the Node keeps the Services you add across restarts:

```bash
docker exec -t pmm-client pmm-admin add mysql --username=pmm --password=pass --query-source=perfschema --host=mysql-host
```

If you would rather declare the Services together with the container, use the `PMM_AGENT_PRERUN_SCRIPT` argument to pass a script with the `pmm-admin add DATABASE [FLAGS] [NAME] [ADDRESS]` commands. The pmm-agent runs the script on every start, after setup. `pmm-admin add` reports a Service which exists already as an error, and a failing script stops the container, so make the script tolerate the Services it added on an earlier start, for example by appending `|| true` to each command. For example:

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
 -v pmm-client-config:/usr/local/percona/pmm/config \
 -e PMM_AGENT_PRERUN_SCRIPT=/opt/percona/pmm-prerun.sh \
 -v ./pmm-prerun.sh:/opt/percona/pmm-prerun.sh \
 percona/pmm-client:3
```

## Tips for Docker configuration

- Ensure your host's firewall and routing rules are configured to allow Docker communications. This is crucial for Docker containers to communicate properly. For more details, see the [troubleshooting checklist](../../troubleshoot/checklist.md).
- To view available pmm-agent command-line options, run: `docker run --rm percona/pmm-client:3 --help`
