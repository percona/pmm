# Configuring MySQL for Best Results

PMM supports all commonly used variants of MySQL, including Percona Server, MariaDB, and Amazon RDS.  To prevent data loss and performance issues, PMM does not automatically change MySQL configuration. However, there are certain recommended settings that help maximize monitoring efficiency. These recommendations depend on the variant and version of MySQL you are using, and mostly apply to very high loads.

PMM can collect query data either from the *slow query log* or from *Performance Schema*.  The *slow query log* provides maximum details, but can impact performance on heavily loaded systems. On Percona Server the query sampling feature may reduce the performance impact.

*Performance Schema* is generally better for recent versions of other MySQL variants. For older MySQL variants, which have neither sampling, nor *Performance Schema*, configure logging only slow queries.

[TOC]

You can add configuration examples provided in this guide to `my.cnf` and restart the server or change variables dynamically using the following syntax:

```
SET GLOBAL <var_name>=<var_value>
```

The following sample configurations can be used depending on the variant and version of MySQL:

* If you are running Percona Server (or XtraDB Cluster), configure the *slow query log* to capture all queries and enable sampling. This will provide the most amount of information with the lowest overhead.

    ```
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

* If you are running MySQL 5.6+ or MariaDB 10.0+, configure Configuring Performance Schema.

    ```
    innodb_monitor_enable=all
    performance_schema=ON
    ```

* If you are running MySQL 5.5 or MariaDB 5.5, configure logging only slow queries to avoid high performance overhead.

    **NOTE**: This may affect the quality of monitoring data gathered by QAN.

    ```
    log_output=file
    slow_query_log=ON
    long_query_time=0
    log_slow_admin_statements=ON
    log_slow_slave_statements=ON
    ```

## Creating a MySQL User Account to Be Used with PMM

When adding a MySQL instance to monitoring, you can specify the MySQL server superuser account credentials.  However, monitoring with the superuser account is not secure. If you also specify the `--create-user` option, it will create a user with only the necessary privileges for collecting data.

You can also set up the `pmm` user manually with necessary privileges and pass its credentials when adding the instance.

To enable complete MySQL instance monitoring, a command similar to the following is recommended:

```
$ sudo pmm-admin add mysql --user root --password root --create-user
```

The superuser credentials are required only to set up the `pmm` user with necessary privileges for collecting data.  If you want to create this user yourself, the following privileges are required:

```
GRANT SELECT, PROCESS, SUPER, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@' localhost' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 10;
GRANT SELECT, UPDATE, DELETE, DROP ON performance_schema.* TO 'pmm'@'localhost';
```

If the `pmm` user already exists, simply pass its credential when you add the instance:

```
$ sudo pmm-admin add mysql --user pmm --password pass
```

For more information, run as root **pmm-admin add** `mysql --help`.

## Configuring the slow query log in Percona Server

If you are running Percona Server, a properly configured slow query log will provide the most amount of information with the lowest overhead.  In other cases, use Performance Schema if it is supported.

By definition, the slow query log is supposed to capture only *slow queries*. These are the queries the execution time of which is above a certain threshold. The threshold is defined by the [`long_query_time`](http://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_long_query_time) variable.

In heavily loaded applications, frequent fast queries can actually have a much bigger impact on performance than rare slow queries.  To ensure comprehensive analysis of your query traffic, set the `long_query_time` to **0** so that all queries are captured.

However, capturing all queries can consume I/O bandwidth and cause the *slow query log* file to quickly grow very large. To limit the amount of queries captured by the *slow query log*, use the *query sampling* feature available in Percona Server.

The [`log_slow_rate_limit`](https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_rate_limit) variable defines the fraction of queries captured by the *slow query log*.  A good rule of thumb is to have approximately 100 queries logged per second.  For example, if your Percona Server instance processes 10_000 queries per second, you should set `log_slow_rate_limit` to `100` and capture every 100th query for the *slow query log*.

!!! note
    When using query sampling, set [`log_slow_rate_type`](https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_rate_type) to `query` so that it applies to queries, rather than sessions.

    It is also a good idea to set [`log_slow_verbosity`](https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_verbosity) to `full` so that maximum amount of information about each captured query is stored in the slow query log.

A possible problem with query sampling is that rare slow queries might not get captured at all.  To avoid this, use the [`slow_query_log_always_write_time`](https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#slow_query_log_always_write_time) variable to specify which queries should ignore sampling.  That is, queries with longer execution time will always be captured by the slow query log.

By default, slow query log settings apply only to new sessions.  If you want to configure the slow query log during runtime and apply these settings to existing connections, set the [`slow_query_log_use_global_control`](https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#slow_query_log_use_global_control) variable to `all`.

## Configuring Performance Schema

The default source of query data for PMM is the *slow query log*.  It is available in MySQL 5.1 and later versions.  Starting from MySQL 5.6 (including Percona Server 5.6 and later), you can choose to parse query data from the *Performance Schema* instead of *slow query log*.  Starting from MySQL 5.6.6, *Performance Schema* is enabled by default.

*Performance Schema* is not as data-rich as the *slow query log*, but it has all the critical data and is generally faster to parse. If you are not running Percona Server (which supports sampling for the slow query log), then *Performance Schema* is a better alternative.

To use *Performance Schema*, set the `performance_schema` variable to `ON`:

```
mysql> SHOW VARIABLES LIKE 'performance_schema';
+--------------------+-------+
| Variable_name      | Value |
+--------------------+-------+
| performance_schema | ON    |
+--------------------+-------+
```

If this variable is not set to **ON**, add the the following lines to the MySQL configuration file `my.cnf` and restart MySQL:

```
[mysql]
performance_schema=ON
```

If you are running a custom Performance Schema configuration, make sure that the `statements_digest` consumer is enabled:

```
mysql> select * from setup_consumers;
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

