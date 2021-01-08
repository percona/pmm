# Managing PMM Client

Use the **pmm-admin** tool to manage PMM Client.

In this chapter

[TOC]

### USAGE

```
pmm-admin [OPTIONS] [COMMAND]
```

**NOTE**: The **pmm-admin** tool requires root access (you should either be logged in as a user with root privileges or be able to run commands with **sudo**).

To view all available commands and options, run **pmm-admin** without any commands or options:

```
$ sudo pmm-admin
```

### OPTIONS

The following options can be used with any command:

`-c`, `--config-file`
:    Specify the location of PMM configuration file (default `/usr/local/percona/pmm-client/pmm.yml`).

`-h`, `--help`
:     Print help for any command and exit.

`-v`, `--version`
:     Print version of PMM Client.

`--verbose`
:     Print verbose output.

### COMMANDS

**pmm-admin add**
:     Add a monitoring service.

Adding annotations
:     Add an annotation

**pmm-admin check-network**
:     Check network connection between PMM Client and PMM Server.

**pmm-admin config**
:     Configure how PMM Client communicates with PMM Server.

**pmm-admin help**
:     Print help for any command and exit.

**pmm-admin info**
:     Print information about PMM Client.

**pmm-admin list**
:     List all monitoring services added for this PMM Client.

**pmm-admin ping**
:     Check if PMM Server is alive.

**pmm-admin purge**
:     Purge metrics data on PMM Server.

**pmm-admin remove**, **pmm-admin rm**
:     Remove monitoring services.

**pmm-admin repair**
:     Remove orphaned services.

**pmm-admin restart**
:     Restart monitoring services.

**pmm-admin show-passwords**
:     Print passwords used by PMM Client (stored in the configuration file).

**pmm-admin start**
:     Start monitoring service.

**pmm-admin stop**
:     Stop monitoring service.

**pmm-admin uninstall**
:     Clean up PMM Client before uninstalling it.

## Adding monitoring services

Use the **pmm-admin add** command to add monitoring services.

### USAGE

```
$ pmm-admin add [OPTIONS] [SERVICE]
```

When you add a monitoring service **pmm-admin** automatically creates and sets up a service in the operating system. You can tweak the **systemd** configuration file and change its behavior.

For example, you may need to disable the HTTPS protocol for the Prometheus exporter associated with the given service. To accomplish this task, you need to remove all SSL related options.

Run the following commands as root or by using the **sudo** command:

1. Open the **systemd** unit file associated with the monitoring service that you need to change, such as `pmm-mysql-metrics-42002.service`.

    ```
    $ cd /etc/systemd/system
    $ cat pmm-mysql-metrics-42002.service
    ```

2. Remove the SSL related configuration options (key, cert) from the **systemd** unit file or init.d startup script. Examples of the systemd Unit File highlights the SSL related options in the **systemd** unit file.

    The following code demonstrates how you can remove the options using the **sed** command. (If you need more information about how **sed** works, see the documentation of your system).

    ```
    $ sed -e -i.backup 's/-web.ssl[^ ]\+[^->]*//g' pmm-mysql-metrics-42002.service
    ```

3. Reload **systemd**:

    ```
    $ systemctl daemon-reload
    ```

4. Restart the monitoring service by using **pmm-admin restart**:

    ```
    $ pmm-admin restart mysql:metrics
    ```

### OPTIONS

The following option can be used with the **pmm-admin add** command:

`--dev-enable`
:     Enable experimental features.

`--disable-ssl`
:     Disable (otherwise enabled) SSL for the connection between PMM Client and PMM Server. Turning off SSL encryption for the data acquired from some objects of monitoring allows to decrease the overhead for a PMM Server connected with a lot of nodes.

`--service-port`
:  Specify the service port.

You can also use global options that apply to any other command.

### SERVICES

Specify a monitoring service alias, along with any relevant additional arguments.

For more information, run **pmm-admin add** `--help`.

### Adding external monitoring services

The **pmm-admin add** command is also used to add external monitoring services. This command adds an external monitoring service assuming that the underlying Prometheus exporter is already set up and accessible. The default scrape timeout is 10 seconds, and the interval equals to 1 minute.

