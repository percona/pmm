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

| Option                                           | Environment variable                          | Description                                                                     |
| ------------------------------------------------ | --------------------------------------------- | ------------------------------------------------------------------------------- |
| `--agent-password=AGENT-PASSWORD`                | `PMM_AGENT_SETUP_NODE_PASSWORD`               | Custom password for /metrics endpoint.                                         |
| `--az=AZ`                                        | `PMM_AGENT_SETUP_AZ`                          | Node availability zone.                                                        |
| `--config-file=path_to/pmm-agent.yaml`          | `PMM_AGENT_CONFIG_FILE`                       | Configuration file path and name.                                              |
| `--container-id=CONTAINER-ID`                   | `PMM_AGENT_SETUP_CONTAINER_ID`                | Container ID.                                                                   |
| `--container-name=CONTAINER-NAME`               | `PMM_AGENT_SETUP_CONTAINER_NAME`              | Container name.                                                                 |
| `--custom-labels=CUSTOM-LABELS`                 | `PMM_AGENT_SETUP_CUSTOM_LABELS`               | Custom labels for the node.                                                    |
| `--debug`                                        | `PMM_AGENT_DEBUG`                             | Enable debug output.                                                            |
| `--disable-collectors=COLLECTOR1,COLLECTOR2`    | `PMM_AGENT_SETUP_DISABLE_COLLECTORS`          | Comma-separated list of collector names to exclude from exporter.              |
| `--distro=distro`                               | `PMM_AGENT_SETUP_DISTRO`                      | Node OS distribution (default is auto-detected).                               |
| `--expose-exporter`                              | `PMM_AGENT_EXPOSE_EXPORTER`                   | Expose the address of the agent's node-exporter publicly on 0.0.0.0.          |
| `--force`                                        | `PMM_AGENT_SETUP_FORCE`                       | Remove Node with that name and all dependent Services and Agents (if existing). |
| `--id=ID`                                        | `PMM_AGENT_ID`                                | ID of this pmm-agent.                                                          |
| `--listen-address=LISTEN-ADDRESS`                | `PMM_AGENT_LISTEN_ADDRESS`                    | Agent local API address.                                                        |
| `--listen-port=LISTEN-PORT`                     | `PMM_AGENT_LISTEN_PORT`                       | Agent local API port.                                                           |
| `--log-level=LEVEL`                             | `PMM_AGENT_LOG_LEVEL`                        | Set logging level (debug, info, warn, error, fatal).                           |
| `--log-lines-count=COUNT`                       | `PMM_AGENT_LOG_LINES_COUNT`                   | Number of most recent log lines to include in logs.zip for each component.     |
| `--machine-id=machine-id`                       | `PMM_AGENT_SETUP_MACHINE_ID`                 | Node machine ID (default is auto-detected).                                    |
| `--metrics-mode=auto`                           | `PMM_AGENT_SETUP_METRICS_MODE`               | Metrics flow mode for agents node-exporter. Can be `push` (agent will push metrics), `pull` (server scrapes metrics from agent) or `auto` (chosen by server). |
| `--node-model=NODE-MODEL`                       | `PMM_AGENT_SETUP_NODE_MODEL`                 | Node model.                                                                     |
| `--paths-azure_exporter=PATH`                   | `PMM_AGENT_PATHS_AZURE_EXPORTER`             | Path to `azure_exporter`.                                                      |
| `--paths-base=PATH`                             | `PMM_AGENT_PATHS_BASE`                       | Base path for PMM client, where all binaries, tools and collectors are located. If not set, default is `/usr/local/percona/pmm`. |
| `--paths-exporters_base=PATH`                   | `PMM_AGENT_PATHS_EXPORTERS_BASE`             | Base path for exporters to use. If not set, or set to a relative path, uses value of `--paths-base` prepended to it. |
| `--paths-mongodb_exporter=PATH`                 | `PMM_AGENT_PATHS_MONGODB_EXPORTER`           | Path to `mongodb_exporter`.                                                    |
| `--paths-mysqld_exporter=PATH`                  | `PMM_AGENT_PATHS_MYSQLD_EXPORTER`            | Path to `mysqld_exporter`.                                                     |
| `--paths-node_exporter=PATH`                    | `PMM_AGENT_PATHS_NODE_EXPORTER`              | Path to `node_exporter`.                                                       |
| `--paths-nomad=PATH`                            | `PMM_AGENT_PATHS_NOMAD`                      | Path to `nomad` binary.                                                        |
| `--paths-nomad-data-dir=PATH`                   | `PMM_AGENT_PATHS_NOMAD_DATA_DIR`             | Nomad data directory.                                                           |
| `--paths-postgres_exporter=PATH`                | `PMM_AGENT_PATHS_POSTGRES_EXPORTER`          | Path to `postgres_exporter`.                                                   |
| `--paths-proxysql_exporter=PATH`                | `PMM_AGENT_PATHS_PROXYSQL_EXPORTER`          | Path to `proxysql_exporter`.                                                   |
| `--paths-pt-mongodb-summary=PATH`               | `PMM_AGENT_PATHS_PT_MONGODB_SUMMARY`         | Path to `pt-mongodb-summary`.                                                  |
| `--paths-pt-mysql-summary=PATH`                 | `PMM_AGENT_PATHS_PT_MYSQL_SUMMARY`           | Path to `pt-mysql-summary`.                                                    |
| `--paths-pt-pg-summary=PATH`                    | `PMM_AGENT_PATHS_PT_PG_SUMMARY`              | Path to `pt-pg-summary`.                                                       |
| `--paths-pt-summary=PATH`                       | `PMM_AGENT_PATHS_PT_SUMMARY`                 | Path to `pt-summary`.                                                          |
| `--paths-tempdir=PATH`                          | `PMM_AGENT_PATHS_TEMPDIR`                    | Temporary directory for exporters.                                             |
| `--paths-valkey-exporter=PATH`                  | `PMM_AGENT_PATHS_VALKEY_EXPORTER`            | Path to `valkey_exporter`.                                                     |
| `--perfschema-refresh-rate=RATE`                | `PMM_AGENT_PERFSCHEMA_REFRESH_RATE`          | How often PMM scrapes data from Performance Schema (in seconds).               |
| `--ports-max=PORTS-MAX`                         | `PMM_AGENT_PORTS_MAX`                        | Highest allowed port number for listening sockets.                             |
| `--ports-min=PORTS-MIN`                         | `PMM_AGENT_PORTS_MIN`                        | Lowest allowed port number for listening sockets.                              |
| `--region=REGION`                               | `PMM_AGENT_SETUP_REGION`                     | Node region.                                                                    |
| `--runner-capacity=CAPACITY`                    | `PMM_AGENT_RUNNER_CAPACITY`                  | Agent internal actions/jobs runner capacity.                                   |
| `--runner-max-connections-per-service=COUNT`    | `PMM_AGENT_RUNNER_MAX_CONNECTIONS_PER_SERVICE` | Agent internal action/job runner connection limit per DB instance.             |
| `--server-address=host:port`                    | `PMM_AGENT_SERVER_ADDRESS`                   | PMM Server address and port number.                                            |
| `--server-insecure-tls`                         | `PMM_AGENT_SERVER_INSECURE_TLS`              | Skip PMM Server TLS certificate validation.                                    |
| `--server-password=SERVER-PASSWORD`             | `PMM_AGENT_SERVER_PASSWORD`                  | Password to connect to PMM Server.                                             |
| `--server-username=SERVER-USERNAME`             | `PMM_AGENT_SERVER_USERNAME`                  | Username to connect to PMM Server.                                             |
| `--skip-registration`                           | `PMM_AGENT_SETUP_SKIP_REGISTRATION`          | Skip registration on PMM Server.                                               |
| `--trace`                                       | `PMM_AGENT_TRACE`                            | Enable trace output (implies `--debug`).                                       |
| `--window-connected-time=DURATION`              | `PMM_AGENT_WINDOW_CONNECTED_TIME`            | Window time for tracking the status of connection between agent and server.    |
| `-h`, `--help`                                  |                                              | Show help (synonym for `pmm-agent help`).                                      |
| `--version`                                     |                                              | Show application version, PMM version, time-stamp, git commit hash and branch. |

