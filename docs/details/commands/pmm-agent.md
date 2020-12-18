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

## OPTIONS (FLAGS)

Most options can be set via environment variables (shown in parentheses).

`--az=AZ`
: Node availability zone (`PMM_AGENT_SETUP_AZ`)

`--config-file=path_to/pmm-agent.yaml`
: Configuration file path and name (`PMM_AGENT_CONFIG_FILE`)

`--container-id=CONTAINER-ID`
: Container ID (`PMM_AGENT_SETUP_CONTAINER_ID`)

`--container-name=CONTAINER-NAME`
: Container name (`PMM_AGENT_SETUP_CONTAINER_NAME`)

`--debug`
: Enable debug output (`PMM_AGENT_DEBUG`)

`--distro=distro`
: Node OS distribution (default is auto-detected) (`PMM_AGENT_SETUP_DISTRO`)

`--force`
: Remove Node with that name and all dependent Services and Agents (if existing) (`PMM_AGENT_SETUP_FORCE`)

`-h`, `--help`
: Show help (synonym for `pmm-agent help`)

`--id=/agent_id/...`
: ID of this pmm-agent (`PMM_AGENT_ID`)

`--listen-address=LISTEN-ADDRESS`
: Agent local API address (`PMM_AGENT_LISTEN_ADDRESS`)

`--listen-port=LISTEN-PORT`
: Agent local API port (`PMM_AGENT_LISTEN_PORT`)

`--machine-id=machine-id`
: Node machine ID (default is auto-detected) (`PMM_AGENT_SETUP_MACHINE_ID`)

`--metrics-mode=auto`
: Metrics flow mode for agents node-exporter. Can be `push` (agent will push metrics), `pull` (server scrapes metrics from agent) or `auto` (chosen by server). (`PMM_AGENT_SETUP_METRICS_MODE`)

`--node-model=NODE-MODEL`
: Node model (`PMM_AGENT_SETUP_NODE_MODEL`)

`--paths-exporters_base=PATHS-EXPORTERS_BASE`
: Base path for exporters to use (`PMM_AGENT_PATHS_EXPORTERS_BASE`)

`--paths-mongodb_exporter=PATHS-MONGODB_EXPORTER`
: Path to `mongodb_exporter` (`PMM_AGENT_PATHS_MONGODB_EXPORTER`)

`--paths-mysqld_exporter=PATHS-MYSQLD_EXPORTER`
: Path to `mysqld_exporter` (`PMM_AGENT_PATHS_MYSQLD_EXPORTER`)

`--paths-node_exporter=PATHS-NODE_EXPORTER`
: Path to `node_exporter` (`PMM_AGENT_PATHS_NODE_EXPORTER`)

`--paths-postgres_exporter=PATHS-POSTGRES_EXPORTER`
: Path to `postgres_exporter` (`PMM_AGENT_PATHS_POSTGRES_EXPORTER`)

`--paths-proxysql_exporter=PATHS-PROXYSQL_EXPORTER`
: Path to `proxysql_exporter` (`PMM_AGENT_PATHS_PROXYSQL_EXPORTER`)

`--paths-pt-summary=PATHS-PT-SUMMARY`
: Path to pt-summary (`PMM_AGENT_PATHS_PT_SUMMARY`)

`--paths-tempdir=PATHS-TEMPDIR`
: Temporary directory for exporters (`PMM_AGENT_PATHS_TEMPDIR`)

`--ports-max=PORTS-MAX`
: Highest allowed port number for listening sockets (`PMM_AGENT_PORTS_MAX`)

`--ports-min=PORTS-MIN`
: Lowest allowed port number for listening sockets (`PMM_AGENT_PORTS_MIN`)

`--region=REGION`
: Node region (`PMM_AGENT_SETUP_REGION`)

`--server-address=host:port`
: PMM Server address and port number (`PMM_AGENT_SERVER_ADDRESS`)

`--server-insecure-tls`
: Skip PMM Server TLS certificate validation (`PMM_AGENT_SERVER_INSECURE_TLS`)

`--server-password=SERVER-PASSWORD`
: Password to connect to PMM Server (`PMM_AGENT_SERVER_PASSWORD`)

`--server-username=SERVER-USERNAME`
: Username to connect to PMM Server (`PMM_AGENT_SERVER_USERNAME`)

`--skip-registration`
: Skip registration on PMM Server (`PMM_AGENT_SETUP_SKIP_REGISTRATION`)

`--trace`
: Enable trace output (implies `--debug`) (`PMM_AGENT_TRACE`)

`--version`
: Show application version, PMM version, timestamp, git commit hash and branch.

## ENVIRONMENT

See OPTIONS.

## LOGGING

By default, pmm-agent sends messages to stderr and to the system log (syslogd or journald on Linux).

To get a separate log file, edit the `pmm-agent` start-up script.

**systemd-based systems**

- Script file: `/usr/lib/systemd/system/pmm-agent.service`
- Parameter: `StandardError`
- Default value: `file:/var/log/pmm-agent.log`

Example:

    StandardError=file:/var/log/pmm-agent.log

**initd-based systems**

- Script file: `/etc/init.d/pmm-agent`
- Parameter: `pmm_log`
- Default value: `/var/log/pmm-agent.log`

Example:

        pmm_log="/var/log/pmm-agent.log"

If you change the default log file name, reflect the change in the log rotation rules file `/etc/logrotate.d/pmm-agent-logrotate`.