To add an external monitoring service use the `external:service` monitoring service followed by the port number, name of a Prometheus job. These options are required. To specify the port number the `--service-port` option.

```
$ pmm-admin add external:service --service-port=9187 postgresql

pmm-admin 1.12.0

PMM Server      | 127.0.0.1:80
Client Name     | percona
Client Address  | 172.17.0.1
Service Manager | linux-systemd

...

Job name    Scrape interval  Scrape timeout  Metrics path  Scheme  Target           Labels              Health
postgresql  10s              1m              /metrics      http    172.17.0.1:9187  instance="percona"
```

By default, the **pmm-admin add** command automatically creates the name of the host to be displayed in the Host field of the *Advanced Data Exploration* dashboard where the metrics of the newly added external monitoring service will be displayed. This name matches the name of the host where **pmm-admin** is installed. You may choose another display name when adding the `external:service` monitoring service giving it explicitly after the Prometheus exporter name.

You may also use the `external:metrics` monitoring service. When using this option, you refer to the exporter by using a URL and a port number. The following example adds an external monitoring service which monitors a PostgreSQL instance at 192.168.200.1, port 9187. After the command completes, the **pmm-admin list** command shows the newly added external exporter at the bottom of the command’s output:

Run this command as root or by using the **sudo** command

```
$ pmm-admin add external:metrics postgresql 192.168.200.1:9187

PMM Server      | 192.168.100.1
Client Name     | percona
Client Address  | 192.168.200.1
Service Manager | linux-systemd

-------------- -------- ----------- -------- ------------ --------
SERVICE TYPE   NAME     LOCAL PORT  RUNNING  DATA SOURCE  OPTIONS
-------------- -------- ----------- -------- ------------ --------
linux:metrics  percona  42000       YES                 -


Name      Scrape interval  Scrape timeout  Metrics path  Scheme  Instances
postgres  10s              1m              /metrics      http    192.168.200.1:9187
```

### Passing options to the exporter

**pmm-admin add** sends all options which follow `--` (two consecutive dashes delimited by whitespace) to the Prometheus exporter that the given monitoring services uses. Each exporter has its own set of options.

Run the following commands as root or by using the **sudo** command.

```
$ pmm-admin add mysql:metrics -- --collect.perf_schema.eventsstatements
```

```
$ pmm-admin add mysql:metrics -- --collect.perf_schema.eventswaits=false
```

The section Exporters Overview contains all option grouped by exporters.

### Passing SSL parameters to the mongodb monitoring service

SSL/TLS related parameters are passed to an SSL enabled MongoDB server as monitoring service parameters along with the **pmm-admin add** command when adding the `mongodb:metrics` monitoring service.

Run this command as root or by using the **sudo** command

```
$ pmm-admin add mongodb:metrics -- --mongodb.tls
```

### Supported SSL/TLS Parameters

| Parameter                                   | Description                               |
| ------------------------------------------- | ------------------------------------------|
| `--mongodb.tls`                             | Enable a TLS connection with mongo server |
| `--mongodb.tls-ca`  *string*                | A path to a PEM file that contains the CAs that are trusted for server connections. *If provided*: MongoDB servers connecting to should present a certificate signed by one of these CAs. *If not provided*: System default CAs are used. |
| `--mongodb.tls-cert` *string*               | A path to a PEM file that contains the certificate and, optionally, the private key in the PEM format. This should include the whole certificate chain. *If provided*: The connection will be opened via TLS to the MongoDB server. |
| `--mongodb.tls-disable-hostname-validation` | Do hostname validation for the server connection. |
| `--mongodb.tls-private-key` *string*        | A path to a PEM file that contains the private key (if not contained in the `mongodb.tls-cert` file). |

**NOTE**: PMM does not support passing SSL/TLS related parameters to `mongodb:queries`.

```
$ mongod --dbpath=DATABASEDIR --profile 2 --slowms 200 --rateLimit 100
```

### Adding general system metrics service

Use the `linux:metrics` alias to enable general system metrics monitoring.

### USAGE

```
$ pmm-admin add linux:metrics [NAME] [OPTIONS]
```

This creates the `pmm-linux-metrics-42000` service that collects local system metrics for this particular OS instance.

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following option can be used with the `linux:metrics` alias:

