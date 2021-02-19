# MySQL

PMM supports all commonly used variants of MySQL, including
Percona Server, MariaDB, and Amazon RDS.  To prevent data loss and
performance issues, PMM does not automatically change MySQL configuration.
However, there are certain recommended settings that help maximize monitoring
efficiency. These recommendations depend on the variant and version of MySQL
you are using, and mostly apply to very high loads.

PMM can collect query data either from the *slow query log* or from
*Performance Schema*.  The *slow query log* provides maximum details, but can
impact performance on heavily loaded systems. On Percona Server the query
sampling feature may reduce the performance impact.

*Performance Schema* is generally better for recent versions of other MySQL
variants. For older MySQL variants, which have neither sampling, nor
*Performance Schema*, configure logging only slow queries.

!!! alert alert-info "Note"

    MySQL with too many tables can lead to PMM Server overload due to the streaming of too much time series data. It can also lead to too many queries from `mysqld_exporter` causing extra load on MySQL. Therefore PMM Server disables most consuming `mysqld_exporter` collectors automatically if there are more than 1000 tables.

You can add configuration examples provided below to `my.cnf` and
restart the server or change variables dynamically using the following syntax:

```sql
SET GLOBAL <var_name>=<var_value>
```

The following sample configurations can be used depending on the variant and
version of MySQL:


* If you are running Percona Server (or XtraDB Cluster), configure the *slow query log* to capture all queries and enable sampling. This will provide the most amount of information with the lowest overhead.

    ```ini
    log_output=file
    slow_query_log=ON
    long_query_time=0
    log_slow_rate_limit=100
    log_slow_rate_type=query
    log_slow_verbosity=full
    log_slow_admin_statements=ON
    log_slow_slave_statements=ON
    slow_query_log_always_write_time=1
    slow_query_log_use_global_control=all
    innodb_monitor_enable=all
    userstat=1
    ```