If the instance is already running, configure the QAN agent to collect data from *Performance Schema*:

1. Open the *PMM Query Analytics* dashboard.
2. Click the Settings button.
3. Open the Settings section.
4. Select `Performance Schema` in the Collect from drop-down list.
5. Click Apply to save changes.

If you are adding a new monitoring instance with the **pmm-admin** tool, use the `--query-source` *perfschema* option:

Run this command as root or by using the **sudo** command

```
pmm-admin add mysql --user root --password root --create-user --query-source perfschema
```

For more information, run **pmm-admin add** `mysql --help`.












## Use **logrotate** instead of the slow log rotation feature to manage the MySQL Slow Log

By default, PMM manages the slow log for the added MySQL monitoring service on the computer where PMM Client is installed. This example demonstrates how to substitute **logrotate** for this default behavior.

## Disable the default behavior of the slow log rotation

The first step is to disable the default slow log rotation when adding the MySQL monitoring service.

For this, set the `--slow-log-rotation` to *false*.

Run this command as root or by using the **sudo** command

```
pmm-admin rm mysql:queries
pmm-admin add mysql:queries --slow-log-rotation=false
```

On PMM Server, you can check the value of the Slow logs rotation field on the QAN Settings page. It should be *OFF*.

On PMM Client (the host where you ran **pmm-admin add** command to add the MySQL monitoring service), use the **pmm-admin list** command to determine if the *slow log* rotation is disabled.

```
$ pmm-admin list

PMM Server      | 127.0.0.1
Client Name     | percona
Client Address  | 172.17.0.1
Service Manager | linux-systemd

-------------- -------- ----------- -------- ------------------------------------------- --------------------------------------------------------------------------------------
SERVICE TYPE   NAME     LOCAL PORT  RUNNING  DATA SOURCE                                 OPTIONS
-------------- -------- ----------- -------- ------------------------------------------- --------------------------------------------------------------------------------------
mysql:queries  percona  -           YES      root:***@unix(/var/run/mysqld/mysqld.sock)  query_source=slowlog, query_examples=true, slow_log_rotation=false, retain_slow_logs=1
```

## Set up **logrotate** to manage the slow log rotation

**logrotate** is a popular utility for managing log files. You can install it using the package manager (apt or yum, for example) of your Linux distribution.

After you add a MySQL with `--slow-log-rotation` set to **false**, you can run **logrotate** as follows.

Run this command as root or by using the **sudo** command

```
$ logrotate CONFIG_FILE
```

*CONFIG_FILE* is a placeholder for a configuration file that you should supply to **logrotate** as a mandatory parameter. To use **logrotate** to manage the *slow log* for PMM, you may supply a file with the following contents.