`--force`
: Force to add another general system metrics service with a different name for testing purposes.

You can also use global options that apply to any other command, as well as options that apply to adding services in general.

For more information, run **pmm-admin add** `linux:metrics --help`.

### Extending metrics with textfile collector

**Versionadded:** New in version 1.16.0.

While PMM provides an excellent solution for system monitoring, sometimes you may have the need for a metric that’s not present in the list of `node_exporter` metrics out of the box. There is a simple method to extend the list of available metrics without modifying the `node_exporter` code. It is based on the textfile collector.

Starting from version 1.16.0, this collector is enabled for the `linux:metrics` in PMM Client by default.

The default directory for reading text files with the metrics is `/usr/local/percona/pmm-client/textfile-collector`, and the exporter reads files from it with the `.prom` extension. By default it contains an example file  `example.prom` which has commented contents and can be used as a template.

You are responsible for running a cronjob or other regular process to generate the metric series data and write it to this directory.

#### Example - collecting docker container information

This example will show you how to collect the number of running and stopped docker containers on a host. It uses a `crontab` task, set with the following lines in the cron configuration file (e.g. in `/etc/crontab`):

```
*/1 * * * *     root   echo -n "" > /tmp/docker_all.prom; /usr/bin/docker ps -a | sed -n '1!p'| /usr/bin/wc -l | sed -ne 's/^/node_docker_containers_total /p' >> /usr/local/percona/pmm-client/docker_all.prom;
*/1 * * * *     root   echo -n "" > /tmp/docker_running.prom; /usr/bin/docker ps | sed -n '1!p'| /usr/bin/wc -l | sed -ne 's/^/node_docker_containers_running_total /p' >>/usr/local/percona/pmm-client/docker_running.prom;
```

The result of the commands is placed into the `docker_all.prom` and `docker_running.prom` files and read by exporter.

The first command executed by cron is rather simple: the destination text file is cleared by executing `echo -n ""`, then a list of running and closed containers is generated with `docker ps -a`, and finally `sed` and `wc` tools are used to count the number of containers in this list and to form the output file which looks like follows:

```
node_docker_containers_total 2
```

The second command is similar, but it counts only running containers.

### Adding MySQL query analytics service

Use the `mysql:queries` alias to enable MySQL query analytics.

### USAGE

```
pmm-admin add mysql:queries [NAME] [OPTIONS]
```

This creates the `pmm-mysql-queries-0` service that is able to collect QAN data for multiple remote MySQL server instances.

The **pmm-admin add** command is able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following options can be used with the `mysql:queries` alias:

`--create-user`
: Create a dedicated MySQL user for PMM Client (named `pmm`).

`--create-user-maxconn`
: Specify maximum connections for the dedicated MySQL user (default is 10).

`--create-user-password`
: Specify password for the dedicated MySQL user.

`--defaults-file`
: Specify path to `my.cnf`.

`--disable-queryexamples`
: Disable collection of query examples.

`--slow-log-rotation`
: Do not manage *slow log* files by using PMM. Set this option to *false* if you intend to manage *slow log* files by using a third party tool.  The default value is *true*

`--force`
:     Force to create or update the dedicated MySQL user.

`--host`
:     Specify the MySQL host name.

`--password`
:     Specify the password for MySQL user with admin privileges.

`--port`
:     Specify the MySQL instance port.

`--query-source`
:     Specify the source of data:

    * `auto`: Select automatically (default).
    * `slowlog`: Use the slow query log.
    * `perfschema`: Use Performance Schema.

`--retain-slow-logs`
:    Specify the maximum number of files of the *slow log* to keep automatically. The default value is 1 file.

`--socket`
: Specify the MySQL instance socket file.

`--user`
: Specify the name of MySQL user with admin privileges.

You can also use global options that apply to any other command, as well as options that apply to adding services in general.

### DETAILED DESCRIPTION

When adding the MySQL query analytics service, the **pmm-admin** tool will attempt to automatically detect the local MySQL instance and MySQL superuser credentials.  You can use options to provide this information, if it cannot be detected automatically.

You can also specify the `--create-user` option to create a dedicated `pmm` user on the MySQL instance that you want to monitor. This user will be given all the necessary privileges for monitoring, and is recommended over using the MySQL superuser.