* If you are running MySQL 5.6+ or MariaDB 10.0+, see [Performance Schema](#performance-schema).

    ```ini
    innodb_monitor_enable=all
    performance_schema=ON
    ```

* If you are running MySQL 5.5 or MariaDB 5.5, configure logging only slow queries to avoid high performance overhead.

    ```ini
    log_output=file
    slow_query_log=ON
    long_query_time=0
    log_slow_admin_statements=ON
    log_slow_slave_statements=ON
    ```

!!! alert alert-warning "Caution"
    This may affect the quality of monitoring data gathered by Query Analytics.


## Creating a MySQL User Account for PMM

When adding a MySQL instance to monitoring, you can specify the MySQL
server superuser account credentials.  However, monitoring with the superuser
account is not advised. It’s better to create a user with only the necessary
privileges for collecting data.

As an example, the user `pmm` can be created manually with the necessary
privileges and pass its credentials when adding the instance.

To enable complete MySQL instance monitoring, a command similar to the
following is recommended:

```sh
sudo pmm-admin add mysql --username pmm --password <password>
```

Of course this user should have necessary privileges for collecting data. If
the `pmm` user already exists, you can grant the required privileges as
follows:

```sql
CREATE USER 'pmm'@'localhost' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 10;
GRANT SELECT, PROCESS, SUPER, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@'localhost';
```

## Performance Schema

The default source of query data for PMM is the *slow query log*.  It is
available in MySQL 5.1 and later versions.  Starting from MySQL 5.6
(including Percona Server 5.6 and later), you can choose to parse query data
from the *Performance Schema* instead of *slow query log*.  Starting from MySQL
5.6.6, *Performance Schema* is enabled by default.

*Performance Schema* is not as data-rich as the *slow query log*, but it has all the
critical data and is generally faster to parse. If you are not running
Percona Server (which supports sampling for the slow query log), then *Performance Schema* is a better alternative.

!!! alert alert-info "Note"

    Use of the performance schema is off by default in MariaDB 10.x.

To use *Performance Schema*, set the `performance_schema` variable to `ON`:

```sql
SHOW VARIABLES LIKE 'performance_schema';
```

```
+--------------------+-------+
| Variable_name      | Value |
+--------------------+-------+
| performance_schema | ON    |
+--------------------+-------+
```

If this variable is not set to **ON**, add the the following lines to the
MySQL configuration file `my.cnf` and restart MySQL:

```ini
[mysql]
performance_schema=ON
```

If you are running a custom Performance Schema configuration, make sure that the
`statements_digest` consumer is enabled:

```sql
select * from setup_consumers;
```

```
+----------------------------------+---------+
| NAME                             | ENABLED |
+----------------------------------+---------+
| events_stages_current            | NO      |
| events_stages_history            | NO      |
| events_stages_history_long       | NO      |
| events_statements_current        | YES     |
| events_statements_history        | YES     |
| events_statements_history_long   | NO      |
| events_transactions_current      | NO      |
| events_transactions_history      | NO      |
| events_transactions_history_long | NO      |
| events_waits_current             | NO      |
| events_waits_history             | NO      |
| events_waits_history_long        | NO      |
| global_instrumentation           | YES     |
| thread_instrumentation           | YES     |
| statements_digest                | YES     |
+----------------------------------+---------+
15 rows in set (0.00 sec)
```

!!! alert alert-info "Note"

    *Performance Schema* instrumentation is enabled by default in MySQL 5.6.6 and later versions. It is not available at all in MySQL versions prior to 5.6.

    If certain instruments are not enabled, you will not see the corresponding graphs in the MySQL Performance Schema dashboard.  To enable full instrumentation, set the option `--performance_schema_instrument` to `'%=on'` when starting the MySQL server:

    ```sh
    mysqld --performance-schema-instrument='%=on'
    ```

* If you are running any MariaDB version, there is no Explain or Example data shown by default in Query Analytics. A workaround is to run this SQL command:

    ```sql
    UPDATE performance_schema.setup_instruments SET ENABLED = 'YES', TIMED = 'YES'
    WHERE NAME LIKE 'statement/%';
    UPDATE performance_schema.setup_consumers SET ENABLED = 'YES'
    WHERE NAME LIKE '%statements%';
    ```

    This option can cause additional overhead and should be used with care.

If the instance is already running, configure the QAN agent to collect data
from *Performance Schema*:

1. Open the *PMM Query Analytics* dashboard.

2. Click the *Settings* button.

3. Open the *Settings* section.

4. Select `Performance Schema` in the *Collect from* drop-down list.

5. Click *Apply* to save changes.

If you are adding a new monitoring instance with the `pmm-admin` tool, use the
`--query-source='perfschema'` option:

Run this command as root or by using the `sudo` command

```sh
pmm-admin add mysql --username=pmm --password=pmmpassword --query-source='perfschema' ps-mysql 127.0.0.1:3306
```

For more information, run `pmm-admin add mysql --help`.

## Adding MySQL Service Monitoring

You add MySQL services (Metrics and Query Analytics) with the following command:

## USAGE

```sh
pmm-admin add mysql --query-source=slowlog --username=pmm --password=pmm
```

where username and password are credentials for the monitored MySQL access, which will be used locally on the database host. Additionally, two positional arguments can be appended to the command line flags: a service name to be used by PMM, and a service address. If not specified, they are substituted automatically as `<node>-mysql` and `127.0.0.1:3306`.

The command line and the output of this command may look as follows:

```sh
pmm-admin add mysql --query-source=slowlog --username=pmm --password=pmm sl-mysql 127.0.0.1:3306
```

```
MySQL Service added.
Service ID  : /service_id/a89191d4-7d75-44a9-b37f-a528e2c4550f
Service name: sl-mysql
```

!!! alert alert-info "Note"

    There are two possible sources for query metrics provided by MySQL to get data for the Query Analytics: the [slow log](#slow-log-settings) and the [Performance Schema](#performance-schema).

    The `--query-source` option can be used to specify it, either as `slowlog` (it is also used by default if nothing specified) or as `perfschema`:

    ```sh
    pmm-admin add mysql --username=pmm --password=pmm --query-source=perfschema ps-mysql 127.0.0.1:3306
    ```

Beside positional arguments shown above you can specify service name and service address with the following flags: `--service-name`, `--host` (the hostname or IP address of the service), and `--port` (the port number of the service). If both flag and positional argument are present, flag gains higher priority. Here is the previous example modified to use these flags:

```sh
pmm-admin add mysql --username=pmm --password=pmm --service-name=ps-mysql --host=127.0.0.1 --port=3306
```

!!! alert alert-info "Note"

    It is also possible to add MySQL instance using UNIX socket with use of a special `--socket` flag followed with the path to a socket without username, password and network type:

    ```sh
    pmm-admin add mysql --socket=/var/path/to/mysql/socket
    ```

After adding the service you can view MySQL metrics or examine the added node on the new PMM Inventory Dashboard.

## MySQL InnoDB Metrics

Collecting metrics and statistics for graphs increases overhead.  You can keep
collecting and graphing low-overhead metrics all the time, and enable
high-overhead metrics only when troubleshooting problems.

InnoDB metrics provide detailed insight about InnoDB operation.  Although you
can select to capture only specific counters, their overhead is low even when
they all are enabled all the time. To enable all InnoDB metrics, set the
global variable `innodb_monitor_enable` to `all`:

```sql
SET GLOBAL innodb_monitor_enable=all
```

## Slow Log Settings

If you are running Percona Server for MySQL, a properly configured slow query log will
provide the most amount of information with the lowest overhead.  In other
cases, use Performance Schema if it is supported.

### Configuring the Slow Log File

The first and obvious variable to enable is `slow_query_log` which controls the global Slow Query on/off status.

Secondly, verify that the log is sent to a FILE instead of a TABLE. This is controlled with the `log_output` variable.

By definition, the slow query log is supposed to capture only *slow queries*. These are the queries the execution time of which is above a certain threshold. The threshold is defined by the `long_query_time` variable.

In heavily-loaded applications, frequent fast queries can actually have a much bigger impact on performance than rare slow queries.  To ensure comprehensive analysis of your query traffic, set the `long_query_time` to **0** so that all queries are captured.

### Fine tune

Depending on the amount of traffic, logging could become aggressive and resource consuming. However, Percona Server for MySQL provides a way to throttle the level of intensity of the data capture without compromising information. The most important variable is `log_slow_rate_limit`, which controls the *query sampling* in Percona Server for MySQL. Details on that variable can be found [here](https://www.percona.com/doc/percona-server/LATEST/diagnostics/slow_extended.html#log_slow_rate_limit).

A possible problem with query sampling is that rare slow queries might not get captured at all.  To avoid this, use the `slow_query_log_always_write_time` variable to specify which queries should ignore sampling.  That is, queries with longer execution time will always be captured by the slow query log.

### Slow log file rotation

PMM will take care of rotating and removing old slow log files, only if you set the `--size-slow-logs` variable via [pmm-admin](../../details/commands/pmm-admin.md).

When the limit is reached, PMM will remove the previous old slow log file, rename the current file with the suffix `.old`, and execute the MySQL command `FLUSH LOGS`. It will only keep one old file. Older files will be deleted on the next iteration.
