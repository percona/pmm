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

Supported agent types:

**MongoDB:**

- `mongodb-exporter`
- `qan-mongodb-profiler-agent`
- `qan-mongodb-mongolog-agent`
- `rta-mongodb-agent`

**Node:**

- `node-exporter`

Only the flags you specify are updated. All other settings remain unchanged. Changes take effect immediately without restarting the agent. The command fails with a clear error if the agent ID doesn't exist or the type doesn't match.

### When to use `change agent` vs `remove/add`

**Use `change agent` for:**

- Update database credentials
- Add or update custom labels
- Enable or disable a collector
- Update collection limits
- Change TLS settings
- Enable or disable an agent
- Change log level

**Use `remove` then `add` for:**

- Change the service name
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

### Flags for MongoDB agents

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
:   Supported for `mongodb-exporter` only, available from PMM 3.10.0. Use this flag to forward environment variables from `pmm-agent` to the exporter, useful for authentication methods like Kerberos that rely on variables PMM doesn't set itself. Pass the variable names as a comma-separated list, for example `KRB5_KTNAME,KRB5_CONFIG`. The names you pass replace any previously stored list, and an empty value removes all stored names. For full setup instructions, see [Pass environment variables to the exporter](#pass-environment-variables-to-the-exporter).

- `--enable`
:   Re-enable a disabled agent

- `--disable`
:   Disable the agent (stops metric collection)

- `--log-level`
:   Set agent log level (e.g., `info`, `debug`, `warn`, `error`)

### Disable collectors for node-exporter

If `node_exporter` is collecting metrics you don't need, disabling specific collectors reduces the load on the node and cuts the number of series PMM stores. You can do this at any time without removing the node from monitoring.

```bash
pmm-admin inventory change agent node-exporter <AGENT_ID> \
  --disable-collectors=diskstats,meminfo
```

To find the agent ID, run `pmm-admin inventory list agents --agent-type=node-exporter`. Each name maps to one `node_exporter` collector, such as `cpu`, `diskstats` or `processes`.

#### How the list works

Each time you pass `--disable-collectors`, the new list replaces the stored one:

- Collectors you leave out return to the PMM default.
- If you omit the flag entirely, the stored list doesn't change.
- To reset all collectors to the PMM default, pass an empty value: `--disable-collectors=`.

#### Stopping built-in collectors

Starting with PMM 3.10.0, disabling a collector also stops collectors that `node_exporter` runs by default. The same applies to collectors you disabled at registration with `pmm-admin config --disable-collectors`.

Collectors enabled by default on Linux:

`arp`, `bcache`, `bonding`, `btrfs`, `conntrack`, `cpu`, `cpufreq`, `diskstats`, `dmi`, `edac`, `entropy`, `fibrechannel`, `filefd`, `filesystem`, `hwmon`, `infiniband`, `ipvs`, `loadavg`, `mdadm`, `meminfo`, `netclass`, `netdev`, `netstat`, `nfs`, `nfsd`, `nvme`, `os`, `powersupplyclass`, `pressure`, `rapl`, `schedstat`, `selinux`, `sockstat`, `softnet`, `stat`, `tapestats`, `textfile`, `thermal_zone`, `time`, `timex`, `udp_queues`, `uname`, `vmstat`, `watchdog`, `xfs`, `zfs`

The change takes effect without interrupting other metrics from the node. This works only on Linux nodes with PMM Client 3.0.0 or later. On PMM Client 2.x, default collectors keep running regardless.

#### Textfile collector

To stop custom metrics from the [textfile collector](../../metrics/extend_metrics.md), use the resolution-specific names: `textfile.hr`, `textfile.mr`, or `textfile.lr`. Disabling `textfile` alone has no effect.

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

- Pass Kerberos environment variables to the exporter:

    ```bash
    pmm-admin inventory change agent mongodb-exporter 12345-67890 \
      --agent-env-vars=KRB5_KTNAME,KRB5_CONFIG
    ```

- Remove all environment variables from the exporter:

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

### Pass environment variables to the exporter

Some authentication methods, such as Kerberos, require environment variables that PMM does not set itself. Use `--agent-env-vars` to pass those variable names to the MongoDB exporter. PMM stores only the names and keeps the values in the `pmm-agent` environment on your PMM Client host. This flag works only with `mongodb-exporter`, as the QAN and RTA agents already have direct access to the `pmm-agent` environment.

Both pmm-admin and PMM Server must be 3.10.0 or later. An older PMM Server ignores the flag and does not save the variable names.

#### Set the variable in the pmm-agent environment

Before you can pass a variable name to the exporter, the value must already exist in the `pmm-agent` environment on your PMM Client host. How you set it depends on how PMM Client is deployed:

=== "systemd"

    Add the variable to the `pmm-agent` service, then restart the service:

    ```bash
    sudo systemctl edit pmm-agent
    ```

    ```ini
    [Service]
    Environment="KRB5_KTNAME=/etc/krb5.keytab"
    ```

    ```bash
    sudo systemctl restart pmm-agent
    ```

=== "Docker"

    Pass the variable to the PMM Client container with `-e`, then recreate the container. The container runs as the `pmm-agent` user (UID 1002), so that user must be able to read any files you mount.

    For Kerberos, pass the variables and mount the keytab and configuration files:

    ```bash
    -e KRB5_KTNAME=/etc/krb5.keytab \
    -e KRB5_CONFIG=/etc/krb5.conf \
    -v /etc/krb5.keytab:/etc/krb5.keytab:ro \
    -v /etc/krb5.conf:/etc/krb5.conf:ro
    ```

#### How the list works

Each time you pass `--agent-env-vars`, the new list replaces the stored one:

- A name you leave out is removed from the stored list.
- If you omit the flag entirely, the stored list does not change.
- To remove all names, pass an empty value: `--agent-env-vars=""`.
- Surrounding whitespace is trimmed, and duplicate names are stored once.

#### Naming rules

Names you pass with `--agent-env-vars` must follow these rules:

- Use only letters, digits, and underscores. Names cannot start with a digit. Pass the name only, not the value — use `KRB5_KTNAME`, not `KRB5_KTNAME=/etc/krb5.keytab`.
- The `PMM_AGENT_` prefix is reserved for `pmm-agent`'s own configuration and credentials, for example `PMM_AGENT_SERVER_PASSWORD`. PMM rejects any name with this prefix regardless of letter case. If you have `PMM_AGENT_` names stored from before PMM Client 3.10.0, they are no longer passed to the exporter. Run the command again with only the names you want to keep, or use `--agent-env-vars=""` to clear the list.
- Don't use `MONGODB_URI` in any letter case. PMM sets this variable for the exporter itself.
- Use at most 32 names, each at most 256 characters long.

!!! note "Upgrading from before PMM Client 3.10.0"
    Names stored in earlier versions may no longer pass the stricter validation introduced in 3.10.0. PMM Client skips any invalid name, logs a warning, and starts the exporter without it. To clean up, run the command with only the names you want to keep, or use `--agent-env-vars=""` to clear the list.

#### What happens on save

PMM restarts the exporter. If a name is not set in the `pmm-agent` environment, `pmm-agent` skips it, logs "Environment variable not found in pmm-agent environment", and starts the exporter without it.

To check which names are currently stored, run `pmm-admin inventory change agent mongodb-exporter <AGENT_ID>` with no flags. The command changes nothing and prints the agent, including the **Environment variables** line.

### Error handling

The command returns a clear error message in these cases:

- **Non-existent agent ID**: The specified agent ID does not exist in PMM inventory.
- **Mismatched agent type**: The agent ID exists but belongs to a different agent type, for example a `qan-postgresql-pgstatmonitor-agent` ID used with the `qan-postgresql-pgstatements-agent` subcommand. The agent is left unchanged, and the error names both types: `Agent with ID <AGENT_ID> has type qan_postgresql_pgstatmonitor_agent, expected qan_postgresql_pgstatements_agent.`
- **QAN for PMM Server's PostgreSQL set by an environment variable**: PMM Server was started with `PMM_ENABLE_INTERNAL_PG_QAN`, and the command tries to enable or disable the QAN agent of PMM Server's own PostgreSQL against that value, for example: `QAN for PMM's internal PostgreSQL server is set to false via an environment variable.` Other changes to that agent are accepted. To control this QAN agent from PMM, start PMM Server without the variable. See [Monitor PMM Server's internal PostgreSQL](../../qan/QAN-stored-metrics.md#monitor-pmm-servers-internal-postgresql).
- **Invalid flag value**: A flag receives a value outside its allowed range (e.g., an invalid log level).
- **Invalid environment variable names**: A name passed with `--agent-env-vars` breaks the [naming rules](#naming-rules). This covers a malformed name (such as `NAME=value`, or an empty element such as `A,,B`), a reserved name (`MONGODB_URI` or a `PMM_AGENT_` name), more than 32 names, and a name longer than 256 characters. No changes are saved. API requests with such names fail with `InvalidArgument`.
- **Connection check failure**: PMM could not validate the new connection-affecting settings (credentials, TLS) against the database. No changes are saved. If the database is intentionally unreachable (down, in maintenance, or you are setting a password PMM does not yet have), re-run the command with `--skip-connection-check`.

## See also

- [pmm-admin add](../pmm-admin/add.md)
- [Configuration commands](../pmm-admin/config.md)
- [Status and diagnostics](../pmm-admin/status.md)