For example, to set up remote monitoring of QAN data on a MySQL server located at 192.168.200.2, use a command similar to the following:

```
$ pmm-admin add mysql:queries --user root --password root --host 192.168.200.2 --create-user
```

QAN can use either the *slow query log* or *Performance Schema* as the source. By default, it chooses the *slow query log* for a local MySQL instance and *Performance Schema* otherwise. For more information about the differences, see Configuring Performance Schema.

You can explicitly set the query source when adding a QAN instance using the `--query-source` option.

For more information, run **pmm-admin add** `mysql:queries --help`.

### Adding MySQL metrics service

Use the `mysql:metrics` alias to enable MySQL metrics monitoring.

### USAGE

```
$ pmm-admin add mysql:metrics [NAME] [OPTIONS]
```

This creates the `pmm-mysql-metrics-42002` service that collects MySQL instance metrics.

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following options can be used with the `mysql:metrics` alias:

`--create-user`
: Create a dedicated MySQL user for PMM Client (named `pmm`).

`--create-user-maxconn`
: Specify maximum connections for the dedicated MySQL user (default is 10).

`--create-user-password`
: Specify password for the dedicated MySQL user.

`--defaults-file`
: Specify the path to `my.cnf`.

`--disable-binlogstats`
: Disable collection of binary log statistics.

`--disable-processlist`
: Disable collection of process state metrics.

`--disable-tablestats`
: Disable collection of table statistics.

`--disable-tablestats-limit`
: Specify the maximum number of tables for which collection of table statistics is enabled (by default, the limit is 1 000 tables).

`--disable-userstats`
: Disable collection of user statistics.

`--force`
: Force to create or update the dedicated MySQL user.

`--host`
: Specify the MySQL host name.

`--password`
: Specify the password for MySQL user with admin privileges.

`--port`
: Specify the MySQL instance port.

`--socket`
: Specify the MySQL instance socket file.

`--user`
: Specify the name of MySQL user with admin privileges.

You can also use global options that apply to any other command, as well as options that apply to adding services in general.

### DETAILED DESCRIPTION

When adding the MySQL metrics monitoring service, the **pmm-admin** tool attempts to automatically detect the local MySQL instance and MySQL superuser credentials.  You can use options to provide this information, if it cannot be detected automatically.

You can also specify the `--create-user` option to create a dedicated `pmm` user on the MySQL host that you want to monitor.  This user will be given all the necessary privileges for monitoring, and is recommended over using the MySQL superuser.

For example, to set up remote monitoring of MySQL metrics on a server located at 192.168.200.3, use a command similar to the following:

```
$ pmm-admin add mysql:metrics --user root --password root --host 192.168.200.3 --create-user
```

For more information, run **pmm-admin add** `mysql:metrics` `--help`.

### Adding MongoDB query analytics service

Use the `mongodb:queries` alias to enable MongoDB query analytics.

### USAGE

```
pmm-admin add mongodb:queries [NAME] [OPTIONS]
```

This creates the `pmm-mongodb-queries-0` service that is able to collect QAN data for multiple remote MongoDB server instances.

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following options can be used with the `mongodb:queries` alias:

`--uri`
: Specify the MongoDB instance URI with the following format:

    ```
    [mongodb://][user:pass@]host[:port][/database][?options]
    ```

    By default, it is `localhost:27017`.

You can also use global options that apply to any other command, as well as options that apply to adding services in general.

**NOTE**: PMM does not support passing SSL/TLS related parameters to `mongodb:queries`.

For more information, run **pmm-admin add** `mongodb:queries` `--help`.

### Adding MongoDB metrics service

Use the `mongodb:metrics` alias to enable MongoDB metrics monitoring.

### USAGE

```
$ pmm-admin add mongodb:metrics [NAME] [OPTIONS]
```

This creates the `pmm-mongodb-metrics-42003` service that collects local MongoDB metrics for this particular MongoDB instance.

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following options can be used with the `mongodb:metrics` alias:

`--cluster`
: Specify the MongoDB cluster name.

`--uri`
: Specify the MongoDB instance URI with the following format:

    ```
    [mongodb://][user:pass@]host[:port][/database][?options]
    ```

    By default, it is `localhost:27017`.

