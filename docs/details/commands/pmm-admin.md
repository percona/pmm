# pmm-admin - Administration Tool

## NAME

`pmm-admin` - Administer PMM

## SYNOPSIS

`pmm-admin [FLAGS]`

`pmm-admin config [FLAGS] --server-url=server-url`

`pmm-admin add DATABASE [FLAGS] [NAME] [ADDRESS]`

`pmm-admin add external [FLAGS] [NAME] [ADDRESS]` (CAUTION: Technical preview feature)

`pmm-admin remove [FLAGS] service-type [service-name]`

`pmm-admin register [FLAGS] [node-address] [node-type] [node-name]`

`pmm-admin list [FLAGS] [node-address]`

`pmm-admin status [FLAGS] [node-address]`

`pmm-admin summary [FLAGS] [node-address]`

`pmm-admin annotate [--node|--service] [--tags <tags>] [node-name|service-name]`

`pmm-admin help [COMMAND]`

## DESCRIPTION

`pmm-admin` is a command-line tool for administering PMM using a set of COMMAND keywords and associated FLAGS.

PMM communicates with the PMM Server via a PMM agent process.

## FLAGS

`-h`, `--help`
:    Show help and exit.

`--help-long`
:    Show extended help and exit.

`--help-man`
:    Generate `man` page. (Use `pmm-admin --help-man | man -l -` to view.)

`--debug`
:    Enable debug logging.

`--trace`
:    Enable trace logging (implies debug).

`--json`
:    Enable JSON output.

`--version`
:    Show the application version and exit.

`--server-url=server-url`
:    PMM Server URL in `https://username:password@pmm-server-host/` format.

`--server-insecure-tls`
:    Skip PMM Server TLS certificate validation.

`--group=<group-name>`
: Group name for external services. Default: `external`

## COMMANDS

### GENERAL COMMANDS

`pmm-admin help [COMMAND]`
:    Show help for `COMMAND`.

### INFORMATION COMMANDS

`pmm-admin list --server-url=server-url [FLAGS]`
:    Show Services and Agents running on this Node, and the agent mode (push/pull).

`pmm-admin status --server-url=server-url [FLAGS]`
:    Show the following information about a local pmm-agent, and its connected server and clients:

    * Agent: Agent ID, Node ID.
    * PMM Server: URL and version.
    * PMM Client: connection status, time drift, latency, vmagent status, pmm-admin version.
    * Agents: Agent ID path and client name.

    FLAGS:

    `--wait=<period><unit>`
    : Time to wait for a successful response from pmm-agent. *period* is an integer. *unit* is one of `ms` for milliseconds, `s` for seconds, `m` for minutes, `h` for hours.

`pmm-admin summary --server-url=server-url [FLAGS]`
:    Creates an archive file in the current directory with default filename `summary_<hostname>_<year>_<month>_<date>_<hour>_<minute>_<second>.zip`. The contents are two directories, `client` and `server` containing diagnostic text files.

     FLAGS:

    `--filename="filename"`
    : The Summary Archive filename.

    `--skip-server`
    : Skip fetching `logs.zip` from PMM Server.

    `--pprof`
    : Include performance profiling data in the summary.


### CONFIGURATION COMMANDS

`pmm-admin config [FLAGS] [node-address] [node-type] [node-name]`
:   Configure a local `pmm-agent`.

    FLAGS:

    `--node-id=node-id`
    : Node ID (default is auto-detected).

    `--node-model=node-model`
    : Node model

    `--region=region`
    : Node region

    `--az=availability-zone`
    : Node availability zone

    `--force`
    : Remove Node with that name with all dependent Services and Agents if one exist

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default)
        - `push`: agent will push metrics
        - `pull`: server scrapes metrics from agent

`pmm-admin register [FLAGS] [node-address] [node-type] [node-name]`
: Register the current Node with the PMM Server.

    `--server-url=server-url`
    : PMM Server URL in `https://username:password@pmm-server-host/` format.

    `--machine-id="/machine_id/9812826a1c45454a98ba45c56cc4f5b0"`
    : Node machine-id (default is auto-detected).

    `--distro="linux"`
    : Node OS distribution (default is auto-detected).

    `--container-id=container-id`
    : Container ID.

    `--container-name=container-name`
    : Container name.

    `--node-model=node-model`
    : Node model.

    `--region=region`
    : Node region.

    `--az=availability-zone`
    : Node availability zone.

    `--custom-labels=labels`
    : Custom user-assigned labels.

    `--force`
    : Remove Node with that name with all dependent Services and Agents if one exists.

`pmm-admin remove [FLAGS] service-type [service-name]`
: Remove Service from monitoring.

    `--service-id=service-id`
    : Service ID.

### DATABASE COMMANDS

#### MongoDB

`pmm-admin add mongodb [FLAGS] [node-name] [node-address]`
:    Add MongoDB to monitoring.

    FLAGS:

    `--node-id=node-id`
    :  Node ID (default is auto-detected).

    `--pmm-agent-id=pmm-agent-id`
    :  The pmm-agent identifier which runs this instance (default is auto-detected).

    `--username=username`
    :  MongoDB username.

    `--password=password`
    :  MongoDB password.

    `--query-source=profiler`
    :  Source of queries, one of: `profiler`, `none` (default: `profiler`).

    `--environment=environment`
    :  Environment name.

    `--cluster=cluster`
    :  Cluster name.

    `--replication-set=replication-set`
    :  Replication set name.

    `--custom-labels=custom-labels`
    :  Custom user-assigned labels.

    `--skip-connection-check`
    :  Skip connection check.

    `--tls`
    :  Use TLS to connect to the database.

    `--tls-skip-verify`
    :  Skip TLS certificates validation.

    `--tls-certificate-key-file=PATHTOCERT`
    : Path to TLS certificate file.

    `--tls-certificate-key-file=IFPASSWORDTOCERTISSET`
    : Password for TLS certificate file.

    `--tls-ca-file=PATHTOCACERT`
    : Path to certificate authority file.

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default)
        - `push`: agent will push metrics
        - `pull`: server scrapes metrics from agent

