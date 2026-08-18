# Manage inventory with pmm-admin inventory

Use `pmm-admin inventory` from the command line to list registered services and agents, add or remove individual agents, and modify agent configurations without removing and re-adding services.

To manage inventory in the UI, go to **Configuration > Inventory**. For programmatic access, see the [PMM API](../../../api/index.md).

## Commands

- [`pmm-admin inventory list agents|nodes|services`](#pmm-admin-inventory-list)
:   Shows registered agents, nodes, or services

- [`pmm-admin inventory add agent rta-mongodb-agent`](#pmm-admin-inventory-add-agent-rta-mongodb-agent)
:   Starts Real-Time Analytics (RTA) on a MongoDB service.

- [`pmm-admin inventory remove agent`](#pmm-admin-inventory-remove-agent)
:   Removes an agent from PMM inventory. To stop RTA on a MongoDB service, remove its `rta-mongodb-agent` with this command.

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

## pmm-admin inventory add agent rta-mongodb-agent

Starts Real-Time Analytics (RTA) on a MongoDB service. Live operations appear in **Query Analytics > Real-time** as they execute. Make sure to register the service with PMM first using [`pmm-admin add mongodb`](add.md#add-mongodb).

### Syntax

```bash
pmm-admin inventory add agent rta-mongodb-agent <pmm-agent-id> <service-id> [<username>] [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `<pmm-agent-id>` | ID of the PMM Agent running on the monitored host. Get it from `pmm-admin status` or `pmm-admin inventory list agents`. |
| `<service-id>` | ID of the MongoDB service to monitor. Get it from `pmm-admin inventory list services`. |
| `<username>` | (Optional) MongoDB username. If omitted, RTA reuses the credentials from the existing MongoDB exporter. |

### Flags

#### Connection and authentication

- `--password`
:   MongoDB password.

- `--authentication-mechanism`
:   Authentication mechanism. Default is empty. Use `MONGODB-X509` for SSL certificate authentication.

- `--tls`
:   Enable TLS for the connection.

- `--tls-skip-verify`
:   Skip TLS certificate verification.

- `--tls-certificate-key-file`
:   Path to the TLS certificate/key PEM file.

- `--tls-certificate-key-file-password`
:   Password for the TLS certificate/key file.

- `--tls-ca-file`
:   Path to the CA certificate file.

- `--skip-connection-check`
:   Skip connection validation before saving the agent.

#### Collection

- `--collect-interval`
:   How often RTA polls MongoDB for live operations. Accepts a duration string (for example, `2s`, `5s`). Defaults to the server-defined value of 2 seconds.

#### Agent management

- `--custom-labels`
:   Custom user-assigned labels in `key=value,key=value` format.

- `--log-level`
:   Agent log level: `fatal`, `error`, `warn`, `info`, or `debug`.

### Examples

- Start RTA using existing MongoDB exporter credentials:

    ```bash
    # 1. Get the PMM Agent ID from the monitored host
    pmm-admin status

    # 2. Get the MongoDB service ID
    pmm-admin inventory list services

    # 3. Start RTA
    pmm-admin inventory add agent rta-mongodb-agent \
      <pmm-agent-id> \
      <service-id>
    ```

- Start RTA with explicit credentials:

    ```bash
    pmm-admin inventory add agent rta-mongodb-agent \
      <pmm-agent-id> \
      <service-id> \
      pmm_user \
      --password=pmm_pass
    ```

- Start RTA with a custom poll interval:

    ```bash
    pmm-admin inventory add agent rta-mongodb-agent \
      <pmm-agent-id> \
      <service-id> \
      --collect-interval=5s
    ```

- Start RTA with TLS:

    ```bash
    pmm-admin inventory add agent rta-mongodb-agent \
      <pmm-agent-id> \
      <service-id> \
      pmm_user \
      --password=pmm_pass \
      --tls \
      --tls-ca-file=/path/to/ca.pem
    ```

The command prints the agent ID. Note it down as you will need it to stop RTA later:

```
Real-Time Analytics MongoDB agent added.
Agent ID              : /agent_id/abc123...
PMM-Agent ID          : /agent_id/xyz456...
Service ID            : /service_id/def789...
Username              : pmm_user
TLS enabled           : false
Skip TLS verification : false
Disabled              : false
Custom labels         : {}
Collect interval      : 2s
Log level             : fatal
```

## pmm-admin inventory remove agent

Removes an agent from PMM inventory. To stop RTA on a MongoDB service, remove its `rta-mongodb-agent`. This stops RTA for that service but does not affect the MongoDB exporter or stored QAN metrics. 

To start RTA again, use `pmm-admin inventory add agent rta-mongodb-agent`.

### Syntax

```bash
pmm-admin inventory remove agent [<agent-id>] [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `<agent-id>` | ID of the agent to remove. Get the RTA agent ID from `pmm-admin inventory list agents` or from the output of `pmm-admin inventory add agent rta-mongodb-agent`. |

### Flags

- `--force`
:   Remove the agent and all its dependencies.

### Examples

- Stop RTA by removing the RTA agent:

    ```bash
    # 1. Find the RTA agent ID
    pmm-admin inventory list agents

    # 2. Remove it
    pmm-admin inventory remove agent /agent_id/abc123...
    ```

- Force-remove an agent with all dependencies:

    ```bash
    pmm-admin inventory remove agent /agent_id/abc123... --force
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

When you change connection-affecting parameters (username, password, TLS settings, etc.), PMM verifies the new settings by connecting to the database before saving them. If the connection fails (for example, wrong credentials), the command returns an error and **no changes are applied**. Use `--skip-connection-check` to bypass this verification (see [Connection and authentication](#connection-and-authentication)).

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

Look for the agent ID in the output:

```
Agent type                  Status      Metrics Mode      Agent ID                              Service ID
mongodb_exporter            Running     push             12345-67890                 abc123
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

- `--skip-connection-check`
:   Save the new settings without verifying the database connection first

!!! note "When to use `--skip-connection-check`"
    By default, PMM verifies connection-affecting changes against the database before saving them.
    Skip this check when the agent cannot reach the database at the moment you make the change, for example:

    - The database is temporarily **down or in a maintenance window**.
    - You are rotating a **password that PMM does not yet have** (the current stored credentials
      are already invalid, so the check would fail).
    - The target instance is otherwise **temporarily unreachable**.

    PMM saves the new settings as-is. If the values are wrong,
    metric collection stays broken until you correct them.

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
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --password=new_secret_pass
    ```

- Update the password while the database is unreachable (skip the connection check):

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --password=new_secret_pass \
      --skip-connection-check
    ```

- Add custom labels to an agent:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --custom-labels=env=production,team=backend
    ```

- Update credentials and labels together:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --password=new_secret_pass \
      --custom-labels=env=production
    ```

- Enable all MongoDB collectors:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --enable-all-collectors
    ```

- Disable a specific collector:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --disable-collectors=topmetrics
    ```

- Change collection limit:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --max-collections-limit=500
    ```

- Update stats collections:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --stats-collections=db1,db2.collection1
    ```

- Disable an agent (stops metric collection without removing it):

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --disable
    ```

- Re-enable a disabled agent:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --enable
    ```

### Error handling

The command returns a clear error message in these cases:

- **Non-existent agent ID**: The specified agent ID does not exist in PMM inventory.
- **Mismatched agent type**: The agent ID exists but belongs to a different agent type (e.g., using a `mysqld-exporter` ID with the `mongodb-exporter` subcommand).
- **Invalid flag value**: A flag receives a value outside its allowed range (e.g., an invalid log level).
- **Connection check failure**: PMM could not validate the new connection-affecting settings (credentials, TLS) against the database. No changes are saved. If the database is intentionally unreachable (down, in maintenance, or you are setting a password PMM does not yet have), re-run the command with `--skip-connection-check`.

## See also

- [pmm-admin add](../pmm-admin/add.md)
- [Configuration commands](../pmm-admin/config.md)
- [Status and diagnostics](../pmm-admin/status.md)