You can also use global options that apply to any other command, as well as options that apply to adding services in general.

For more information, run **pmm-admin add** `mongodb:metrics` `--help`.

### Monitoring a cluster

When using PMM to monitor a cluster, you should enable monitoring for each instance by using the **pmm-admin add** command. This includes each member of replica sets in shards, mongos, and all configuration servers. Make sure that for each instance you supply the cluster name via the `--cluster` option and provide its URI via the `--uri` option.

Run this command as root or by using the **sudo** command. This examples uses *127.0.0.1* as a URL.

```
$ pmm-admin add mongodb:metrics \
--uri mongodb://127.0.0.1:<port>/admin <instance name> \
--cluster <cluster name>
```

### Adding ProxySQL metrics service

Use the `proxysql:metrics` alias to enable ProxySQL performance metrics monitoring.

### USAGE

```
$ pmm-admin add proxysql:metrics [NAME] [OPTIONS]
```

This creates the `pmm-proxysql-metrics-42004` service that collects local ProxySQL performance metrics.

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following option can be used with the `proxysql:metrics` alias:

`--dsn`
: Specify the ProxySQL connection DSN. By default, it is `stats:stats@tcp(localhost:6032)/`.

You can also use global options that apply to any other command, as well as options that apply to adding services in general.

For more information, run **pmm-admin add** `proxysql:metrics` `--help`.

## Adding annotations

Use the **pmm-admin annotate** command to set notifications about important application events and display them on all dashboards. By using annotations, you can conveniently analyze the impact of application events on your database.

### USAGE

Run this command as root or by using the **sudo** command

```
$ pmm-admin annotate "Upgrade to v1.2" --tags "UX Imrovement,v1.2"
```

### OPTIONS

The **pmm-admin annotate** supports the following options:

`--tags`

> Specify one or more tags applicable to the annotation that you are
> creating. Enclose your tags in quotes and separate individual tags by a
> comma, such as “tag 1,tag 2”.

You can also use global options that apply to any other command.

## Checking network connectivity

Use the **pmm-admin check-network** command to run tests that verify connectivity between PMM Client and PMM Server.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin check-network [OPTIONS]
```

### OPTIONS

The **pmm-admin check-network** command does not have its own options, but you can use global options that apply to any other command

### DETAILED DESCRIPTION

Connection tests are performed both ways, with results separated accordingly:

* `Client --> Server`

    Pings Consul API, *PMM Query Analytics* API, and Prometheus API to make sure they are alive and reachable.

    Performs a connection performance test to see the latency from PMM Client to PMM Server.

* `Client <-- Server`

    Checks the status of Prometheus endpoints and makes sure it can scrape metrics from corresponding exporters.

    Successful pings of PMM Server from PMM Client do not mean that Prometheus is able to scrape from exporters. If the output shows some endpoints in problem state, make sure that the corresponding service is running (see **pmm-admin list**). If the services that correspond to problematic endpoints are running, make sure that firewall settings on the PMM Client host allow incoming connections for corresponding ports.

### OUTPUT EXAMPLE

```
$ pmm-admin check-network
PMM Network Status

Server Address | 192.168.100.1
Client Address | 192.168.200.1

* System Time
NTP Server (0.pool.ntp.org)         | 2017-05-03 12:05:38 -0400 EDT
PMM Server                          | 2017-05-03 16:05:38 +0000 GMT
PMM Client                          | 2017-05-03 12:05:38 -0400 EDT
PMM Server Time Drift               | OK
PMM Client Time Drift               | OK
PMM Client to PMM Server Time Drift | OK

* Connection: Client --> Server
-------------------- -------------
SERVER SERVICE       STATUS
-------------------- -------------
Consul API           OK
Prometheus API       OK
Query Analytics API  OK

Connection duration | 166.689µs
Request duration    | 364.527µs
Full round trip     | 531.216µs

