# Configure PMM Client with pmm-admin

Use these `pmm-admin` commands from the command line to configure PMM Client, register nodes with PMM Server, remove services from monitoring, and add event annotations.

`config` and `register` are CLI-only operations. To perform some of these tasks from the UI, go to **PMM Inventory** to add or remove services, or see [Annotate dashboards](../../../use/dashboards-panels/annotate/annotate.md) to add event markers. For programmatic access, see the [PMM API](../../../api/index.md).

## Commands

Use these commands to set up and maintain your PMM Client connection, control which services are monitored, and mark events on your dashboards:

- [`pmm-admin config`](#pmm-admin-config):   Set PMM Server connection details for the local pmm-agent

- [`pmm-admin register`](#pmm-admin-register):   Register this node with PMM Server

- [`pmm-admin remove`](#pmm-admin-remove):   Stop monitoring a service and remove it from PMM

- [`pmm-admin annotate`](#pmm-admin-annotate):   Add event markers to dashboards (deployments, maintenance, incidents)

## pmm-admin config

Set the PMM Server URL and credentials that pmm-agent uses to communicate with the server.

Run this after installing PMM Client or when changing server connection details.

Running it again for a registered pmm-agent keeps its registration, along with the Node and every Service on it:

- The Node is registered again only when the PMM Server address changed, when PMM Server no longer knows the pmm-agent on that Node, or with `--force`.
- If PMM Server has the pmm-agent on a Node with another name, the command stops. Re-run it with that name as the Node name argument to keep that Node together with its Services, or use `--force` to register the name you gave as a new Node instead, which leaves the other Node and its Services on PMM Server with nothing monitoring them.
- Settings which describe the Node, such as `--custom-labels` or the Node type, only take effect when the Node is registered. The command lists the ones it did not apply, and reports a registered Node address or type which differs from the one you gave.
- The credentials in `--server-url` only serve to register the Node. A registered pmm-agent keeps its service token, and the command says so when the credentials you gave were not the ones used.
- The configuration file is left unchanged, so settings it holds which have no `pmm-admin config` flag, such as the ports range or the path to `/proc/mounts`, are kept.
- If PMM Server cannot be reached, or answers something pmm-agent cannot interpret, the registration is kept, a warning is written to standard error and the command still succeeds, so that a Client can be configured while PMM Server is unavailable.
- If the configuration file is encrypted, set `PMM_AGENT_CONFIG_FILE_KEY_FILE` in the environment. Without the key the command cannot tell whether the pmm-agent is registered, and stops instead of guessing. See [Encrypt the PMM Client configuration file](../../../admin/security/client_config_encryption.md).

### Syntax

```bash
pmm-admin config [<node-address> [<node-type> [<node-name>]]] [FLAGS]
```

### Flags

- `--server-url=URL`:   PMM Server URL in `https://username:password@pmm-server-host/` format

- `--server-insecure-tls`:   Skip PMM Server TLS certificate validation

- `--node-id=node-id`:   Node ID (default is auto-detected)

- `--node-model=node-model`:   Node model

- `--region=region`:   Node region

- `--az=availability-zone`:   Node availability zone

- `--metrics-mode=mode`:   Metrics flow mode for node-exporter: `auto` (default), `push`, `pull`

- `--paths-base=dir`:   Base path for PMM client binaries, tools, and collectors

- `--agent-password=password`:   Custom agent password

- `--force`:   Register the Node even if this pmm-agent is registered already, removing the Node with that name together with all dependent Services and Agents

### Examples

- Configure PMM Client to connect to PMM Server:

    ```bash
    pmm-admin config --server-url=https://admin:admin@192.168.1.100:443
    ```

- Configure with self-signed certificate:

    ```bash
    pmm-admin config \
      --server-url=https://admin:admin@192.168.1.100:443 \
      --server-insecure-tls
    ```

## pmm-admin register

Register this node with PMM Server. Use this when setting up PMM Client for the first time or re-registering after server changes.

### Syntax

```bash
pmm-admin register [<node-address> [<node-type> [<node-name>]]] [FLAGS]
```

### Flags

- `--server-url=URL`:   PMM Server URL in `https://username:password@pmm-server-host/` format

- `--machine-id=ID`:   Node machine-id (default is auto-detected)

- `--distro=NAME`:   Node OS distribution (default is auto-detected)

- `--container-id=ID`:   Container ID

- `--container-name=NAME`:   Container name

- `--node-model=MODEL`:   Node model

- `--region=REGION`:   Node region

- `--az=AZ`:   Node availability zone

- `--custom-labels=LABELS`:   Custom user-assigned labels in `key=value,key=value` format

- `--agent-password=password`:   Custom agent password

### Examples

- Register node with PMM Server:

    ```bash
    pmm-admin register \
      --server-url=https://admin:admin@192.168.1.100:443
    ```

- Register with a custom node name:

    ```bash
    pmm-admin register \
      db-server-01 \
      --server-url=https://admin:admin@192.168.1.100:443
    ```

- Register a container node:

    ```bash
    pmm-admin register \
      --server-url=https://admin:admin@192.168.1.100:443 \
      --container-name=mysql-prod
    ```

- Register with environment labels:

    ```bash
    pmm-admin register \
      --server-url=https://admin:admin@192.168.1.100:443 \
      --custom-labels="env=production,team=backend"
    ```

## pmm-admin remove

Stop monitoring a service and remove it from PMM. This removes the service and its agents from PMM Server but does not delete any collected data. 

Data remains on PMM Server for the configured [retention period](../../../reference/faq.md#retention).

### Syntax

```bash
pmm-admin remove <SERVICE_TYPE> <SERVICE_NAME> [FLAGS]
```

Where `SERVICE_TYPE` is one of: `mysql`, `postgresql`, `mongodb`, `valkey`, `proxysql`, `haproxy`, `external`, `external-serverless`.

### Flags

- `--service-id=ID`: Remove by service ID instead of name

- `--force`: Remove service with that name or ID and all dependent services and agents

### Examples

- Remove a MySQL service by name:

  ```bash
  pmm-admin remove mysql mysql-prod
  ```

- Remove a MongoDB service:

  ```bash
  pmm-admin remove mongodb mongodb-prod
  ```

- Remove a service by ID:

  ```bash
  pmm-admin remove mysql --service-id=abc123
  ```

- Force removal when service is unreachable:

  ```bash
  pmm-admin remove mysql mysql-prod --force
  ```

### Verify removal

After removing a service, verify it's gone:

```bash
pmm-admin list
```

## pmm-admin annotate

Add event annotations to PMM dashboards. Use annotations to mark deployments, maintenance windows, incidents, or other events that might affect database performance.

Annotations appear as vertical lines on Grafana dashboards, helping you correlate performance changes with events. 

For more details, see [Annotate dashboards](../../../use/dashboards-panels/annotate/annotate.md).

### Syntax

```bash
pmm-admin annotate <TEXT> [FLAGS]
```

### Flags

- `--tags=TAGS`: A quoted string of comma-separated tags (e.g., `"tag 1,tag 2"`)

- `--node`: Apply to current node

- `--node-name=NAME`: Apply to specific node

- `--service`:   Apply to all services on the current node

- `--service-name=NAME`: Apply to specific service

### Combining flags

You can combine `--node`, `--service`, `--node-name`, and `--service-name` to annotate multiple targets at once. 

If a node or service name is specified, it takes precedence over the auto-detected current node or service.

- `--node`: Current node

- `--node-name=NAME`: Named node

- `--service`: All services on the current node

- `--service-name=NAME`: Named service

- `--node --service`: Current node and all its services

`--node-name=NAME --service-name=NAME`: Named node and named service

### Examples

- Add a deployment annotation:

    ```bash
    pmm-admin annotate "Deployed v2.1.0"
    ```

- Add an annotation with tags:

    ```bash
    pmm-admin annotate "Database maintenance" --tags="maintenance,scheduled"
    ```

- Add an annotation for a specific service:

    ```bash
    pmm-admin annotate "Schema migration completed" --service-name=mysql-prod
    ```

- Add an annotation for a specific node:

    ```bash
    pmm-admin annotate "Kernel upgrade" --node-name=db-server-01
    ```

- Add an annotation for the current node only:

    ```bash
    pmm-admin annotate "Memory upgrade to 64GB" --node
    ```

- Combine tags and service:

    ```bash
    pmm-admin annotate "Deployed hotfix v2.1.1" \
      --tags="deployment,hotfix" \
      --service-name=mysql-prod
    ```

## See also

- [`pmm-admin` command overview](../pmm-admin/pmm-admin.md)
- [Add database services to monitoring](../pmm-admin/add.md)
- [Modify agent configurations to manage inventory](../pmm-admin/inventory.md)
- [Check connection status and troubleshoot](../pmm-admin/status.md)