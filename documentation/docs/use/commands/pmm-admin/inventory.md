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

When you change connection-affecting parameters (username, password, TLS settings, etc.), PMM verifies the new settings by connecting to the database before saving them. If the connection fails (for example, wrong credentials), the command returns an error and **no changes are applied**. Use `--skip-connection-check` to bypass this verification (see [Connection and authentication](#connection-and-authentication)).

`--agent-env-vars` is **not** part of this verification: the connection check runs with the environment the exporter already has, so it cannot validate names you are adding. A change to `--agent-env-vars` is always saved, and its effect only shows once the exporter restarts.

### When to use `change agent` vs `remove/add`

**Use `change agent` for:**

- Update database credentials
- Add or update custom labels
- Add, replace, or remove exporter environment variables
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

- `--agent-env-vars`
:   Comma-separated list of environment variable names to pass to the exporter, for example `KRB5_KTNAME,KRB5_CONFIG`

!!! note "How `--agent-env-vars` is applied"
    The flag **replaces** the entire list, so any name you leave out is removed. Omitting the flag
    keeps the current list unchanged, and `--agent-env-vars=""` removes all names.

    Names must match `[A-Z_][A-Z0-9_]*` — uppercase letters, digits and underscores, not starting
    with a digit. The same rule applies when adding an agent (`pmm-admin add mongodb` and
    `pmm-admin inventory add agent mongodb-exporter`) and is enforced by the server, so it also
    applies to the UI and direct API calls. Surrounding whitespace is trimmed and repeated names
    are collapsed.

    Only the names are stored: the values are read from the `pmm-agent` environment every time the
    exporter starts. A name that is not set in that environment is skipped and logged as a warning
    by `pmm-agent`, so a misspelled name is accepted but has no effect. Names that `pmm-agent`
    already sets for the exporter (such as `MONGODB_URI`) are skipped as well and cannot be
    overridden.

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

- Pass environment variables from `pmm-agent` to the exporter:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --agent-env-vars=KRB5_KTNAME,KRB5_CONFIG
    ```

- Remove all environment variables from an agent:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --agent-env-vars=""
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