* Connection: Client <-- Server
---------------- ----------- -------------------- -------- ---------- ---------
SERVICE TYPE     NAME        REMOTE ENDPOINT      STATUS   HTTPS/TLS  PASSWORD
---------------- ----------- -------------------- -------- ---------- ---------
linux:metrics    mongo-main  192.168.200.1:42000  OK       YES        -
mongodb:metrics  mongo-main  192.168.200.1:42003  PROBLEM  YES        -
```

For more information, run **pmm-admin check-network** `--help`.

## Obtaining Diagnostics Data for Support

PMM Client is able to generate a set of files for enhanced diagnostics, which is designed to be shared with Percona Support to solve an issue faster. This feature fetches logs, network, and the Percona Toolkit output. To perform data collection by PMM Client, execute:

```
pmm-admin summary
```

The output will be a tarball you can examine and/or attach to your Support ticket in the Percona’s [issue tracking system](https://jira.percona.com/projects/PMM/issues). The single file will look like this:

```
summary__2018_10_10_16_20_00.tar.gz
```

## Configuring PMM Client

Use the **pmm-admin config** command to configure how PMM Client communicates with PMM Server.

### USAGE

Run this command as root or by using the **sudo** command.

```
pmm-admin config [OPTIONS]
```

### OPTIONS

The following options can be used with the **pmm-admin config** command:

`--bind-address`
: Specify the bind address, which is also the local (private) address mapped from client address via NAT or port forwarding By default, it is set to the client address.

`--client-address`
: Specify the client address, which is also the remote (public) address for this system. By default, it is automatically detected via request to server.

`--client-name`
: Specify the client name. By default, it is set to the host name.

`--force`
: Force to set the client name on initial setup after uninstall with unreachable server.

`--server`
: Specify the address of the PMM Server host. If necessary, you can also specify the port after colon, for example:

    ```
    pmm-admin config --server 192.168.100.6:8080
    ```

    By default, port 80 is used with SSL disabled, and port 443 when SSL is enabled.

`--server-insecure-ssl`
: Enable insecure SSL (self-signed certificate).

`--SERVER_PASSWORD`
: Specify the HTTP password configured on PMM Server.

`--server-ssl`
: Enable SSL encryption for connection to PMM Server.

`--SERVER_USER`
: Specify the HTTP user configured on PMM Server (default is `pmm`).

You can also use global options that apply to any other command.

For more information, run **pmm-admin config** –help.

## Getting help for any command

Use the **pmm-admin help** command to print help for any command.

### USAGE

Run this command as root or by using the **sudo** command

```
$ pmm-admin help [COMMAND]
```

This will print help information and exit.  The actual command is not run and options are ignored.

**NOTE**: You can also use the global `-h` or `--help` option after any command to get the same help information.

### COMMANDS

You can print help information for any command or service alias.

## Getting information about PMM Client

Use the **pmm-admin info** command to print basic information about PMM Client.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin info [OPTIONS]
```

### OPTIONS

The **pmm-admin info** command does not have its own options, but you can use global options that apply to any other command

### OUTPUT

The output provides the following information:

* Version of **pmm-admin**
* PMM Server host address, and local host name and address (this can be configured using **pmm-admin config**)
* System manager that **pmm-admin** uses to manage PMM services
* Go version and runtime information

For example:

```
$ pmm-admin info

PMM Server      | 192.168.100.1
Client Name     | ubuntu-amd64
Client Address  | 192.168.200.1
Service manager | linux-systemd

Go Version      | 1.8
Runtime Info    | linux/amd64
```

For more information, run **pmm-admin info** `--help`.

## Listing monitoring services

Use the **pmm-admin list** command to list all enabled services with details.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin list [OPTIONS]
```

### OPTIONS

The **pmm-admin list** command supports global options that apply to any other command and also provides a machine friendly JSON output.

`--json`
: list the enabled services as a JSON document. The information provided in the standard tabular form is captured as keys and values. The general information about the computer where PMM Client is installed is given as top level elements:

    Note that you can quickly determine if there are any errors by inspecting the `Err` top level element in the JSON output. Similarly, the `ExternalErr` element reports errors in external services.

    The `Services` top level element contains a list of documents which represent enabled monitoring services. Each attribute in a document maps to the column in the tabular output.

    The `ExternalServices` element contains a list of documents which represent enabled external monitoring services. Each attribute in a document maps to the column in the tabular output.

### OUTPUT

The output provides the following information:

* Version of **pmm-admin**
* PMM Server host address, and local host name and address (this can be configured using **pmm-admin config**)
* System manager that **pmm-admin** uses to manage PMM services
* A table that lists all services currently managed by `pmm-admin`, with basic information about each service

For example, if you enable general OS and MongoDB metrics monitoring, output should be similar to the following:

Run this command as root or by using the **sudo** command

```
$ pmm-admin list