## Setup command arguments

The following arguments are available for the `pmm-agent setup` command:

| Argument                                         | Environment variable                          | Description                                                                     |
| ------------------------------------------------ | --------------------------------------------- | ------------------------------------------------------------------------------- |
| `node-address`                                   | `PMM_AGENT_SETUP_NODE_ADDRESS`               | Node address (required if not auto-detected).                                  |
| `node-type`                                      | `PMM_AGENT_SETUP_NODE_TYPE`                  | Node type: `generic` or `container` (default: `generic`).                      |
| `node-name`                                      | `PMM_AGENT_SETUP_NODE_NAME`                  | Node name (default: hostname).                                                 |

## Docker entrypoint variables

The following environment variables are recognized specifically by the PMM Client Docker container entrypoint:

| Environment variable                             | Description                                                                     |
| ------------------------------------------------ | ------------------------------------------------------------------------------- |
| `PMM_AGENT_SETUP`                                | If `true`, `pmm-agent setup` is called before `pmm-agent run`.                 |
| `PMM_AGENT_PRERUN_FILE`                          | If non-empty, runs the specified file with `pmm-agent run` running in the background. |
| `PMM_AGENT_PRERUN_SCRIPT`                        | If non-empty, runs the specified shell script content with `pmm-agent run` running in the background. |
| `PMM_AGENT_SIDECAR`                              | If `true`, `pmm-agent` will be restarted if it fails.                          |
| `PMM_AGENT_SIDECAR_SLEEP`                        | Time (in seconds) to wait before restarting pmm-agent if `PMM_AGENT_SIDECAR` is `true`. Default is 1 second. |

## Cross-component variables

The following environment variables affect multiple PMM components:

| Environment variable                             | Description                                                                     |
| ------------------------------------------------ | ------------------------------------------------------------------------------- |
| `PMM_DEBUG`                                      | Enable debug logging across all PMM components (overrides `PMM_AGENT_DEBUG`). |
| `PMM_TRACE`                                      | Enable trace logging across all PMM components (overrides `PMM_AGENT_TRACE`). |

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
