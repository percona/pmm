# Manage inventory with pmm-admin inventory

Use `pmm-admin inventory` from the command line to list registered services and agents, and modify agent configurations without removing and re-adding services.

To manage inventory in the UI, go to **Configuration > Inventory**. For programmatic access, see the [PMM API](../../../api/index.md).

## Commands

- [`pmm-admin inventory list agents|nodes|services`](#pmm-admin-inventory-list)
:   Shows registered agents, nodes, or services

- [`pmm-admin inventory change agent`](#pmm-admin-inventory-change-agent)
:   Modifies agent configuration without removing the service

## pmm-admin inventory list

View agents, nodes, or services registered with PMM Server. You must specify which type to list:

```bash
pmm-admin inventory list agents
pmm-admin inventory list nodes
pmm-admin inventory list services
```

### Examples

- List all agents:

    ```bash
    pmm-admin inventory list agents
    ```

- List all nodes:

    ```bash
    pmm-admin inventory list nodes
    ```

- List all services:

    ```bash
    pmm-admin inventory list services
    ```

## pmm-admin inventory change agent

Modify agent configuration without removing and re-adding the service. Use this to update collector settings, enable or disable features, or change connection parameters.

!!! note "PMM 3.7.0+"
    This command is available starting with PMM 3.7.0.

### Syntax

```bash
pmm-admin inventory change agent <AGENT_TYPE> <AGENT_ID> [FLAGS]
```

### How `inventory change agent` works

Currently supports MongoDB agent types only:

- `mongodb-exporter`
- `qan-mongodb-profiler-agent`
- `qan-mongodb-mongolog-agent`
- `rta-mongodb-agent`

Only the flags you specify are updated — all other settings remain unchanged. Changes take effect immediately without restarting the agent. The command fails with a clear error if the agent ID doesn't exist or the type doesn't match.

### When to use `change agent` vs `remove/add`

**Use `change agent` for:**

- Update database credentials
- Add or update custom labels
- Enable/disable a collector
- Update collection limits
- Change TLS settings
- Enable or disable an agent
- Change log level

**Use `remove` then `add` for:**

- Change service name
- Switch to a different database instance

### Finding the agent ID

Get the agent ID from the inventory list:

```bash
pmm-admin inventory list agents
```

Look for the agent ID in the output (format: `/agent_id/...`):

```
Agent type                  Status      Metrics Mode      Agent ID                              Service ID
mongodb_exporter            Running     push              /agent_id/12345-67890                 /service_id/abc123
```

You can also use `pmm-admin list` to see agents alongside their services.

### Available flags for MongoDB agents

#### Connection and authentication

- `--username`
:   MongoDB username

- `--password`
:   MongoDB password

- `--tls`
:   Enable TLS

- `--tls-skip-verify`
:   Skip TLS certificate validation

- `--tls-ca-file`
:   Path to CA certificate

- `--tls-certificate-key-file`
:   Path to combined cert/key file

#### Collectors

- `--enable-all-collectors`
:   Enable all collectors

- `--disable-collectors`
:   Comma-separated list of collectors to disable

- `--max-collections-limit`
:   Max collections to monitor (-1=PMM decides, 0=unlimited)

- `--stats-collections`
:   Limit stats to specific databases/collections

#### Agent management

- `--custom-labels`
:   Custom user-assigned labels in `key=value,key=value` format

- `--enable`
:   Re-enable a disabled agent

- `--disable`
:   Disable the agent (stops metric collection)

- `--log-level`
:   Set agent log level (e.g., `info`, `debug`, `warn`, `error`)

### Examples

- Update the MongoDB password for a running agent:

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --password=new_secret_pass
    ```

- Add custom labels to an agent:

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --custom-labels=env=production,team=backend
    ```

- Update credentials and labels together:

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --password=new_secret_pass \
      --custom-labels=env=production
    ```

- Enable all MongoDB collectors:

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --enable-all-collectors
    ```

- Disable a specific collector:

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --disable-collectors=topmetrics
    ```

- Change collection limit:

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --max-collections-limit=500
    ```

- Update stats collections:

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --stats-collections=db1,db2.collection1
    ```

- Disable an agent (stops metric collection without removing it):

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --disable
    ```

- Re-enable a disabled agent:

    ```bash
    pmm-admin inventory change agent mongodb-exporter /agent_id/12345-67890 \
      --enable
    ```

### Error handling

The command returns a clear error message in these cases:

- **Non-existent agent ID**: The specified agent ID does not exist in PMM inventory.
- **Mismatched agent type**: The agent ID exists but belongs to a different agent type (e.g., using a `mysqld-exporter` ID with the `mongodb-exporter` subcommand).
- **Invalid flag value**: A flag receives a value outside its allowed range (e.g., an invalid log level).

## See also

- [pmm-admin add](../pmm-admin/add.md)
- [Configuration commands](../pmm-admin/config.md)
- [Status and diagnostics](../pmm-admin/status.md)