This is a basic example of **logrotate** for the MySQL slow logs at 1G for 30 copies (30GB).

```
/var/mysql/mysql-slow.log {
             nocompress
             create 660 mysql mysql
             size 1G
             dateext
             missingok
             notifempty
             sharedscripts
             postrotate
             /bin/mysql -e 'SELECT @@global.long_query_time INTO @LQT_SAVE; \
             SET GLOBAL long_query_time=2000; SELECT SLEEP(2); \
             FLUSH SLOW LOGS; SELECT SLEEP(2); SET GLOBAL long_query_time=@LQT_SAVE;'
             endscript
             rotate 30
}
```

For more information about how to use **logrotate**, refer to its documentation installed along with the program.



















## Configuring MySQL 8.0 for PMM

MySQL 8 (in version 8.0.4) changes the way clients are authenticated by default. The `default_authentication_plugin` parameter is set to `caching_sha2_password`. This change of the default value implies that MySQL drivers must support the SHA-256 authentication. Also, the communication channel with MySQL 8 must be encrypted when using `caching_sha2_password`.

The MySQL driver used with PMM does not yet support the SHA-256 authentication.

With currently supported versions of MySQL, PMM requires that a dedicated MySQL user be set up. This MySQL user should be authenticated using the `mysql_native_password` plugin.  Although MySQL is configured to support SSL clients, connections to MySQL Server are not encrypted.

There are two workarounds to be able to add MySQL Server version 8.0.4 or higher as a monitoring service to PMM:

1. Alter the MySQL user that you plan to use with PMM
2. Change the global MySQL configuration

### Altering the MySQL User

Provided you have already created the MySQL user that you plan to use with PMM, alter this user as follows:

```
mysql> ALTER USER pmm@'localhost' IDENTIFIED WITH mysql_native_password BY '$eCR8Tp@s$w*rD';
```

Then, pass this user to `pmm-admin add` as the value of the `--user` parameter.

This is a preferred approach as it only weakens the security of one user.

### Changing the global MySQL Configuration

A less secure approach is to set `default_authentication_plugin` to the value **mysql_native_password** before adding it as a monitoring service. Then, restart your MySQL Server to apply this change.

```
[mysqld]
default_authentication_plugin=mysql_native_password
```

## Settings for Dashboards

Not all dashboards in Metrics Monitor are available by default for all MySQL variants and configurations: Oracle’s MySQL, Percona Server. or MariaDB. Some graphs require Percona Server, specialized plugins, or additional configuration.

Collecting metrics and statistics for graphs increases overhead.  You can keep collecting and graphing low-overhead metrics all the time, and enable high-overhead metrics only when troubleshooting problems.

### MySQL InnoDB Metrics

InnoDB metrics provide detailed insight about InnoDB operation.  Although you can select to capture only specific counters, their overhead is low even when they all are enabled all the time.  To enable all InnoDB metrics, set the global variable `innodb_monitor_enable` to `all`:

```
mysql> SET GLOBAL innodb_monitor_enable=all
```

### MySQL User Statistics

User statistics is a feature of Percona Server and MariaDB.  It provides information about user activity, individual table and index access.  In some cases, collecting user statistics can lead to high overhead, so use this feature sparingly.

To enable user statistics, set the `userstat` variable to `1`.

### Percona Server Query Response Time Distribution

Query response time distribution is a feature available in Percona Server.  It provides information about changes in query response time for different groups of queries, often allowing to spot performance problems before they lead to serious issues.

To enable collection of query response time:

1. Install the `QUERY_RESPONSE_TIME` plugins:

    ```
    mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_AUDIT SONAME 'query_response_time.so';
    mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME SONAME 'query_response_time.so';
    mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_READ SONAME 'query_response_time.so';
    mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_WRITE SONAME 'query_response_time.so';
    ```

2. Set the global varible `query_response_time_stats` to `ON`:

    ```
    mysql> SET GLOBAL query_response_time_stats=ON;
    ```

## Executing Custom Queries

