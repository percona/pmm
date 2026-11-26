# Configure the pmm-agent daemon

`pmm-agent` is the daemon process that runs on each monitored host. It manages exporters, collects metrics, and handles communication between your databases and PMM Server. 

You typically don't interact with pmm-agent directly, `pmm-admin` communicates with PMM Server, which then sends commands to `pmm-agent`. 

## NAME

`pmm-agent`: The PMM Client daemon program.

## Syntax

`pmm-agent [command] [options]`

## Description

`pmm-agent`, part of the PMM Client package, runs as a daemon process on all monitored hosts.

## Commands

`pmm-agent run`
: Run pmm-agent (default).

`pmm-agent setup [node-address] [node-type] [node-name]`
: Configure local pmm-agent (requires root permissions)

    Running `setup` again for a registered pmm-agent keeps its registration, along with the Node and every Service on it:

    - The Node is registered again only when the PMM Server address changed, when PMM Server no longer knows the pmm-agent on that Node, or with `--force`.
    - If PMM Server has the pmm-agent on a Node with another name, `setup` stops. Re-run it with that name as the `node-name` argument to keep that Node together with its Services, or use `--force` to register the name you gave as a new Node instead, which leaves the other Node and its Services on PMM Server with nothing monitoring them.
    - Settings which describe the Node, such as `--custom-labels`, `--disable-collectors` or the `node-type` argument, only take effect when the Node is registered. `setup` lists the ones it did not apply, and reports a registered Node address or type which differs from the one you gave.
    - The credentials given to `setup` only serve to register the Node. A registered pmm-agent keeps its service token, and `setup` says so when the credentials you gave were not the ones used.
    - Without `--config-file`, as when `pmm-admin config` runs `setup`, the configuration file is left unchanged, so settings it holds beyond the flags given are kept.
    - If PMM Server cannot be reached, or answers something pmm-agent cannot interpret, the registration is kept, a warning is written to standard error and `setup` still succeeds, so that a Client can be set up while PMM Server is unavailable.
    - If the configuration file is encrypted, pass `--config-file-key-file`. Without the key `setup` cannot tell whether the pmm-agent is registered, and stops instead of guessing.
    - `--skip-registration` needs an Agent ID already in hand, from `--id`, `PMM_AGENT_ID`, or a configuration file given with `--config-file`. It stops when nothing supplied one, rather than adopting the ID of a configuration file it only located through the running pmm-agent.

`pmm-agent help [command]`
: Show help (for command) and exit.

## Options and environment

When you run `pmm-agent` from the command line, pass options as flags. When you run PMM Client in a container or as a service, set them as environment variables instead, so you don't have to change the start command. 

Use the following table to find the flag or variable for the setting you want to change. Options with no variable, such as `--version`, work only as flags:

| Option | Environment variable | Description |
|--------|----------------------|-------------|
| `--agent-password=PASSWORD` | `PMM_AGENT_SETUP_NODE_PASSWORD` | Custom password for the `/metrics` endpoint of `node_exporter`. |
| `--az=AZ` | `PMM_AGENT_SETUP_AZ` | Node availability zone. |
| `--config-file=path_to/pmm-agent.yaml` | `PMM_AGENT_CONFIG_FILE` | Configuration file path and name. |
| `--config-file-key-file` | `PMM_AGENT_CONFIG_FILE_KEY_FILE` | Path to RSA private key for config file encryption. See [Encrypt the PMM Client configuration file](../../admin/security/client_config_encryption.md). |
| `--config-file-key-password` | `PMM_AGENT_CONFIG_FILE_KEY_PASSWORD` | Password for the RSA private key, if the key has one. |
| `--container-id=CONTAINER-ID` | `PMM_AGENT_SETUP_CONTAINER_ID` | Container ID. |
| `--container-name=CONTAINER-NAME` | `PMM_AGENT_SETUP_CONTAINER_NAME` | Container name. |
| `--custom-labels=KEY=VALUE,...` | `PMM_AGENT_SETUP_CUSTOM_LABELS` | Custom labels for the Node, in `key=value,key=value` format. |
| `--debug` | `PMM_AGENT_DEBUG` | Turn on debug output. |
| `--disable-collectors=COLLECTORS` | `PMM_AGENT_SETUP_DISABLE_COLLECTORS` | Comma-separated list of `node_exporter` collectors to turn off. |
| `--distro=distro` | `PMM_AGENT_SETUP_DISTRO` | Node OS distribution. Detected automatically by default. |
| `--expose-exporter` | `PMM_AGENT_EXPOSE_EXPORTER` | Make `node_exporter` listen on all network interfaces (`0.0.0.0`) instead of only on localhost, so that other hosts can reach its metrics endpoint. To limit who can read the metrics, set `--agent-password` and restrict access with a firewall. |
| `--force` | `PMM_AGENT_SETUP_FORCE` | Register the Node even if this pmm-agent is registered, removing any existing Node with that name and its Services and Agents. |
| `--id=...` | `PMM_AGENT_ID` | ID of this pmm-agent. |
| `--listen-address=LISTEN-ADDRESS` | `PMM_AGENT_LISTEN_ADDRESS` | Agent local API address. |
| `--listen-port=LISTEN-PORT` | `PMM_AGENT_LISTEN_PORT` | Agent local API port. |
| `--log-level=LEVEL` | `PMM_AGENT_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` or `fatal`. Default is `info`. `--debug` and `--trace` take precedence over this setting. |
| `--log-lines-count=N` | `PMM_AGENT_LOG_LINES_COUNT` | Number of most recent log lines to include for PMM Server and each exporter and agent when you download logs. Default is `1024`. |
| `--machine-id=machine-id` | `PMM_AGENT_SETUP_MACHINE_ID` | Node machine ID. Detected automatically by default. |
| `--metrics-mode=auto` | `PMM_AGENT_SETUP_METRICS_MODE` | How metrics reach PMM Server: `push` (pmm-agent sends them), `pull` (PMM Server collects them) or `auto` (PMM Server decides). |
| `--node-model=NODE-MODEL` | `PMM_AGENT_SETUP_NODE_MODEL` | Node model. |
| `--paths-azure_exporter=PATH` | `PMM_AGENT_PATHS_AZURE_EXPORTER` | Path to `azure_exporter`. |
| `--paths-base=PATH` | `PMM_AGENT_PATHS_BASE` | Base path for PMM Client, where its binaries, tools and collectors are located. Default is `/usr/local/percona/pmm`. |
| `--paths-exporters_base=PATH` | `PMM_AGENT_PATHS_EXPORTERS_BASE` | Base path for exporters. A relative path is resolved under `--paths-base`. Default is `exporters` under `--paths-base`. |
| `--paths-mongodb_exporter=PATH` | `PMM_AGENT_PATHS_MONGODB_EXPORTER` | Path to `mongodb_exporter`. |
| `--paths-mysqld_exporter=PATH` | `PMM_AGENT_PATHS_MYSQLD_EXPORTER` | Path to `mysqld_exporter`. |
| `--paths-node_exporter=PATH` | `PMM_AGENT_PATHS_NODE_EXPORTER` | Path to `node_exporter`. |
| `--paths-nomad=PATH` | `PMM_AGENT_PATHS_NOMAD` | Path to the `nomad` binary. Default is `tools/nomad` under `--paths-base`. |
| `--paths-nomad-data-dir=PATH` | `PMM_AGENT_PATHS_NOMAD_DATA_DIR` | Nomad data directory. Default is `data/nomad` under `--paths-base`. |
| `--paths-postgres_exporter=PATH` | `PMM_AGENT_PATHS_POSTGRES_EXPORTER` | Path to `postgres_exporter`. |
| `--paths-proxysql_exporter=PATH` | `PMM_AGENT_PATHS_PROXYSQL_EXPORTER` | Path to `proxysql_exporter`. |
| `--paths-pt-mongodb-summary=PATH` | `PMM_AGENT_PATHS_PT_MONGODB_SUMMARY` | Path to `pt-mongodb-summary`. |
| `--paths-pt-mysql-summary=PATH` | `PMM_AGENT_PATHS_PT_MYSQL_SUMMARY` | Path to `pt-mysql-summary`. |
| `--paths-pt-pg-summary=PATH` | `PMM_AGENT_PATHS_PT_PG_SUMMARY` | Path to `pt-pg-summary`. |
| `--paths-pt-summary=PATH` | `PMM_AGENT_PATHS_PT_SUMMARY` | Path to `pt-summary`. |
| `--paths-tempdir=PATH` | `PMM_AGENT_PATHS_TEMPDIR` | Temporary directory for exporters. |
| `--paths-valkey-exporter=PATH` | `PMM_AGENT_PATHS_VALKEY_EXPORTER` | Path to `valkey_exporter`. |
| `--perfschema-refresh-rate=SECONDS` | `PMM_AGENT_PERFSCHEMA_REFRESH_RATE` | How often PMM reads Performance Schema data for Query Analytics, in seconds. Default is `5`. |
| `--ports-max=PORTS-MAX` | `PMM_AGENT_PORTS_MAX` | Highest allowed port number for listening sockets. |
| `--ports-min=PORTS-MIN` | `PMM_AGENT_PORTS_MIN` | Lowest allowed port number for listening sockets. |
| `--proc-mounts-path=PATH` | `PMM_AGENT_SETUP_PROC_MOUNTS_PATH` | Path to the `/proc/mounts` file that `node_exporter` uses. |
| `--region=REGION` | `PMM_AGENT_SETUP_REGION` | Node region. |
| `--runner-capacity=N` | `PMM_AGENT_RUNNER_CAPACITY` | Maximum number of actions and jobs that pmm-agent runs at the same time. Default is `32`. |
| `--runner-max-connections-per-service=N` | `PMM_AGENT_RUNNER_MAX_CONNECTIONS_PER_SERVICE` | Maximum number of actions and jobs that pmm-agent runs at the same time against one database instance. Default is `2`. |
| `--server-address=host:port` | `PMM_AGENT_SERVER_ADDRESS` | PMM Server address and port number. |
| `--server-insecure-tls` | `PMM_AGENT_SERVER_INSECURE_TLS` | Skip PMM Server TLS certificate validation. |
| `--server-password=SERVER-PASSWORD` | `PMM_AGENT_SERVER_PASSWORD` | Password to connect to PMM Server. |
| `--server-username=SERVER-USERNAME` | `PMM_AGENT_SERVER_USERNAME` | Username to connect to PMM Server. |
| `--skip-registration` | `PMM_AGENT_SETUP_SKIP_REGISTRATION` | Skip registration on PMM Server. |
| `--trace` | `PMM_AGENT_TRACE` | Turn on trace output, which includes debug output. |
| `--window-connected-time=DURATION` | `PMM_AGENT_WINDOW_CONNECTED_TIME` | Time window over which pmm-agent tracks its connection status to PMM Server. Default is `1h`. |
| `--version` |  | Show application version, PMM version, time-stamp, git commit hash and branch. |
| `-h`, `--help` |  | Show help. Same as `pmm-agent help`. |

