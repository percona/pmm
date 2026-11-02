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

Starting with PMM 3.7.0, you can change how an agent is configured without removing and adding the service again. Use this to rotate credentials, update TLS files and labels, turn collectors on or off, or change collection limits, while the service keeps its ID and history.

### Syntax

```bash
pmm-admin inventory change agent <AGENT_TYPE> <AGENT_ID> [FLAGS]
```

To change `pmm-agent` itself, for example its PMM Server address, use [`pmm-admin config`](config.md) instead. For every other agent, replace `<AGENT_TYPE>` with one of the following agent types:

- MongoDB: `mongodb-exporter`, `qan-mongodb-profiler-agent`, `qan-mongodb-mongolog-agent`, `rta-mongodb-agent`
- MySQL: `mysqld-exporter`, `qan-mysql-perfschema-agent`, `qan-mysql-slowlog-agent`
- PostgreSQL: `postgres-exporter`, `qan-postgresql-pgstatements-agent`, `qan-postgresql-pgstatmonitor-agent`
- ProxySQL: `proxysql-exporter`
- Valkey and Redis: `valkey-exporter`
- Node: `node-exporter`
- Amazon RDS: `rds-exporter`
- Microsoft Azure: `azure-database-exporter`
- External exporters: `external-exporter`

### When you change an agent

Only the flags you pass are changed. Every other setting keeps its current value. If you change connection settings, such as credentials or TLS files, PMM first checks that the agent can connect with them, and saves nothing if it can't. 

`pmm-agent` then restarts the agent with the new settings, so expect a short gap in the data it collects.

### Decide whether to change or re-add an agent

Whether you can change an agent in place or have to remove and add its service again depends on what you want to change:

| If you want to | Use | Result |
|----------------|-----|--------|
| Update credentials or TLS settings | `change agent` | The service keeps its ID and history |
| Change labels, collectors, collection limits or the log level | `change agent` | The service keeps its ID and history |
| Turn an agent on or off | `change agent` | The service keeps its ID and history |
| Rename the service or monitor a different database instance | `pmm-admin remove`, then `pmm-admin add` | PMM creates a new service with a new ID |

### Find the agent ID

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

### Turn an agent on or off, relabel it, or set its log level

Use the following flags to turn an agent on or off, replace its labels, change its log level, or save changes without a connection check. The **Available for** column shows which agent types accept each flag:

| Flag | Available for | Description |
|------|---------------|-------------|
| `--enable` | All agent types | Enable the agent, or disable it with `--enable=false`. A disabled agent stops collecting data until you enable it again. |
| `--custom-labels` | All agent types | Custom labels in `key=value,key=value` format. The labels you pass replace all stored labels. Pass an empty value to remove them. |
| `--log-level` | All agent types except `external-exporter` | Log level of the agent: `debug`, `info`, `warn` or `error`. All types except `mysqld-exporter`, `node-exporter`, `postgres-exporter`, `valkey-exporter` also accept `fatal`. |
| `--skip-connection-check` | All agent types except `azure-database-exporter`, `node-exporter`, `rds-exporter` | Save the new settings without checking the connection to the database first. See [Save changes without a connection check](#save-changes-without-a-connection-check). |

### Change connection and collection settings

Use the following flags to change how an agent connects to the database and what it collects. These flags differ by technology, so select the tab for yours. In each tab, the **Agent types** column shows which agents accept each flag.

=== "MongoDB"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--username` | All | Username that the agent connects with. |
    | `--password` | All | Password that the agent connects with. |
    | `--agent-password` | `mongodb-exporter` | Custom password for the exporter's `/metrics` endpoint. |
    | `--tls` | All | Use TLS to connect to the database. |
    | `--tls-skip-verify` | All | Skip TLS certificate and hostname validation. |
    | `--tls-certificate-key-file` | All | Path to the PEM file that holds the TLS certificate and key. |
    | `--tls-certificate-key-file-password` | All | Password for the TLS certificate and key file. |
    | `--tls-ca-file` | All | Path to the TLS CA certificate file. |
    | `--authentication-mechanism` | All | Authentication mechanism. Use `MONGODB-X509` for certificate authentication. |
    | `--authentication-database` | `mongodb-exporter`, `qan-mongodb-profiler-agent`, `qan-mongodb-mongolog-agent` | Database to authenticate against. |
    | `--stats-collections` | `mongodb-exporter` | Comma-separated list of databases and collections to collect statistics for, for example `db1,db2.col1`. |
    | `--collections-limit` | `mongodb-exporter` | If the instance has more collections than this, the exporter stops collecting `collstats`, `dbstats`, `topmetrics` and `indexstats`. `0` means no limit, and `-1` lets PMM decide. |
    | `--enable-diagnostic-data-histograms` | `mongodb-exporter` | Collect histogram bucket metrics from `getDiagnosticData`. |
    | `--disable-collectors` | `mongodb-exporter` | Comma-separated list of collectors to turn off. |
    | `--expose-exporter` | `mongodb-exporter` | Expose the exporter on all network interfaces of the host, not only on localhost. |
    | `--push-metrics` | `mongodb-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |
    | `--connection-timeout` | `mongodb-exporter` | Timeout for connecting to the database, for example `1s` or `1.5s`. |
    | `--agent-env-vars` | `mongodb-exporter` | Comma-separated list of environment variable names that `pmm-agent` passes to the exporter, for example `KRB5_KTNAME,KRB5_CONFIG`. The names replace the stored list, and an empty value removes it. See [Pass environment variables to the exporter](#pass-environment-variables-to-the-exporter). |
    | `--max-query-length` | `qan-mongodb-profiler-agent`, `qan-mongodb-mongolog-agent` | Maximum length of queries in Query Analytics. `-1` means no limit. If you never set it, PMM Server decides. |
    | `--collect-interval` | `rta-mongodb-agent` | How often Real-Time Analytics collects running operations, for example `2s`. If you never set it, PMM Server uses 2 seconds. |

=== "MySQL"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--username` | All | Username that the agent connects with. |
    | `--password` | All | Password that the agent connects with. |
    | `--agent-password` | `mysqld-exporter` | Custom password for the exporter's `/metrics` endpoint. |
    | `--tls` | All | Use TLS to connect to the database. |
    | `--tls-skip-verify` | All | Skip TLS certificate and hostname validation. |
    | `--tls-ca-file` | All | Path to the TLS CA certificate file. |
    | `--tls-cert-file` | All | Path to the TLS certificate file. |
    | `--tls-key-file` | All | Path to the TLS key file. |
    | `--tablestats-group-table-limit` | `mysqld-exporter` | If the instance has more tables than this, the exporter stops collecting per-table statistics. |
    | `--disable-collectors` | `mysqld-exporter` | Comma-separated list of collectors to turn off. |
    | `--expose-exporter` | `mysqld-exporter` | Expose the exporter on all network interfaces of the host, not only on localhost. |
    | `--push-metrics` | `mysqld-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |
    | `--connection-timeout` | `mysqld-exporter` | Timeout for connecting to the database, for example `1s` or `1.5s`. |
    | `--comments-parsing` | `qan-mysql-perfschema-agent`, `qan-mysql-slowlog-agent` | Turn parsing of query comments `on` or `off`. |
    | `--max-query-length` | `qan-mysql-perfschema-agent`, `qan-mysql-slowlog-agent` | Maximum length of queries in Query Analytics. `-1` means no limit. If you never set it, PMM Server decides. |
    | `--disable-query-examples` | `qan-mysql-perfschema-agent`, `qan-mysql-slowlog-agent` | Stop showing query examples in Query Analytics. Use `--disable-query-examples=false` to show them again. |
    | `--max-slowlog-file-size` | `qan-mysql-slowlog-agent` | Maximum size of the slow log file in bytes. The default is 1 GiB (`1073741824`), and `0` means no limit. |

=== "PostgreSQL"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--username` | All | Username that the agent connects with. |
    | `--password` | All | Password that the agent connects with. |
    | `--agent-password` | `postgres-exporter` | Custom password for the exporter's `/metrics` endpoint. |
    | `--tls` | All | Use TLS to connect to the database. |
    | `--tls-skip-verify` | All | Skip TLS certificate and hostname validation. |
    | `--tls-ca-file` | All | Path to the TLS CA certificate file. |
    | `--tls-cert-file` | All | Path to the TLS certificate file. |
    | `--tls-key-file` | All | Path to the TLS key file. |
    | `--disable-collectors` | `postgres-exporter` | Comma-separated list of collectors to turn off. |
    | `--expose-exporter` | `postgres-exporter` | Expose the exporter on all network interfaces of the host, not only on localhost. |
    | `--push-metrics` | `postgres-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |
    | `--auto-discovery-limit` | `postgres-exporter` | If the instance has more databases than this, the exporter stops discovering databases automatically. |
    | `--max-exporter-connections` | `postgres-exporter` | Maximum number of connections that the exporter can open to PostgreSQL. |
    | `--connection-timeout` | `postgres-exporter` | Timeout for connecting to the database, for example `1s` or `1.5s`. |
    | `--comments-parsing` | `qan-postgresql-pgstatements-agent`, `qan-postgresql-pgstatmonitor-agent` | Turn parsing of query comments `on` or `off`. |
    | `--max-query-length` | `qan-postgresql-pgstatements-agent`, `qan-postgresql-pgstatmonitor-agent` | Maximum length of queries in Query Analytics. `-1` means no limit. If you never set it, PMM Server decides. |
    | `--disable-query-examples` | `qan-postgresql-pgstatmonitor-agent` | Stop showing query examples in Query Analytics. Use `--disable-query-examples=false` to show them again. |

=== "ProxySQL"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--username` | `proxysql-exporter` | Username that the agent connects with. |
    | `--password` | `proxysql-exporter` | Password that the agent connects with. |
    | `--agent-password` | `proxysql-exporter` | Custom password for the exporter's `/metrics` endpoint. |
    | `--tls` | `proxysql-exporter` | Use TLS to connect to the database. |
    | `--tls-skip-verify` | `proxysql-exporter` | Skip TLS certificate and hostname validation. |
    | `--disable-collectors` | `proxysql-exporter` | Comma-separated list of collectors to turn off. |
    | `--expose-exporter` | `proxysql-exporter` | Expose the exporter on all network interfaces of the host, not only on localhost. |
    | `--push-metrics` | `proxysql-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |
    | `--connection-timeout` | `proxysql-exporter` | Timeout for connecting to the database, for example `1s` or `1.5s`. |

=== "Valkey and Redis"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--username` | `valkey-exporter` | Username that the agent connects with. |
    | `--password` | `valkey-exporter` | Password that the agent connects with. |
    | `--agent-password` | `valkey-exporter` | Custom password for the exporter's `/metrics` endpoint. |
    | `--tls` | `valkey-exporter` | Use TLS to connect to the database. |
    | `--tls-skip-verify` | `valkey-exporter` | Skip TLS certificate and hostname validation. |
    | `--tls-ca-file` | `valkey-exporter` | Path to the TLS CA certificate file. |
    | `--tls-cert-file` | `valkey-exporter` | Path to the TLS certificate file. |
    | `--tls-key-file` | `valkey-exporter` | Path to the TLS key file. |
    | `--disable-collectors` | `valkey-exporter` | Comma-separated list of collectors to turn off. |
    | `--expose-exporter` | `valkey-exporter` | Expose the exporter on all network interfaces of the host, not only on localhost. |
    | `--push-metrics` | `valkey-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |
    | `--connection-timeout` | `valkey-exporter` | Timeout for connecting to the database, for example `1s` or `1.5s`. |

=== "Node"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--push-metrics` | `node-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |
    | `--expose-exporter` | `node-exporter` | Expose the exporter on all network interfaces of the host, not only on localhost. |
    | `--disable-collectors` | `node-exporter` | Comma-separated list of collectors to turn off. See [Disable collectors for node-exporter](#disable-collectors-for-node-exporter). |

=== "Amazon RDS"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--aws-access-key` | `rds-exporter` | AWS access key. |
    | `--aws-secret-key` | `rds-exporter` | AWS secret key. |
    | `--disable-basic-metrics` | `rds-exporter` | Stop collecting CloudWatch basic metrics. |
    | `--disable-enhanced-metrics` | `rds-exporter` | Stop collecting Enhanced Monitoring metrics. |
    | `--push-metrics` | `rds-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |

=== "Microsoft Azure"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--azure-client-id` | `azure-database-exporter` | Azure client ID. |
    | `--azure-client-secret` | `azure-database-exporter` | Azure client secret. |
    | `--azure-tenant-id` | `azure-database-exporter` | Azure tenant ID. |
    | `--azure-subscription-id` | `azure-database-exporter` | Azure subscription ID. |
    | `--azure-resource-group` | `azure-database-exporter` | Azure resource group. |
    | `--push-metrics` | `azure-database-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |

=== "External exporters"

    | Flag | Agent types | Description |
    |------|-------------|-------------|
    | `--username` | `external-exporter` | Username for the metrics endpoint of the external exporter. |
    | `--listen-port` | `external-exporter` | Port that the external exporter listens on. |
    | `--metrics-scheme` | `external-exporter` | Scheme of the metrics endpoint: `http` or `https`. |
    | `--metrics-path` | `external-exporter` | Path of the metrics endpoint, for example `/metrics`. |
    | `--push-metrics` | `external-exporter` | Push metrics to PMM Server with vmagent instead of having PMM Server pull them. Don't turn this on for agents that run on PMM Server's own pmm-agent (`pmm-server`), because their metrics stop arriving. |

### Save changes without a connection check

By default, PMM checks new connection settings against the database before it saves them. Skip this check with `--skip-connection-check` when the agent can't reach the database at the moment you make the change, for example:

- The database is down or in a maintenance window.
- You're setting a password that PMM doesn't have yet, so the stored credentials no longer work.
- The database is temporarily unreachable for another reason.

PMM then saves the new settings as they are. If a value is wrong, the agent stops collecting data until you correct it.

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

#### Update credentials and TLS settings

To rotate the password of a MongoDB exporter:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --password=<new-password>
```

To rotate the password while the database is unreachable, skip the connection check:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --password=<new-password> \
  --skip-connection-check
```

To rotate a MySQL password, change both the exporter and its QAN agent:

```bash
pmm-admin inventory change agent mysqld-exporter <exporter-agent-id> \
  --password=<new-password>
pmm-admin inventory change agent qan-mysql-perfschema-agent <qan-agent-id> \
  --password=<new-password>
```

To replace the TLS files of a PostgreSQL exporter:

```bash
pmm-admin inventory change agent postgres-exporter <agent-id> \
  --tls-ca-file=/path/to/ca.pem \
  --tls-cert-file=/path/to/client-cert.pem \
  --tls-key-file=/path/to/client-key.pem
```

#### Change labels and collection settings

To replace the custom labels of an agent:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --custom-labels=env=production,team=backend
```

To turn off a collector:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --disable-collectors=topmetrics
```

To change the collection limit:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --collections-limit=500
```

To collect statistics only for specific databases and collections:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --stats-collections=db1,db2.collection1
```

To allow longer queries in Query Analytics:

```bash
pmm-admin inventory change agent qan-postgresql-pgstatmonitor-agent <agent-id> \
  --max-query-length=4096
```

#### Turn an agent on or off

To stop an agent from collecting data without removing it:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --enable=false
```

To turn the agent back on:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --enable
```

#### Pass environment variables to the MongoDB exporter

To pass Kerberos environment variables to the exporter:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --agent-env-vars=KRB5_KTNAME,KRB5_CONFIG
```

To remove all environment variables from the exporter:

```bash
pmm-admin inventory change agent mongodb-exporter <agent-id> \
  --agent-env-vars=""
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

### Troubleshoot errors

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
