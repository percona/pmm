# pmm-agent - PMM Client agent

## NAME

`pmm-agent` - The PMM Client daemon program.

## SYNOPSIS

`pmm-agent [command] [options]`

## DESCRIPTION

`pmm-agent`, part of the PMM Client package, runs as a daemon process on all monitored hosts.

## COMMANDS

`pmm-agent run`
: Run pmm-agent (default).

`pmm-agent setup [node-address] [node-type] [node-name]`
: Configure local pmm-agent (requires root permissions)

`pmm-agent help [command]`
: Show help (for command) and exit.

## OPTIONS AND ENVIRONMENT

Most options can be set via environment variables (shown in parentheses).

| Option                                 | Environment variable                | Description
| -------------------------------------- | ----------------------------------- | -----------------------------
| `--server-password=SERVER-PASSWORD`    | `PMM_AGENT_SERVER_PASSWORD`         | Password to connect to PMM Server.
| `--server-username=SERVER-USERNAME`    | `PMM_AGENT_SERVER_USERNAME`         | Username to connect to PMM Server.
| `--server-address=host:port`           | `PMM_AGENT_SERVER_ADDRESS`          | PMM Server address and port number.
| `--server-insecure-tls`                | `PMM_AGENT_SERVER_INSECURE_TLS`     | Skip PMM Server TLS certificate validation.
| `--az=AZ`                              | `PMM_AGENT_SETUP_AZ`                | Node availability zone.
| `--config-file=path_to/pmm-agent.yaml` | `PMM_AGENT_CONFIG_FILE`             | Configuration file path and name.
| `--container-id=CONTAINER-ID`          | `PMM_AGENT_SETUP_CONTAINER_ID`      | Container ID.
| `--container-name=CONTAINER-NAME`      | `PMM_AGENT_SETUP_CONTAINER_NAME`    | Container name.
| `--debug`                              | `PMM_AGENT_DEBUG`                   | Enable debug output.
| `--distro=distro`                      | `PMM_AGENT_SETUP_DISTRO`            | Node OS distribution (default is auto-detected).
| `--force`                              | `PMM_AGENT_SETUP_FORCE`             | Remove Node with that name and all dependent Services and Agents (if existing).
| `--id=...`                   | `PMM_AGENT_ID`                      | ID of this pmm-agent.
| `--listen-address=LISTEN-ADDRESS`      | `PMM_AGENT_LISTEN_ADDRESS`          | Agent local API address.
| `--listen-port=LISTEN-PORT`            | `PMM_AGENT_LISTEN_PORT`             | Agent local API port.
| `--machine-id=machine-id`              | `PMM_AGENT_SETUP_MACHINE_ID`        | Node machine ID (default is auto-detected).
| `--metrics-mode=auto`                  | `PMM_AGENT_SETUP_METRICS_MODE`      | Metrics flow mode for agents node-exporter. Can be `push` (agent will push metrics), `pull` (server scrapes metrics from agent) or `auto` (chosen by server).
| `--node-model=NODE-MODEL`              | `PMM_AGENT_SETUP_NODE_MODEL`        | Node model.
| `--paths-base=PATH`                    | `PMM_AGENT_PATHS_BASE`              | Base path for PMM client, where all binaries, tools and collectors are located. If not set, default is `/usr/local/percona/pmm`.
| `--paths-exporters_base=PATH`          | `PMM_AGENT_PATHS_EXPORTERS_BASE`    | Base path for exporters to use. If not set, or set to a relative path, uses value of `--paths-base` prepended to it.
| `--paths-mongodb_exporter=PATH`        | `PMM_AGENT_PATHS_MONGODB_EXPORTER`  | Path to `mongodb_exporter`.
| `--paths-mysqld_exporter=PATH`         | `PMM_AGENT_PATHS_MYSQLD_EXPORTER`   | Path to `mysqld_exporter`.
| `--paths-node_exporter=PATH`           | `PMM_AGENT_PATHS_NODE_EXPORTER`     | Path to `node_exporter`.
| `--paths-postgres_exporter=PATH`       | `PMM_AGENT_PATHS_POSTGRES_EXPORTER` | Path to `postgres_exporter`.
| `--paths-proxysql_exporter=PATH`       | `PMM_AGENT_PATHS_PROXYSQL_EXPORTER` | Path to `proxysql_exporter`.
| `--paths-pt-summary=PATH`              | `PMM_AGENT_PATHS_PT_SUMMARY`        | Path to `pt-summary`.
| `--paths-pt-mysql-summary=PATH`        | `PMM_AGENT_PATHS_PT_MYSQL_SUMMARY`  | Path to `pt-mysql-summary`.
| `--paths-pt-pg-summary=PATH`           | `PMM_AGENT_PATHS_PT_PG_SUMMARY`     | Path to `pt-pg-summary`.
| `--paths-tempdir=PATH`                 | `PMM_AGENT_PATHS_TEMPDIR`           | Temporary directory for exporters.
| `--ports-max=PORTS-MAX`                | `PMM_AGENT_PORTS_MAX`               | Highest allowed port number for listening sockets.
| `--ports-min=PORTS-MIN`                | `PMM_AGENT_PORTS_MIN`               | Lowest allowed port number for listening sockets.
| `--region=REGION`                      | `PMM_AGENT_SETUP_REGION`            | Node region.
| `--skip-registration`                  | `PMM_AGENT_SETUP_SKIP_REGISTRATION` | Skip registration on PMM Server.
| `--trace`                              | `PMM_AGENT_TRACE`                   | Enable trace output (implies `--debug`).
| `-h`, `--help`                         |                                     | Show help (synonym for `pmm-agent help`).
| `--version`                            |                                     | Show application version, PMM version, time-stamp, git commit hash and branch.
| `--expose-exporter` | | If you enable this flag, any IP address on the local network and anywhere on the internet can access node exporter endpoints. If the flag is disabled, node exporter endpoints can be accessed only locally.

