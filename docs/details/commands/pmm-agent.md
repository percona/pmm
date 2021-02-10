# pmm-agent - PMM Client agent

## NAME

`pmm-agent` - The PMM Client daemon program

## SYNOPSIS

`pmm-agent [command] [options]`

## DESCRIPTION

pmm-agent, part of the PMM Client package, runs as a daemon process on all monitored hosts.

## COMMANDS

`pmm-agent run`
: Run pmm-agent (default)

`pmm-agent setup [node-address] [node-type] [node-name]`
: Configure local pmm-agent

`pmm-agent help [command]`
: Show help (for command) and exit

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
| `--id=/agent_id/...`                   | `PMM_AGENT_ID`                      | ID of this pmm-agent.
| `--listen-address=LISTEN-ADDRESS`      | `PMM_AGENT_LISTEN_ADDRESS`          | Agent local API address.
| `--listen-port=LISTEN-PORT`            | `PMM_AGENT_LISTEN_PORT`             | Agent local API port.
| `--machine-id=machine-id`              | `PMM_AGENT_SETUP_MACHINE_ID`        | Node machine ID (default is auto-detected).
| `--metrics-mode=auto`                  | `PMM_AGENT_SETUP_METRICS_MODE`      | Metrics flow mode for agents node-exporter. Can be `push` (agent will push metrics), `pull` (server scrapes metrics from agent) or `auto` (chosen by server).
| `--node-model=NODE-MODEL`              | `PMM_AGENT_SETUP_NODE_MODEL`        | Node model.
| `--paths-exporters_base=PATH`          | `PMM_AGENT_PATHS_EXPORTERS_BASE`    | Base path for exporters to use.
| `--paths-mongodb_exporter=PATH`        | `PMM_AGENT_PATHS_MONGODB_EXPORTER`  | Path to `mongodb_exporter`.
| `--paths-mysqld_exporter=PATH`         | `PMM_AGENT_PATHS_MYSQLD_EXPORTER`   | Path to `mysqld_exporter`.
| `--paths-node_exporter=PATH`           | `PMM_AGENT_PATHS_NODE_EXPORTER`     | Path to `node_exporter`.
| `--paths-postgres_exporter=PATH`       | `PMM_AGENT_PATHS_POSTGRES_EXPORTER` | Path to `postgres_exporter`.
| `--paths-proxysql_exporter=PATH`       | `PMM_AGENT_PATHS_PROXYSQL_EXPORTER` | Path to `proxysql_exporter`.
| `--paths-pt-summary=PATH`              | `PMM_AGENT_PATHS_PT_SUMMARY`        | Path to `pt-summary`.
| `--paths-tempdir=PATH`                 | `PMM_AGENT_PATHS_TEMPDIR`           | Temporary directory for exporters.
| `--ports-max=PORTS-MAX`                | `PMM_AGENT_PORTS_MAX`               | Highest allowed port number for listening sockets.
| `--ports-min=PORTS-MIN`                | `PMM_AGENT_PORTS_MIN`               | Lowest allowed port number for listening sockets.
| `--region=REGION`                      | `PMM_AGENT_SETUP_REGION`            | Node region.
| `--skip-registration`                  | `PMM_AGENT_SETUP_SKIP_REGISTRATION` | Skip registration on PMM Server.
| `--trace`                              | `PMM_AGENT_TRACE`                   | Enable trace output (implies `--debug`).
| `-h`, `--help`                         |                                     | Show help (synonym for `pmm-agent help`).
| `--version`                            |                                     | Show application version, PMM version, time-stamp, git commit hash and branch.

## LOGGING

By default, pmm-agent sends messages to stderr and to the system log (`syslogd` or `journald` on Linux).

To get a separate log file, edit the `pmm-agent` start-up script.

**`systemd`-based systems**

- Script file: `/usr/lib/systemd/system/pmm-agent.service`
- Parameter: `StandardError`
- Default value: `file:/var/log/pmm-agent.log`

Example:

    StandardError=file:/var/log/pmm-agent.log

**`initd`-based systems**

- Script file: `/etc/init.d/pmm-agent`
- Parameter: `pmm_log`
- Default value: `/var/log/pmm-agent.log`

Example:

        pmm_log="/var/log/pmm-agent.log"

If you change the default log file name, reflect the change in the log rotation rules file `/etc/logrotate.d/pmm-agent-logrotate`.