...

PMM Server      | 192.168.100.1
Client Name     | ubuntu-amd64
Client Address  | 192.168.200.1
Service manager | linux-systemd

---------------- ----------- ----------- -------- ---------------- --------
SERVICE TYPE     NAME        LOCAL PORT  RUNNING  DATA SOURCE      OPTIONS
---------------- ----------- ----------- -------- ---------------- --------
linux:metrics    mongo-main  42000       YES      -
mongodb:metrics  mongo-main  42003       YES      localhost:27017
```

## Pinging PMM Server

Use the **pmm-admin ping** command to verify connectivity with PMM Server.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin ping [OPTIONS]
```

If the ping is successful, it returns `OK`.

```
$ pmm-admin ping
OK, PMM server is alive.

PMM Server      | 192.168.100.1 (insecure SSL, password-protected)
Client Name     | centos7.vm
Client Address  | 192.168.200.1
```

### OPTIONS

The **pmm-admin ping** command does not have its own options, but you can use global options that apply to any other command.

For more information, run **pmm-admin ping** `--help`.

## Purging metrics data

Use the **pmm-admin purge** command to purge metrics data associated with a service on PMM Server. This is usually required after you remove a service and do not want its metrics data to show up on graphs.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin purge [SERVICE [NAME]] [OPTIONS]
```

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### SERVICES

Specify a monitoring service alias. To see which services are enabled, run **pmm-admin list**.

### OPTIONS

The **pmm-admin purge** command does not have its own options, but you can use global options that apply to any other command

For more infomation, run **pmm-admin purge** `--help`.

## Removing monitoring services

Use the **pmm-admin rm** command to remove monitoring services.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin rm [OPTIONS] [SERVICE]
```

When you remove a service, collected data remains in Metrics Monitor on PMM Server. To remove the collected data, use the **pmm-admin purge** command.

### OPTIONS

The following option can be used with the **pmm-admin rm** command:

`--all`
: Remove all monitoring services.

You can also use global options that apply to any other command.

### SERVICES

Specify a monitoring service alias. To see which services are enabled, run **pmm-admin list**.

### EXAMPLES

* To remove all services enabled for this PMM Client:

    ```
    $ pmm-admin rm --all
    ```

* To remove all services related to MySQL:

    ```
    $ pmm-admin rm mysql
    ```

* To remove only `mongodb:metrics` service:

    ```
    $ pmm-admin rm mongodb:metrics
    ```

For more information, run **pmm-admin rm** –help.

## Removing orphaned services

Use the **pmm-admin repair** command to remove information about orphaned services from PMM Server. This can happen if you removed services locally while PMM Server was not available (disconnected or shut down), for example, using the **pmm-admin uninstall** command.

### USAGE

Run this command as root or by using the **sudo** command

```
$ pmm-admin repair [OPTIONS]
```

### OPTIONS

The **pmm-admin repair** command does not have its own options, but you can use global options that apply to any other command.

For more information, run **pmm-admin repair** –help.

## Restarting monitoring services

Use the **pmm-admin restart** command to restart services managed by this PMM Client. This is the same as running **pmm-admin stop** and **pmm-admin start**.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin restart [SERVICE [NAME]] [OPTIONS]
```

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following option can be used with the **pmm-admin restart** command:

`--all`

    Restart all monitoring services.

You can also use global options that apply to any other command.

### SERVICES

Specify a monitoring service alias that you want to restart. To see which services are available, run **pmm-admin list**.

### EXAMPLES

* To restart all available services for this PMM Client:

    ```
    # pmm-admin restart --all
    ```

* To restart all services related to MySQL:

    ```
    $ pmm-admin restart mysql
    ```

* To restart only the `mongodb:metrics` service:

    ```
    $ pmm-admin restart mongodb:metrics
    ```

For more information, run **pmm-admin restart** `--help`.

## Getting passwords used by PMM Client

Use the **pmm-admin show-passwords** command to print credentials stored in the configuration file (by default: `/usr/local/percona/pmm-client/pmm.yml`).

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin show-passwords [OPTIONS]
```

