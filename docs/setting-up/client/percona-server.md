# Percona Server

Not all dashboards in Metrics Monitor are available by default for all MySQL
variants and configurations: Oracle’s MySQL, Percona Server. or MariaDB.
Some graphs require Percona Server, and specialized plugins, or additional
configuration.

## `log_slow_rate_limit`

The `log_slow_rate_limit` variable defines the fraction of queries captured by
the *slow query log*.  A good rule of thumb is to have approximately 100 queries
logged per second.  For example, if your Percona Server instance processes
10_000 queries per second, you should set `log_slow_rate_limit` to `100` and
capture every 100th query for the *slow query log*.

!!! note

    When using query sampling, set `log_slow_rate_type` to `query` so that it applies to queries, rather than sessions.

    It is also a good idea to set `log_slow_verbosity` to `full` so that maximum amount of information about each captured query is stored in the slow query log.

## `log_slow_verbosity`

`log_slow_verbosity` variable specifies how much information to include in
the slow query log. It is a good idea to set `log_slow_verbosity` to `full`
so that maximum amount of information about each captured query is stored.

## `slow_query_log_use_global_control`

By default, slow query log settings apply only to new sessions.  If you want to
configure the slow query log during runtime and apply these settings to existing
connections, set the `slow_query_log_use_global_control` variable to `all`.

## Query Response Time Plugin

Query response time distribution is a feature available in Percona Server.  It
provides information about changes in query response time for different groups
of queries, often allowing to spot performance problems before they lead to
serious issues.

To enable collection of query response time:

1. Install the `QUERY_RESPONSE_TIME` plugins:

    ```sql
    INSTALL PLUGIN QUERY_RESPONSE_TIME_AUDIT SONAME 'query_response_time.so';
    INSTALL PLUGIN QUERY_RESPONSE_TIME SONAME 'query_response_time.so';
    INSTALL PLUGIN QUERY_RESPONSE_TIME_READ SONAME 'query_response_time.so';
    INSTALL PLUGIN QUERY_RESPONSE_TIME_WRITE SONAME 'query_response_time.so';
    ```


2. Set the global variable `query_response_time_stats` to `ON`:

    ```sql
    SET GLOBAL query_response_time_stats=ON;
    ```

## MySQL User Statistics (`userstat`)

User statistics is a feature of Percona Server and MariaDB.  It provides
information about user activity, individual table and index access.  In some
cases, collecting user statistics can lead to high overhead, so use this feature
sparingly.

To enable user statistics, set the `userstat` variable to `1`.