Starting from the version 1.15.0, PMM provides user the ability to take a SQL `SELECT` statement and turn the result set into metric series in PMM. The queries are executed at the LOW RESOLUTION level, which by default is every 60 seconds. A key advantage is that you can extend PMM to profile metrics unique to your environment (see users table example below), or to introduce support for a table that isn’t part of PMM yet. This feature is on by default and only requires that you edit the configuration file and use vaild YAML syntax. The default configuration file location is `/usr/local/percona/pmm-client/queries-mysqld.yml`.

### Example - Application users table

We’re going to take a users table of upvotes and downvotes and turn this into two metric series, with a set of labels. Labels can also store a value. You can filter against labels.

#### Browsing metrics series using Advanced Data Exploration Dashboard

Lets look at the output so we understand the goal - take data from a MySQL table and store in PMM, then display as a metric series. Using the Advanced Data Exploration Dashboard you can review your metric series.

#### MySQL table

Lets assume you have the following users table that includes true/false, string, and integer types.

```
SELECT * FROM `users`
+----+------+--------------+-----------+------------+-----------+---------------------+--------+---------+-----------+
| id | app  | user_type    | last_name | first_name | logged_in | active_subscription | banned | upvotes | downvotes |
+----+------+--------------+-----------+------------+-----------+---------------------+--------+---------+-----------+
|  1 | app2 | unprivileged | Marley    | Bob        |         1 |                   1 |      0 |     100 |        25 |
|  2 | app3 | moderator    | Young     | Neil       |         1 |                   1 |      1 |     150 |        10 |
|  3 | app4 | unprivileged | OConnor   | Sinead     |         1 |                   1 |      0 |      25 |        50 |
|  4 | app1 | unprivileged | Yorke     | Thom       |         0 |                   1 |      0 |     100 |       100 |
|  5 | app5 | admin        | Buckley   | Jeff       |         1 |                   1 |      0 |     175 |         0 |
+----+------+--------------+-----------+------------+-----------+---------------------+--------+---------+-----------+
```

#### Explaining the YAML syntax

We’ll go through a simple example and mention what’s required for each line. The metric series is constructed based on the first line and appends the column name to form metric series. Therefore the number of metric series per table will be the count of columns that are of type `GAUGE` or `COUNTER`. This metric series will be called `app1_users_metrics_downvotes`:

```
app1_users_metrics:                                 ## leading section of your metric series.
  query: "SELECT * FROM app1.users"                 ## Your query. Don't forget the schema name.
  metrics:                                          ## Required line to start the list of metric items
    - downvotes:                                    ## Name of the column returned by the query. Will be appended to the metric series.
        usage: "COUNTER"                            ## Column value type.  COUNTER will make this a metric series.
        description: "Number of upvotes"            ## Helpful description of the column.
```

#### Full queries-mysqld.yml example

Each column in the `SELECT` is named in this example, but that isn’t required, you can use a `SELECT \*` as well. Notice the format of schema.table for the query is included.

```
---
app1_users_metrics:
  query: "SELECT app,first_name,last_name,logged_in,active_subscription,banned,upvotes,downvotes FROM app1.users"
  metrics:
    - app:
        usage: "LABEL"
        description: "Name of the Application"
    - user_type:
        usage: "LABEL"
        description: "User's privilege level within the Application"
    - first_name:
        usage: "LABEL"
        description: "User's First Name"
    - last_name:
        usage: "LABEL"
        description: "User's Last Name"
    - logged_in:
        usage: "LABEL"
        description: "User's logged in or out status"
    - active_subscription:
        usage: "LABEL"
        description: "Whether User has an active subscription or not"
    - banned:
        usage: "LABEL"
        description: "Whether user is banned or not"
    - upvotes:
        usage: "COUNTER"
        description: "Count of upvotes the User has earned. Upvotes once granted cannot be revoked, so the number can only increase."
    - downvotes:
        usage: "GAUGE"
        description: "Count of downvotes the User has earned. Downvotes can be revoked so the number can increase as well as decrease."
...
```

This custom query description should be placed in a YAML file (`queries-mysqld.yml` by default) on the corresponding server with MySQL.


In order to modify the location of the queries file, for example if you have multiple mysqld instances per server, you need to explicitly identify to the PMM Server MySQL with the `pmm-admin add` command after the double dash:

```
pmm-admin add mysql:metrics ... -- --queries-file-name=/usr/local/percona/pmm-client/query.yml
```