### OPTIONS

The **pmm-admin show-passwords** command does not have its own options, but you can use global options that apply to any other command

### OUTPUT

This command prints HTTP authentication credentials and the password for the `pmm` user that is created on the MySQL instance if you specify the `--create-user` option when adding a service.

Run this command as root or by using the **sudo** command

```
$ pmm-admin show-passwords
HTTP basic authentication
User     | aname
Password | secr3tPASS

MySQL new user creation
Password | g,3i-QR50tQJi9M1yl9-
```

For more information, run **pmm-admin show-passwords**  `--help`.

## Starting monitoring services

Use the **pmm-admin start** command to start services managed by this PMM Client.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin start [SERVICE [NAME]] [OPTIONS]
```

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following option can be used with the **pmm-admin start** command:

`--all`
: Start all monitoring services.

You can also use global options that apply to any other command.

### SERVICES

Specify a monitoring service alias that you want to start. To see which services are available, run **pmm-admin list**.

### EXAMPLES

* To start all available services for this PMM Client:

    ```
    $ pmm-admin start --all
    ```

* To start all services related to MySQL:

    ```
    $ pmm-admin start mysql
    ```

* To start only the `mongodb:metrics` service:

    ```
    $ pmm-admin start mongodb:metrics
    ```

For more information, run **pmm-admin start** `--help`.

## Stopping monitoring services

Use the **pmm-admin stop** command to stop services managed by this PMM Client.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin stop [SERVICE [NAME]] [OPTIONS]
```

**NOTE**: It should be able to detect the local PMM Client name, but you can also specify it explicitly as an argument.

### OPTIONS

The following option can be used with the **pmm-admin stop** command:

`--all`
: Stop all monitoring services.

You can also use global options that apply to any other command.

### SERVICES

Specify a monitoring service alias that you want to stop. To see which services are available, run **pmm-admin list**.

### EXAMPLES

* To stop all available services for this PMM Client:

    ```
    $ pmm-admin stop --all
    ```

* To stop all services related to MySQL:

    ```
    $ pmm-admin stop mysql
    ```

* To stop only the `mongodb:metrics` service:

    ```
    $ pmm-admin stop mongodb:metrics
    ```

For more information, run **pmm-admin stop** `--help`.

## Cleaning Up Before Uninstall

Use the **pmm-admin uninstall** command to remove all services even if PMM Server is not available.  To uninstall PMM correctly, you first need to remove all services, then uninstall PMM Client, and then stop and remove PMM Server.  However, if PMM Server is not available (disconnected or shut down), **pmm-admin rm** will not work.  In this case, you can use **pmm-admin uninstall** to force the removal of monitoring services enabled for PMM Client.

**NOTE**: Information about services will remain in PMM Server, and it will not let you add those services again.  To remove information about orphaned services from PMM Server, once it is back up and available to PMM Client, use the **pmm-admin repair** command.

### USAGE

Run this command as root or by using the **sudo** command

```
pmm-admin uninstall [OPTIONS]
```

### OPTIONS

The **pmm-admin uninstall** command does not have its own options, but you can use global options that apply to any other command.

For more information, run **pmm-admin uninstall** `--help`.

## Monitoring Service Aliases

The following aliases are used to designate PMM services that you want to add, remove, restart, start, or stop:

| Alias              | Services                                                                                    |
| ------------------ | ------------------------------------------------------------------------------------------- |
| `linux:metrics`    | General system metrics monitoring service                                                   |
| `mysql:metrics`    | MySQL metrics monitoring service                                                            |
| `mysql:queries`    | MySQL query analytics service                                                               |
| `mongodb:metrics`  | MongoDB metrics monitoring service                                                          |
| `mongodb:queries`  | MongoDB query analytics service                                                             |
| `proxysql:metrics` | ProxySQL metrics monitoring service                                                         |
| `mysql`            | Complete MySQL instance monitoring: `linux:metrics`, `mysql:metrics`, `mysql:queries`       |
| `mongodb`          | Complete MongoDB instance monitoring: `linux:metrics`, `mongodb:metrics`, `mongodb:queries` |