## CONFIG FILE

PMM manages the configuration file, and it's not recommended to modify it manually. However, if necessary, you can make adjustments to specific properties in the config file, such as the username or password used for authorization through service accounts.

To do this, set the username to `service_token` and add your service token as the password. For more information about service account authorization, see [Service accounts authentication](../../api/authentication.md).

## USAGE AND EXAMPLES OF `paths-base` FLAG

Since 2.23.0 this flag could be used for easier setup of PMM agent. With this flag the root permissions for PMM client aren't needed anymore and it will be fully working.

**Examples:**

- **Case 1:** There are no root permissions for `/usr/local/percona/pmm` folder or there is a need to change default folder for PMM files.
Command:
````
pmm-agent setup --paths-base=/home/user/custom/pmm --config-file=pmm-agent-dev.yaml --server-insecure-tls --server-address=127.0.0.1:443 --server-username=admin --server-password=admin
````
Config output:
````
# Updated by `pmm-agent setup`.
---
id: be568008-b1b4-4bd9-98c7-392d1f4b724e
listen-address: 127.0.0.1
listen-port: 7777
server:
    address: 127.0.0.1:443
    username: admin
    password: admin
    insecure-tls: true
paths:
    paths_base: /home/user/custom/pmm
    exporters_base: /home/user/custom/pmm/exporters
    node_exporter: /home/user/custom/pmm/exporters/node_exporter
    mysqld_exporter: /home/user/custom/pmm/exporters/mysqld_exporter
    mongodb_exporter: /home/user/custom/pmm/exporters/mongodb_exporter
    postgres_exporter: /home/user/custom/pmm/exporters/postgres_exporter
    proxysql_exporter: /home/user/custom/pmm/exporters/proxysql_exporter
    rds_exporter: /home/user/custom/pmm/exporters/rds_exporter
    azure_exporter: /home/user/custom/pmm/exporters/azure_exporter
    vmagent: /home/user/custom/pmm/exporters/vmagent
    tempdir: /tmp
    pt_summary: /home/user/custom/pmm/tools/pt-summary
    pt_pg_summary: /home/user/custom/pmm/tools/pt-pg-summary
    pt_mysql_summary: /home/user/custom/pmm/tools/pt-mysql-summary
    pt_mongodb_summary: /home/user/custom/pmm/tools/pt-mongodb-summary
ports:
    min: 42000
    max: 51999
debug: false
trace: false

````
As could be seen above, base for all exporters and tools was changed only by setting `--paths-base`. With this tag the folder for PMM that doesn't require root access could be specified.

- **Case 2:** The older `--paths-exporters_base` flag could be passed along with the `--paths-base`
Command:
````
pmm-agent setup --paths-base=/home/user/custom/pmm --paths-exporters_base=/home/user/exporters --config-file=pmm-agent-dev.yaml --server-insecure-tls --server-address=127.0.0.1:443 --server-username=admin --server-password=admin
````
Config output:
````
# Updated by `pmm-agent setup`.
---
id: afce1917-8836-4857-b3e5-ad372c2ddbe5
listen-address: 127.0.0.1
listen-port: 7777
server:
    address: 127.0.0.1:443
    username: admin
    password: admin
    insecure-tls: true
paths:
    paths_base: /home/user/custom/pmm
    exporters_base: /home/user/exporters
    node_exporter: /home/user/exporters/node_exporter
    mysqld_exporter: /home/user/exporters/mysqld_exporter
    mongodb_exporter: /home/user/exporters/mongodb_exporter
    postgres_exporter: /home/user/exporters/postgres_exporter
    proxysql_exporter: /home/user/exporters/proxysql_exporter
    rds_exporter: /home/user/exporters/rds_exporter
    azure_exporter: /home/user/exporters/azure_exporter
    vmagent: /home/user/exporters/vmagent
    tempdir: /tmp
    pt_summary: /home/user/custom/pmm/tools/pt-summary
    pt_pg_summary: /home/user/custom/pmm/tools/pt-pg-summary
    pt_mysql_summary: /home/user/custom/pmm/tools/pt-mysql-summary
    pt_mongodb_summary: /home/user/custom/pmm/tools/pt-mongodb-summary
ports:
    min: 42000
    max: 51999
debug: false
trace: false
````
As could be seen above the behavior for the `--paths-base` was the same, but paths for all exporters were overwritten by the `--paths-exporter_base` flag.

**Summary:**
Flag `--paths-base` will set path for all exporters and tools, but each one could be overridden by specific flag (like `--paths-mongodb_exporter`, `--paths-pt-mysql-summary` and etc).

## LOGGING

By default, pmm-agent sends messages to stderr and to the system log (`syslogd` or `journald` on Linux).

To get a separate log file, edit the `pmm-agent` start-up script.

**`systemd`-based systems**

- Script file: `/usr/lib/systemd/system/pmm-agent.service`
- Parameter: `StandardError`
- Default value: `file:/var/log/pmm-agent.log`

Example:

```ini
StandardError=file:/var/log/pmm-agent.log
```

**`initd`-based systems**

- Script file: `/etc/init.d/pmm-agent`
- Parameter: `pmm_log`
- Default value: `/var/log/pmm-agent.log`

Example:

```ini
pmm_log="/var/log/pmm-agent.log"
```
