# pmm-admin - PMM Administration Tool

## NAME

`pmm-admin` - Administer PMM

## SYNOPSIS

`pmm-admin [FLAGS]`

`pmm-admin config [FLAGS] --server-url=server-url`

`pmm-admin add DATABASE [FLAGS] [NAME] [ADDRESS]`

DATABASE:= [[MongoDB](#mongodb) | [MySQL](#mysql) | [PostgreSQL](#postgresql) | [ProxySQL](#proxysql)]

`pmm-admin add --pmm-agent-listen-port=LISTEN_PORT DATABASE [FLAGS] [NAME] [ADDRESS]`

`pmm-admin add haproxy [FLAGS] [NAME]`

`pmm-admin add external [FLAGS] [NAME] [ADDRESS]`

`pmm-admin add external-serverless [FLAGS] [NAME] [ADDRESS]`

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

## COMMON FLAGS

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

`--log-level` 
:  Set the level for the logs as per your requirement such as INFO, WARNING, ERROR, and FATAL.

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

`--expose-exporter` 
: If you enable this flag, any IP address on the local network and anywhere on the internet can access exporter endpoints. If the flag is disabled/not present, exporter endpoints can be accessed only locally. The flag is disabled by default

## COMMANDS

### GENERAL COMMANDS

`pmm-admin help [COMMAND]`
:    Show help for `COMMAND`.

### INFORMATION COMMANDS

`pmm-admin list --server-url=server-url [FLAGS]`
:    Show Services and Agents running on this Node, and the agent mode (push/pull).

`pmm-admin status --server-url=server-url [FLAGS]`
:    Show the following information about a local pmm-agent, and its connected server and clients:

    - Agent: Agent ID, Node ID.
    - PMM Server: URL and version.
    - PMM Client: connection status, time drift, latency, `vmagent` status, `pmm-admin` version.
    - Agents: Agent ID path and client name.

    FLAGS:

    `--wait=<period><unit>`
    : Time to wait for a successful response from pmm-agent. *period* is an integer. *unit* is one of `ms` for milliseconds, `s` for seconds, `m` for minutes, `h` for hours.

`pmm-admin summary --server-url=server-url [FLAGS]`
:    Creates an archive file in the current directory with default file name `summary_<hostname>_<year>_<month>_<date>_<hour>_<minute>_<second>.zip`. The contents are two directories, `client` and `server` containing diagnostic text files.

     FLAGS:

    `--filename="filename"`
    : The Summary Archive filename.

    `--skip-server`
    : Skip fetching `logs.zip` from PMM Server.

    `--pprof`
    : Include performance profiling data in the summary.

### CONFIGURATION COMMANDS

#### `pmm-admin config`

`pmm-admin config [FLAGS] [node-address] [node-type] [node-name]`
:   Configure a local `pmm-agent`.

    FLAGS:

    `--node-id=node-id`
    : Node ID (default is auto-detected).

    `--node-model=node-model`
    : Node model.

    `--region=region`
    : Node region.

    `--az=availability-zone`
    : Node availability zone.

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default).
        - `push`: agent will push metrics.
        - `pull`: server scrapes metrics from agent.

    `--paths-base=dir`
    : Base path where all binaries, tools and collectors of PMM client are located

    `--agent-password=password`
    : Custom agent password.

#### `pmm-admin register`

`pmm-admin register [FLAGS] [node-address] [node-type] [node-name]`
: Register the current Node with the PMM Server.

    `--server-url=server-url`
    : PMM Server URL in `https://username:password@pmm-server-host/` format.

    `--machine-id="9812826a1c45454a98ba45c56cc4f5b0"`
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

    `--agent-password=password`
    : Custom agent password.
 
#### `pmm-admin add --pmm-agent-listen-port=LISTEN_PORT`

`pmm-admin add --pmm-agent-listen-port=LISTEN_PORT DATABASE [FLAGS] [NAME] [ADDRESS]`
: Configure the PMM agent with a listen port.

    ` --pmm-agent-listen-port=LISTEN_PORT`
    : The PMM agent listen port.

DATABASE:= [[MongoDB](#mongodb) | [MySQL](#mysql) | [PostgreSQL](#postgresql) | [ProxySQL](#proxysql)]


#### `pmm-admin remove`

`pmm-admin remove [FLAGS] service-type [service-name]`
: Remove Service from monitoring.

    `--service-id=service-id`
    : Service ID.

    `--force`
    : Remove service with that name or ID and all dependent services and agents.

When you remove a service, collected data remains on PMM Server for the specified [retention period](../../reference/faq.md#how-to-control-data-retention--retention-).
#### `pmm-admin annotate`

`pmm-admin annotate [--node|--service] <annotation> [--tags <tags>] [--node-name=<node>] [--service-name=<service>]`
: Annotate an event. ([Read more](../../use/dashboards-panels/annotate/annotate.md))

    `<annotation>`
    : The annotation string. If it contains spaces, it should be quoted.

    `--node`
    : Annotate the current node or that specified by `--node-name`.

    `--service`
    : Annotate all services running on the current node, or that specified by `--service-name`.

    `--tags`
    : A quoted string that defines one or more comma-separated tags for the annotation. Example: `"tag 1,tag 2"`.

    `--node-name`
    : The node name being annotated.

    `--service-name`
    : The service name being annotated.

    **Combining flags**

    Flags may be combined as shown in the following examples.

    `--node`
    : Current node.

    `--node-name`
    : Node with name.

    `--node --node-name=NODE_NAME`
    : Node with name.

    `--node --service-name`
    : Current node and service with name.

    `--node --node-name --service-name`
    : Node with name and service with name.

    `--node --service`
    : Current node and all services of current node.

    `-node --node-name --service --service-name`
    : Service with name and node with name.

    `--service`
    : All services of the current node.

    `--service-name`
    : Service with name.

    `--service --service-name`
    : Service with name.

    `--service --node-name`
    : All services of current node and node with name.

    `--service-name --node-name`
    : Service with name and node with name.

    `--service --service-name -node-name`
    : Service with name and node with name.

    !!! hint alert alert-success "Tip"
        If node or service name is specified, they are used instead of other parameters.

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

    `--agent-password=password`
    :  Override the default password for accessing the `/metrics` endpoint. (Username is `pmm` and default password is the agent ID.)

        !!! caution ""
            Avoid using special characters like '\', ';' and '$' in the custom password.

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

    `--tls-certificate-key-file-password=IFPASSWORDTOCERTISSET`
    : Password for TLS certificate file.

    `--tls-ca-file=PATHTOCACERT`
    : Path to certificate authority file.

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default).
        - `push`: agent will push metrics.
        - `pull`: server scrapes metrics from agent.

    `--max-query-length=NUMBER` 
    : Limit query length in QAN. Allowed values:
        - -1: No limit.
        -  0: Default value. The default value is 4096 chars.
        - >0: Query will be truncated after <NUMBER> chars.

        !!! caution ""
            Ensure you do not set the value of `max-query-length` to 1, 2, or 3. Otherwise, the PMM agent will get terminated.

##### Advanced Options

PMM starts the MongoDB exporter by default only with `diagnosticdata` and `replicasetstatus` collectors enabled.

FLAGS:

`--enable-all-collectors`
:  Enable all collectors.

`--disable-collectors`
:  Comma-separated list of collector names to exclude from exporter.

`--max-collections-limit=-1`
:  Disable collstats, dbstats, topmetrics and indexstats if there are more than <n> collections. 0: No limit. Default is -1, PMM automatically sets this value.

    !!! caution ""
        A very high limit of `max-collections-limit` could impact the CPU and Memory usage. Check `--stats-collections` to limit the scope of collections and DB's metrics to be fetched.

`--stats-collections=db1,db2.col1`
:  Collections for collstats & indexstats.


###### Enable all collectors

To enable all collectors, pass the parameter `--enable-all-collectors` in the `pmm-admin add mongodb` command.
This will enable `collstats`, `dbstats`, `indexstats`, and `topmetrics` collectors.

###### Disable some collectors

To enable only some collectors, pass the parameter `--enable-all-collectors` along with the parameter `--disable-collectors`.

For example, if you want all collectors except `topmetrics`, specify:

```
--enable-all-collectors --disable-collectors=topmetrics
```

###### Limit `dbStats`, `collStats` and `indexStats`

By default, PMM decides the limit for the number of collections to monitor the `collStats` and `indexStats` collectors.

You can also set an additional limit for the `collStats`, `indexStats`, `dbStats`, and `topmetrics` collectors with the `--max-collections-limit` parameter.

Set the value of the parameter `--max-collections-limit` to:

- 0: which indicates that `collStats` and `indexStats` can handle unlimited collections.
- n, which indicates that `collStats` and `indexStats` can handle <=n collections. If the limit is crossed - exporter stops collecting monitoring data for the `collStats` and `indexStats` collectors.
- -1 (default) doesn't need to be explicitly set. It indicates that PMM decides how many collections it would monitor, currently <=200 (subject to change).


To further limit collections to monitor, enable `collStats` and `indexStats` for some databases or collections:

- Specify the databases and collections that `collStats` and `indexStats` will use to collect data using the parameter `--stats-collections`. This parameter receives a comma-separated list of name spaces in the form `database[.collection]`.



###### Examples

To add MongoDB with all collectors (`diagnosticdata`, `replicasetstatus`, `collstats`, `dbstats`, `indexstats`, and `topmetrics`) with default limit detected by PMM (currently <=200 collections, but subject to change):

`pmm-admin add mongodb --username=admin --password=admin_pass --enable-all-collectors mongodb_srv_1 127.0.0.1:27017`

To add MongoDB with all collectors (`diagnosticdata`, `replicasetstatus`, `collstats`, `dbstats`, `indexstats`, and `topmetrics`) with `max-collections-limit` set to 1000:

`pmm-admin add mongodb --username=admin --password=admin_pass --enable-all-collectors --max-collections-limit=1000 mongodb_srv_1 127.0.0.1:27017`

To enable all the collectors with an unlimited number of collections monitored:

`pmm-admin add mongodb --username=admin --password=admin_pass --enable-all-collectors --max-collections-limit=0 mongodb_srv_1 127.0.0.1:27017`

To add MongoDB with default collectors (`diagnosticdata` and `replicasetstatus`):

`pmm-admin add mongodb --username=admin --password=admin_pass mongodb_srv_1 127.0.0.1:27017`

Disable `collstats` collector and enable all the others without limiting `max-collections-limit`:

`pmm-admin add mongodb --username=admin --password=admin_pass --enable-all-collectors --max-collections-limit=0 --disable-collectors=collstats mongodb_srv_1 127.0.0.1:27017`

If `--stats-collections=db1,db2.col1` then the collectors are run as follows:

| Database | Collector is run on            |
|----------|--------------------------------|
| `db1`    | All the collections            |
| `db2`    | **Only** for collection `col1` |

Enable all collectors and limit monitoring for `dbstats`, `indexstats`, `collstats` and `topmetrics` for all collections in `db1` and `col1` collection in `db2`, without limiting `max-collections-limit` for a number of collections in `db1`:

`pmm-admin add mongodb --username=admin --password=admin_pass --enable-all-collectors --max-collections-limit=0 --stats-collections=db1,db2.col1 mongodb_srv_1 127.0.0.1:27017`

##### Resolutions

PMM collects metrics in two [resolutions](../../configure-pmm/metrics_res.md) to decrease CPU and Memory usage: high and low resolutions.

In high resolution we collect metrics from collectors which work fast:
- `diagnosticdata`
- `replicasetstatus`
- `topmetrics`

In low resolution we collect metrics from collectors which could take some time:
- `dbstats`
- `indexstats`
- `collstats`


#### MySQL

`pmm-admin add mysql [FLAGS] node-name node-address | [--name=service-name] --address=address[:port] | --socket`
:   Add MySQL to monitoring.

    FLAGS:

    `--address`
    : MySQL address and port (default: 127.0.0.1:3306).

    `--socket=socket`
    : Path to MySQL socket. (Find the socket path with `mysql -u root -p -e "select @@socket"`.)

    `--node-id=node-id`
    : Node ID (default is auto-detected).

    `--pmm-agent-id=pmm-agent-id`
    : The pmm-agent identifier which runs this instance (default is auto-detected).

    `--username=username`
    : MySQL username.

    `--password=password`
    : MySQL password.

    `--agent-password=password`
    :  Override the default password for accessing the `/metrics` endpoint. (Username is `pmm` and default password is the agent ID.)

        !!! caution ""
            Avoid using special characters like '\', ';' and '$' in the custom password.

    `--query-source=slowlog`
    : Source of SQL queries, one of: `slowlog`, `perfschema`, `none` (default: `slowlog`). For `slowlog` query source, you need change permissions for
    specific files. Root permissions are needed for this.

    `--size-slow-logs=N`
    : Rotate slow log file at this size. If `0`, use server-defined default. Negative values disable log rotation. A unit suffix must be appended to the number and can be one of:

        - `KiB`, `MiB`, `GiB`, `TiB` for base 2 units (1024, 1048576, etc).

    `--disable-queryexamples`
    : Disable collection of query examples.

    `--disable-tablestats`
    : Disable table statistics collection.

        Excluded collectors for low-resolution time intervals:

        - `--collect.auto_increment.columns`
        - `--collect.info_schema.tables`
        - `--collect.info_schema.tablestats`
        - `--collect.perf_schema.indexiowaits`
        - `--collect.perf_schema.tableiowaits`
        - `--collect.perf_schema.file_instances`

        Excluded collectors for medium-resolution time intervals:

        - `--collect.perf_schema.tablelocks`

    `--disable-tablestats-limit=disable-tablestats-limit`
    : Table statistics collection will be disabled if there are more than specified number of tables
        (default: server-defined). 0=no limit. Negative value disables collection.

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

    `--tls-cert-file=PATHTOCERT`
    : Path to TLS client certificate file.

    `--tls-key=PATHTOCERTKEY`
    : Key for TLS client certificate file.

    `--tls-ca-file=PATHTOCACERT`
    : Path to certificate authority file.

    `--ssl-ca=PATHTOCACERT`
    : The path name of the Certificate Authority (CA) certificate file. If used, must specify the same certificate used by the server. (-ssl-capath is similar, but specifies the path name of a directory of CA certificate files.)

    `--ssl-cert=PATHTOCERTKEY`
    : The path name of the client public key certificate file.

    `--ssl-key`
    : The path name of the client private key file.

    `--ssl-skip-verify`
    : Skip SSL certificate verification.

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default).
        - `push`: agent will push metrics.
        - `pull`: server scrapes metrics from agent.

    `--max-query-length=NUMBER`
    : Limit query length in QAN. Allowed values:
        - -1: No limit.
        -  0: Default value. The default value is 2048 chars.
        - >0: Query will be truncated after <NUMBER> chars.

        !!! caution ""
            Ensure you do not set the value of `max-query-length` to 1, 2, or 3. Otherwise, the PMM agent will get terminated.

    `--comments-parsing=off/on`
    : Enable/disable parsing comments from queries into QAN filter groups:
        - off: Disabled.
        - on: Enabled.

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

    `--database=<database>`
    : PostgreSQL database (default: postgres).

    `--agent-password=password`
    :  Override the default password for accessing the `/metrics` endpoint. (Username is `pmm` and default password is the agent ID.)

        !!! caution ""
            Avoid using special characters like '\', ';' and '$' in the custom password.

    `--query-source=<query source>`
    : Source of SQL queries, one of: `pgstatements`, `pgstatmonitor`, `none` (default: `pgstatements`).

    `--disable-queryexamples`
    : Disable collection of query examples. Applicable only if `query-source` is set to `pgstatmonitor`.
    
    `--environment=<environment>`
    : Environment name.

    `--cluster=<cluster>`
    : Cluster name.

    `--replication-set=<replication set>`
    : Replication set name.

    `--custom-labels=<custom labels>`
    : Custom user-assigned labels.

    `--skip-connection-check`
    : Skip connection check.

    `--tls`
    : Use TLS to connect to the database.

    `--tls-skip-verify`
    : Skip TLS certificates validation.

    `--tls-ca-file`
    : TLS CA certificate file.

    `--tls-cert-file`
    : TLS certificate file.

    `--tls-key-file`
    : TLS certificate key file.

    `--metrics-mode=mode`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default).
        - `push`: agent will push metrics.
        - `pull`: server scrapes metrics from agent.

    `--max-query-length=NUMBER` 
    : Limit query length in QAN. Allowed values:
        - -1: No limit.
        -  0: Default value. The default value is 2048 chars.
        - >0: Query will be truncated after <NUMBER> chars.

        !!! caution ""
            Ensure you do not set the value of `max-query-length` to 1, 2, or 3. Otherwise, the PMM agent will get terminated.

    `--comments-parsing=off/on`
    : Enable/disable parsing comments from queries into QAN filter groups:
        - off: Disabled.
        - on: Enabled.

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

    `--agent-password=password`
    :  Override the default password for accessing the `/metrics` endpoint. (Username is `pmm` and default password is the agent ID.)

        !!! caution ""
            Avoid using special characters like '\', ';' and '$' in the custom password.

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
        - `auto`: chosen by server (default).
        - `push`: agent will push metrics.
        - `pull`: server scrapes metrics from agent.

    `--disable-collectors`
    : Comma-separated list of collector names to exclude from exporter.

#### HAProxy

`pmm-admin add haproxy [FLAGS] [NAME]`
:   Add HAProxy to monitoring.

    FLAGS:

    `--server-url=SERVER-URL`
    : PMM Server URL in `https://username:password@pmm-server-host/` format.

    `--server-insecure-tls`
    : Skip PMM Server TLS certificate validation.

    `--username=USERNAME`
    : HAProxy username.

    `--password=PASSWORD`
    : HAProxy password.

    `--scheme=SCHEME`
    : Scheme to generate URI to exporter metrics endpoints (http or https).

    `--metrics-path=METRICS-PATH`
    : Path under which metrics are exposed, used to generate URI (default: /metrics).

    `--listen-port=LISTEN-PORT`
    : Listen port of haproxy exposing the metrics for scraping metrics (Required).

    `--service-node-id=SERVICE-NODE-ID`
    : Node ID where service runs (default is auto-detected).

    `--environment=ENVIRONMENT`
    : Environment name like 'production' or 'qa'.

    `--cluster=CLUSTER`
    : Cluster name.

    `--replication-set=REPLICATION-SET`
    : Replication set name.

    `--custom-labels=CUSTOM-LABELS`
    : Custom user-assigned labels. Example: region=east,app=app1.

    `--metrics-mode=MODE`
    : Metrics flow mode for agents node-exporter. Allowed values:
        - `auto`: chosen by server (default).
        - `push`: agent will push metrics.
        - `pull`: server scrapes metrics from agent.

    `--skip-connection-check`
    : Skip connection check.

### OTHER COMMANDS

`pmm-admin add external [FLAGS]`
: Add External source of data (like a custom exporter running on a port) to be monitored.

    FLAGS:

    `--service-name="current-hostname"`
    : Service name (autodetected defaults to the hostname where `pmm-admin` is running).

    `--agent-node-id=AGENT-NODE-ID`
    : Node ID where agent runs (default is autodetected).

    `--username=USERNAME`
    : External username.

    `--password=PASSWORD`
    : External password.

    `--scheme=http or https`
    : Scheme to generate URI to exporter metrics endpoints.

    `--metrics-path=/metrics`
    : Path under which metrics are exposed, used to generate URI.

    `--listen-port=LISTEN-PORT`
    : Listen port of external exporter for scraping metrics. (Required.)

    `--service-node-id=SERVICE-NODE-ID`
    : Node ID where service runs (default is autodetected).

    `--environment=prod`
    : Environment name like 'production' or 'qa'.

    `--cluster=east-cluster`
    : Cluster name.

    `--replication-set=rs1`
    : Replication set name.

    `--custom-labels=CUSTOM-LABELS`
    : Custom user-assigned labels. Example: `region=east,app=app1`.

    `--metrics-mode=auto`
    : Metrics flow mode, can be `push`: agent will push metrics, `pull`: server scrape metrics from agent or `auto`: chosen by server.

    `--group="external"`
    : Group name of external service. (Default: `external`.)

`pmm-admin add external-serverless [FLAGS]`
: Add External Service on Remote node to monitoring.

    Usage example: `pmm-admin add external-serverless --url=http://1.2.3.4:9093/metrics`.

    Also, individual parameters can be set instead of `--url` like: `pmm-admin add external-serverless --scheme=http --host=1.2.3.4 --listen-port=9093 --metrics-path=/metrics --container-name=ddd --external-name=e125`.

    Note that some parameters are mandatory depending on the context. For example, if you specify `--url`, `--schema` and other related parameters are not mandatory. But if you specify `--host` you must provide all other parameters needed to build the destination URL, or you can specify `--address` instead of host and port as individual parameters.

    FLAGS:

    `--url=URL`
    : Full URL to exporter metrics endpoints.

    `--scheme=https`
    : Scheme to generate URL to exporter metrics endpoints.

    `--username=USERNAME`
    : External username.

    `--password=PASSWORD`
    : External password.

    `--address=1.2.3.4:9000`
    : External exporter address and port.

    `--host=1.2.3.4`
    : External exporters hostname or IP address.

    `--listen-port=9999`
    : Listen port of external exporter for scraping metrics.

    `--metrics-path=/metrics`
    : Path under which metrics are exposed, used to generate URL.

    `--environment=testing`
    : Environment name.

    `--cluster=CLUSTER`
    : Cluster name.

    `--replication-set=rs1`
    : Replication set name.

    `--custom-labels='app=myapp,region=s1'`
    : Custom user-assigned labels.

    `--group="external"`
    : Group name of external service. (Default: `external`.)

    `--machine-id=MACHINE-ID`
    : Node machine-id.

    `--distro=DISTRO`
    : Node OS distribution.

    `--container-id=CONTAINER-ID`
    : Container ID.

    `--container-name=CONTAINER-NAME`
    : Container name.

    `--node-model=NODE-MODEL`
    : Node model.

    `--region=REGION`
    : Node region.

    `--az=AZ`
    : Node availability zone.

## EXAMPLES

```sh
pmm-admin add mysql --query-source=slowlog --username=pmm --password=pmm sl-mysql 127.0.0.1:3306
```

```txt
MySQL Service added.
Service ID  : a89191d4-7d75-44a9-b37f-a528e2c4550f
Service name: sl-mysql
```

```sh
pmm-admin add mysql --username=pmm --password=pmm --service-name=ps-mysql --host=127.0.0.1 --port=3306
```

```sh
pmm-admin status
pmm-admin status --wait=30s
```

```txt
Agent ID: c2a55ac6-a12f-4172-8850-4101237a4236
Node ID : 29b2cc24-3b90-4892-8d7e-4b44258d9309
PMM Server:
 URL : https://x.x.x.x:443/
 Version: 2.5.0
PMM Client:
 Connected : true
 Time drift: 2.152715ms
 Latency : 465.658µs
 pmm-admin version: 2.5.0
 pmm-agent version: 2.5.0
Agents: aeb42475-486c-4f48-a906-9546fc7859e8 mysql_slowlog_agent Running
```

### Disable collectors

```sh
pmm-admin add mysql --disable-collectors='heartbeat,global_status,info_schema.innodb_cmp' --username=pmm --password=pmm --service-name=db1-mysql --host=127.0.0.1 --port=3306
```

For other collectors that you can disable with the `--disable-collectors` option, please visit the official repositories for each exporter:

- [`node_exporter`](https://github.com/percona/node_exporter)
- [`mysqld_exporter`](https://github.com/percona/mysqld_exporter)
- [`mongodb_exporter`](https://github.com/percona/mongodb_exporter)
- [`postgres_exporter`](https://github.com/percona/postgres_exporter)
- [`proxysql_exporter`](https://github.com/percona/proxysql_exporter)

[inventory]: ../dashboards/dashboard-inventory.md


[inventory]: ../dashboards/dashboard-inventory.md
