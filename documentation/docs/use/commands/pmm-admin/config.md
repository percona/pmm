# Configure PMM Client with pmm-admin

Use these `pmm-admin` commands from the command line to configure PMM Client, register nodes with PMM Server, remove services from monitoring, and add event annotations.

To perform these tasks in the UI instead, see [PMM Settings](link). For programmatic access, see the [PMM API](../../../api/index.md).

## Commands

Use these commands to set up and maintain your PMM Client connection, control which services are monitored, and mark events on your dashboards:

- [`pmm-admin config`](#pmm-admin-config)
:   Set PMM Server connection details for the local pmm-agent

- [`pmm-admin register`](#pmm-admin-register)
:   Register this node with PMM Server

- [`pmm-admin remove`](#pmm-admin-remove)
:   Stop monitoring a service and remove it from PMM

- [`pmm-admin annotate`](#pmm-admin-annotate)
:   Add event markers to dashboards (deployments, maintenance, incidents)

## pmm-admin config

Set the PMM Server URL and credentials that pmm-agent uses to communicate with the server.

Run this after installing PMM Client or when changing server connection details.

### Syntax

```bash
pmm-admin config [FLAGS] [node-address] [node-type] [node-name]
```

### Flags

- `--server-url=URL`
:   PMM Server URL in `https://username:password@pmm-server-host/` format

- `--server-insecure-tls`
:   Skip PMM Server TLS certificate validation

- `--node-id=node-id`
:   Node ID (default is auto-detected)

- `--node-model=node-model`
:   Node model

- `--region=region`
:   Node region

- `--az=availability-zone`
:   Node availability zone

- `--metrics-mode=mode`
:   Metrics flow mode for node-exporter: `auto` (default), `push`, `pull`

- `--paths-base=dir`
:   Base path for PMM client binaries, tools, and collectors

- `--agent-password=password`
:   Custom agent password

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
pmm-admin register [FLAGS] [node-address] [node-type] [node-name]
```

### Flags

- `--server-url=URL`
:   PMM Server URL in `https://username:password@pmm-server-host/` format

- `--machine-id=ID`
:   Node machine-id (default is auto-detected)

- `--distro=NAME`
:   Node OS distribution (default is auto-detected)

- `--container-id=ID`
:   Container ID

- `--container-name=NAME`
:   Container name

- `--node-model=MODEL`
:   Node model

- `--region=REGION`
:   Node region

- `--az=AZ`
:   Node availability zone

- `--custom-labels=LABELS`
:   Custom user-assigned labels in `key=value,key=value` format

- `--agent-password=password`
:   Custom agent password

### Examples

- Register node with PMM Server:

  ```bash
  pmm-admin register \
    --server-url=https://admin:admin@192.168.1.100:443
  ```

- Register with a custom node name:

  ```bash
  pmm-admin register \
    --server-url=https://admin:admin@192.168.1.100:443 \
    node-name db-server-01
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

- `--service-id=ID`
:   Remove by service ID instead of name

- `--force`
:   Remove service with that name or ID and all dependent services and agents

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
pmm-admin remove mysql --service-id=/service_id/abc123
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

- `--tags=TAGS`
:   A quoted string of comma-separated tags (e.g., `"tag 1,tag 2"`)

- `--node`
:   Apply to current node

- `--node-name=NAME`
:   Apply to specific node

- `--service`
:   Apply to all services on the current node

- `--service-name=NAME`
:   Apply to specific service

### Combining flags

You can combine `--node`, `--service`, `--node-name`, and `--service-name` to annotate multiple targets at once. 

If a node or service name is specified, it takes precedence over the auto-detected current node or service.

- `--node`
:   Current node

- `--node-name=NAME`
:   Named node

- `--service`
:   All services on the current node

- `--service-name=NAME`
:   Named service

- `--node --service`
:   Current node and all its services

`--node-name=NAME --service-name=NAME`
:   Named node and named service

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
- [Check connection status and troubleshoot](../pmm-admin/inventory.md)