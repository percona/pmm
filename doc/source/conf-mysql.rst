.. _conf-mysql:

================================================================================
Configuring MySQL for Best Results
================================================================================

|pmm| supports all commonly used variants of |mysql|, including
|percona-server|, |mariadb|, and |amazon-rds|.  To prevent data loss and
performance issues, |pmm| does not automatically change |mysql| configuration.
However, there are certain recommended settings that help maximize monitoring
efficiency. These recommendations depend on the variant and version of |mysql|
you are using, and mostly apply to very high loads.

|pmm| can collect query data either from the |slow-query-log| or from
|performance-schema|.  Using the |slow-query-log| to capture all queries provides
maximum details, but can impact performance on heavily loaded systems unless it
is used with the query sampling feature available only in |percona-server|.
|performance-schema| is generally better for recent versions of other |mysql|
variants. For older |mysql| variants, which have neither sampling, nor
|performance-schema|, configure logging only slow queries.

You can add configuration examples provided in this guide to :file:`my.cnf` and
restart the server or change variables dynamically using the following syntax:

.. code-block:: sql

   SET GLOBAL <var_name>=<var_value>

The following sample configurations can be used depending on the variant and
version of |mysql|:

* If you are running |percona-server| (or |xtradb-cluster|), configure the
  |slow-query-log| to capture all queries and enable sampling. This will provide
  the most amount of information with the lowest overhead.

  ::

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

* If you are running |mysql| 5.6+ or |mariadb| 10.0+, configure
  :ref:`perf-schema`.

  ::

   innodb_monitor_enable=all
   performance_schema=ON

* If you are running |mysql| 5.5 or |mariadb| 5.5, configure logging only slow
  queries to avoid high performance overhead.

  .. note:: This may affect the quality of monitoring data gathered by
            |qan.intro|.

  ::

   log_output=file
   slow_query_log=ON
   long_query_time=0.01
   log_slow_admin_statements=ON
   log_slow_slave_statements=ON

.. _slow-log-settings:

Configuring the |slow-query-log| in |percona-server|
================================================================================

If you are running |percona-server|, a properly configured slow query log will
provide the most amount of information with the lowest overhead.  In other
cases, use :ref:`Performance Schema <perf-schema-settings>` if it is supported.

By definition, the slow query log is supposed to capture only *slow queries*.
These are the queries the execution time of which is above a certain
threshold. The threshold is defined by the |long_query_time|_ variable.

In heavily loaded applications, frequent fast queries can actually have a much
bigger impact on performance than rare slow queries.  To ensure comprehensive
analysis of your query traffic, set the |long_query_time| to **0** so that all
queries are captured.

However, capturing all queries can consume I/O bandwidth and cause the
|slow-query-log| file to quickly grow very large. To limit the amount of
queries captured by the |slow-query-log|, use the *query sampling* feature
available in |percona-server|.

The |log_slow_rate_limit|_ variable defines the fraction of queries captured by
the |slow-query-log|.  A good rule of thumb is to have approximately 100 queries
logged per second.  For example, if your |percona-server| instance processes
10_000 queries per second, you should set ``log_slow_rate_limit`` to ``100`` and
capture every 100th query for the |slow-query-log|.

.. note:: When using query sampling, set |log_slow_rate_type|_ to ``query``
   so that it applies to queries, rather than sessions.

   It is also a good idea to set |log_slow_verbosity|_ to ``full``
   so that maximum amount of information about each captured query
   is stored in the slow query log.

A possible problem with query sampling is that rare slow queries might not get
captured at all.  To avoid this, use the |slow_query_log_always_write_time|_
variable to specify which queries should ignore sampling.  That is, queries with
longer execution time will always be captured by the slow query log.

By default, slow query log settings apply only to new sessions.  If you want to
configure the slow query log during runtime and apply these settings to existing
connections, set the |slow_query_log_use_global_control|_ variable to ``all``.

.. |long_query_time| replace:: ``long_query_time``
.. _long_query_time: http://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_long_query_time

.. |log_slow_rate_limit| replace:: ``log_slow_rate_limit``
.. _log_slow_rate_limit: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_rate_limit

.. |log_slow_rate_type| replace:: ``log_slow_rate_type``
.. _log_slow_rate_type: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_rate_type

.. |log_slow_verbosity| replace:: ``log_slow_verbosity``
.. _log_slow_verbosity: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_verbosity