#### MySQL

`pmm-admin add mysql [FLAGS] node-name node-address | [--name=service-name] --address=address[:port] | --socket`
:   Add MySQL to monitoring.

    FLAGS:

    `--address`
    : MySQL address and port (default: 127.0.0.1:3306).

    `--socket=socket`
    : Path to MySQL socket.

    `--node-id=node-id`
    : Node ID (default is auto-detected).

    `--pmm-agent-id=pmm-agent-id`
    : The pmm-agent identifier which runs this instance (default is auto-detected).

    `--username=username`
    : MySQL username.

    `--password=password`
    : MySQL password.

    `--query-source=slowlog`
    : Source of SQL queries, one of: `slowlog`, `perfschema`, `none` (default: `slowlog`).

    `--size-slow-logs=N`
    : Rotate slow log file at this size (default: server-defined; negative value disables rotation).

    `--disable-queryexamples`
    : Disable collection of query examples.

    `--disable-tablestats`
    : Disable table statistics collection.

    `--disable-tablestats-limit=disable-tablestats-limit`
    : Table statistics collection will be disabled if there are more than specified number of tables
        (default: server-defined).

    `--environment=environment`
    : Environment name.

    `--cluster=cluster`
    : Cluster name.

    `--replication-set=replication-set`
    : Replication set name.

    `--custom-labels=custom-labels`
    : Custom user-assigned labels.

    `--skip-connection-check`
    : Skip connection check.

    `--tls`
    : Use TLS to connect to the database.

    `--tls-skip-verify`
    : Skip TLS certificates validation.

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default)
        - `push`: agent will push metrics
        - `pull`: server scrapes metrics from agent

#### PostgreSQL

`pmm-admin add postgresql [FLAGS] [node-name] [node-address]`
:   Add PostgreSQL to monitoring.

    FLAGS:

    `--node-id=<node id>`
    : Node ID (default is auto-detected).

    `--pmm-agent-id=<pmm agent id>`
    : The pmm-agent identifier which runs this instance (default is auto-detected).

    `--username=<username>`
    : PostgreSQL username.

    `--password=<password>`
    : PostgreSQL password.

    `--query-source=<query source>`
    : Source of SQL queries, one of: `pgstatements`, `pgstatmonitor`, `none` (default: `pgstatements`).

    `--environment=<environment>`
    : Environment name.

    `--cluster=<cluster>`
    : Cluster name.

    `--replication-set=<replication set>`
    : Replication set name

    `--custom-labels=<custom labels>`
    : Custom user-assigned labels.

    `--skip-connection-check`
    : Skip connection check.

    `--tls`
    : Use TLS to connect to the database.

    `--tls-skip-verify`
    : Skip TLS certificates validation.

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default)
        - `push`: agent will push metrics
        - `pull`: server scrapes metrics from agent

#### ProxySQL

`pmm-admin add proxysql [FLAGS] [node-name] [node-address]`
:   Add ProxySQL to monitoring.

    FLAGS:

    `--node-id=node-id`
    : Node ID (default is auto-detected).

    `--pmm-agent-id=pmm-agent-id`
    : The pmm-agent identifier which runs this instance (default is auto-detected).

    `--username=username`
    : ProxySQL username.

    `--password=password`
    : ProxySQL password.

    `--environment=environment`
    : Environment name.

    `--cluster=cluster`
    : Cluster name.

    `--replication-set=replication-set`
    : Replication set name.

    `--custom-labels=custom-labels`
    : Custom user-assigned labels.

    `--skip-connection-check`
    : Skip connection check.

    `--tls`
    : Use TLS to connect to the database.

    `--tls-skip-verify`
    : Skip TLS certificates validation.

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default)
        - `push`: agent will push metrics
        - `pull`: server scrapes metrics from agent

## EXAMPLES

```sh
pmm-admin add mysql --query-source=slowlog --username=pmm --password=pmm sl-mysql 127.0.0.1:3306
```

```
MySQL Service added.
Service ID  : /service_id/a89191d4-7d75-44a9-b37f-a528e2c4550f
Service name: sl-mysql
```

```sh
pmm-admin add mysql --username=pmm --password=pmm --service-name=ps-mysql --host=127.0.0.1 --port=3306
```

```sh
pmm-admin status
pmm-admin status --wait=30s
```

```
Agent ID: /agent_id/c2a55ac6-a12f-4172-8850-4101237a4236
Node ID : /node_id/29b2cc24-3b90-4892-8d7e-4b44258d9309
PMM Server:
 URL : https://x.x.x.x:443/
 Version: 2.5.0
PMM Client:
 Connected : true
 Time drift: 2.152715ms
 Latency : 465.658µs
 pmm-admin version: 2.5.0
 pmm-agent version: 2.5.0
Agents:
 /agent_id/aeb42475-486c-4f48-a906-9546fc7859e8 mysql_slowlog_agent Running
```