## Configure the PMM Client container

If you run PMM Client as a container (`percona/pmm-client`), you configure it with environment variables. Besides the `pmm-agent` variables in the previous table, use the following variables to register the container with PMM Server, add services when it starts, and restart `pmm-agent` if it fails:

| Variable | Default | Description |
|----------|---------|-------------|
| `PMM_AGENT_SETUP` | `false` | Set to `true` to register the container with PMM Server when it starts. |
| `PMM_AGENT_PRERUN_FILE` | Not set | Path to an executable script that the container runs every time it starts, for example to [add services](../../install-pmm/install-pmm-client/docker.md#add-monitoring-services) with `pmm-admin add`. |
| `PMM_AGENT_PRERUN_SCRIPT` | Not set | Shell commands that the container runs every time it starts. Use it instead of `PMM_AGENT_PRERUN_FILE` to pass the commands directly. You can't set both. |
| `PMM_AGENT_SIDECAR` | `false` | Set to `true` to restart `pmm-agent` automatically if it fails, for example when PMM Client runs as a sidecar container in Kubernetes. |
| `PMM_AGENT_SIDECAR_SLEEP` | `1` | Seconds to wait before restarting `pmm-agent`. Applies only when `PMM_AGENT_SIDECAR` is `true`. |

## Switch to a service account token

PMM creates and manages the `pmm-agent` configuration file for you, so you don't normally edit it. Edit it only to change how `pmm-agent` authenticates to PMM Server, for example to use a [service account token](../../api/authentication.md) instead of a username and password.

To authenticate with a service account token:
{.power-number}

1. Open the configuration file. By default, it's `/usr/local/percona/pmm/config/pmm-agent.yaml`.

2. In the `server` section, set the username to `service_token` and the password to your token:

    ```yaml
    server:
      username: service_token
      password: <your-service-account-token>
    ```

3. Restart `pmm-agent`. If PMM Client runs in a container, restart the container instead:

    ```sh
    sudo systemctl restart pmm-agent
    ```

The configuration file stores this token in plain text. To protect it, [encrypt the PMM Client configuration file](../../admin/security/client_config_encryption.md).

## Install PMM Client files in a custom location

By default, PMM Client keeps its exporters and tools in `/usr/local/percona/pmm`, which needs root permissions. To run PMM Client without root, or to keep its files somewhere else, set a different base path when you register `pmm-agent`. All exporters and tools then use that path.

To use a custom location:
{.power-number}

1. Register `pmm-agent` with `--paths-base` set to a directory that the `pmm-agent` user can write to:

    ```sh
    pmm-agent setup --paths-base=/home/user/pmm --config-file=pmm-agent.yaml \
      --server-address=<pmm-server-address>:443 \
      --server-username=service_token --server-password=<service-account-token>
    ```

2. Check the `paths` section of the configuration file. Every exporter and tool path now starts with your base path:

    ```yaml
    paths:
        paths_base: /home/user/pmm
        exporters_base: /home/user/pmm/exporters
        node_exporter: /home/user/pmm/exporters/node_exporter
        pt_summary: /home/user/pmm/tools/pt-summary
    ```

To keep the exporters in a separate directory, also pass `--paths-exporters_base`, for example `--paths-exporters_base=/home/user/exporters`. To move a single exporter or tool, use its own flag, such as `--paths-mongodb_exporter` or `--paths-pt-mysql-summary`. For all path flags, see [Options and environment](#options-and-environment).

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