.. |slow_query_log_always_write_time| replace:: ``slow_query_log_always_write_time``
.. _slow_query_log_always_write_time: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#slow_query_log_always_write_time

.. |slow_query_log_use_global_control| replace:: ``slow_query_log_use_global_control``
.. _slow_query_log_use_global_control: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#slow_query_log_use_global_control

.. _perf-schema-settings:

Configuring Performance Schema
================================================================================

Performance Schema is not as data-rich as the slow query log, but it has all the
critical data and is generally faster to parse.  If you are not running Percona
Server (which supports :ref:`sampling for the slow query log
<slow-log-settings>`), then Performance Schema is the better alternative.

As of MySQL 5.6 (including MariaDB 10.0+ and Percona Server 5.6+), Performance
Schema is enabled by default with no additional configuration required.

If you are running a custom Performance Schema configuration, make sure that the
``statements_digest`` consumer is enabled:

::

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

For more information about using Performance Schema in PMM, see
:ref:`perf-schema`.

.. _pmm/mysql/conf/dashboard:

Settings for Dashboards
=======================

Not all dashboards in :ref:`using-mm` are available by default for all MySQL
variants and configurations.  Some graphs require Percona Server, specialized
plugins, or additional configuration.

Collecting metrics and statistics for graphs increases overhead.  You can keep
collecting and graphing low-overhead metrics all the time, and enable
high-overhead metrics only when troubleshooting problems.

.. _pmm/mysql/conf/dashboard/mysql-innodb-metrics:

MySQL InnoDB Metrics
--------------------

InnoDB metrics provide detailed insight about InnoDB operation.  Although you
can select to capture only specific counters, their overhead is low even when
all them are enabled all the time.  To enable all InnoDB metrics, set the global
|innodb_monitor_enable|_ variable to ``all``::

 mysql> SET GLOBAL innodb_monitor_enable=all

.. |innodb_monitor_enable| replace:: ``innodb_monitor_enable``
.. _innodb_monitor_enable: https://dev.mysql.com/doc/refman/5.7/en/innodb-parameters.html#sysvar_innodb_monitor_enable

.. _pmm/mysql/conf/dashboard/mysql-user-statistics:

MySQL User Statistics
---------------------

User statistics is a feature available in Percona Server and MariaDB.  It
provides information about user activity, individual table and index access.  In
some cases, collecting user statistics can lead to high overhead, so use this
feature sparingly.

To enable user statistics, set the |userstat|_ variable to ``1``.

.. |userstat| replace:: ``userstat``
.. _userstat: https://www.percona.com/doc/percona-server/5.7/diagnostics/user_stats.html#userstat

MySQL Performance Schema
--------------------------------------------------------------------------------

With MySQL version 5.6 or later, Performance Schema instrumentation is enabled
by default.  If certain instruments are not enabled, you will not see the
corresponding graphs in the *Performance Schema* dashboard.  To enable full
instrumentation, set the |performance_schema_instrument|_ option to ``'%=on'``
at startup::

   mysqld --performance-schema-instrument='%=on'

.. note:: This option can cause additional overhead
   and should be used with care.

.. |performance_schema_instrument| replace:: ``--performance_schema_instrument``
.. _performance_schema_instrument: https://dev.mysql.com/doc/refman/5.7/en/performance-schema-options.html#option_mysqld_performance-schema-instrument

.. _pmm/mysql/conf/dashboard/mysql-query-response-time:

MySQL Query Response Time
--------------------------------------------------------------------------------

Query response time distribution is a feature available in Percona Server.  It
provides information about changes in query response time for different groups
of queries, often allowing to spot performance problems before they lead to
serious issues.

.. note:: This feature causes very high overhead,
   especially on systems processing more than 10 000 queries per second.
   Use it only temporarily when troubleshooting problems.

To enable collection of query response time:

1. Install the ``QUERY_RESPONSE_TIME`` plugins::

    mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_AUDIT SONAME 'query_response_time.so';
    mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME SONAME 'query_response_time.so';
    mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_READ SONAME 'query_response_time.so';
    mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_WRITE SONAME 'query_response_time.so';

   For more information, see `this guide
   <https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#installing-the-plugins>`_

2. Set the global |query_response_time_stats|_ varible to ``ON``::

      mysql> SET GLOBAL query_response_time_stats=ON;

.. |query_response_time_stats| replace:: ``query_response_time_stats``
.. _query_response_time_stats: https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#query_response_time_stats

.. include:: .res/replace/name.txt
.. include:: .res/replace/option.txt
