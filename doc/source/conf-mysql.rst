.. _conf-mysql:

=======================================================
Configuring MySQL for Percona Monitoring and Management
=======================================================

PMM supports all commonly used variants of MySQL,
including Percona Server, MariaDB, and Amazon RDS.
Although it will work with default settings,
you can additionally configure MySQL
to enhance monitoring information gathered by PMM.

We recommend using PMM with default settings in most cases,
but certain MySQL configuration is possible
via the Query Analytics web app interface 
(for more information, see :ref:`Configuring PMM <configure>`).

Many more variables and options can be configured
directly on the MySQL server for special cases to improve monitoring.
This document provides some advanced recommendations
for configuring MySQL to get the most out of PMM.

.. contents::
   :local:
   :depth: 1

.. _slow-log-settings:

Settings for the Slow Query Log
===============================

If you are running Percona Server, a properly configured slow query log
will provide the most amount of information with the lowest overhead.
In all other cases, use :ref:`Performance Schema <perf-schema-settings>`.

By definition, the slow query log is supposed to capture only *slow queries*.
That is, queries with execution time above a certain threshold,
which is defined by the |long_query_time|_ variable.

In heavily loaded applications, frequent fast queries can actually have
a much bigger impact on performance than rare slow queries.
To ensure comprehensive analysis of your query traffic,
set the ``long_query_time`` to ``0`` so that all queries are captured.

However, capturing all queries can consume I/O bandwidth
and cause the slow query log file to quickly grow very large.
To limit the amount of queries logged by QAN,
use the *query sampling* feature available in Percona Server.

The |log_slow_rate_limit|_ variable defines the fraction of queries
captured by the slow query log.
A good rule of thumb is to have approximately 100 queries logged per second.
For example,
if your Percona Server instance processes 10 000 queries per second,
you should set ``log_slow_rate_limit`` to ``100``
and capture every 100th query for the slow query log.

.. note:: When using query sampling, set |log_slow_rate_type|_ to ``query``
   so that it applies to queries, rather than sessions.

   It is also a good idea to set |log_slow_verbosity|_ to ``full``
   so that maximum amount of information about each captured query
   is stored in the slow query log.

A possible problem with query sampling is that rare slow queries
might not get captured at all.
To avoid this, use the |slow_query_log_always_write_time|_ variable
to specify which queries should ignore sampling.
That is, queries with longer execution time
will always be captured by the slow query log.

By default, the slow query log settings apply only to new sessions.
If you want to configure the slow query log during runtime
and apply these settings to existing connections,
set the |slow_query_log_use_global_control|_ variable to ``all``.

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

Settings for Performance Schema
===============================

Performance Schema is not as data-rich as the slow query log,
but it has all the critical data and is generally faster to parse.
If you are not running Percona Server
with a :ref:`thoroughly tuned slow query log <slow-log-settings>`,
then Performance Schema is the better alternative.

As of MySQL 5.6 (including Percona Server 5.6 and later versions),
Performance Schema is enabled by default
with no additional configuration required.

If you are running a custom Performance Schema configuration,
make sure that the ``statements_digest`` consumer is enabled:

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

For more information about using Performance Schema in PMM,
see :ref:`perf-schema`.

Special Dashboards
==================

Not all dashboards in :ref:`using-mm` are available by default
for all MySQL variants and configurations.
Some graphs require Percona Server, specialized plugins,
or additional configuration.

Collecting metrics and statistics for graphs increases overhead.
You can keep collecting and graphing low-overhead metrics all the time,
and enable high-overhead metrics only when troubleshooting problems.

MySQL InnoDB Metrics
--------------------

InnoDB metrics provide detailed insight about InnoDB operation.
Although you can select to capture only specific counters,
they introduce minimal overhead even when enabled all the time.
To enable all InnoDB metrics,
set the global |innodb_monitor_enable|_ variable to ``all``::

 mysql> SET GLOBAL innodb_monitor_enable=all

.. |innodb_monitor_enable| replace:: ``innodb_monitor_enable``
.. _innodb_monitor_enable: https://dev.mysql.com/doc/refman/5.7/en/innodb-parameters.html#sysvar_innodb_monitor_enable

MySQL User Statistics
---------------------

User statistics is a feature available in Percona Server and MariaDB.
It provides information about user activity, individual table and index access.
In some cases, collecting user statistics can lead to high overhead,
so use this feature sparingly.

To enable user statistics, set the |userstat|_ variable to ``1``.

.. |userstat| replace:: ``userstat``
.. _userstat: https://www.percona.com/doc/percona-server/5.7/diagnostics/user_stats.html#userstat

MySQL Performance Schema
------------------------

With MySQL version 5.6 or later,
Performance Schema instrumentation is enabled by default.
If certain instruments are not enabled,
you will not see the corresponding graphs in the Performance Schema dashboard.
To enable full instrumentation,
set the |performance_schema_instrument|_ option to ``'%=on'`` at startup::

   mysqld --performance-schema-instrument='%=on'

.. note:: This option can cause additional overhead
   and should be used with care.

.. |performance_schema_instrument| replace:: ``--performance_schema_instrument``
.. _performance_schema_instrument: https://dev.mysql.com/doc/refman/5.7/en/performance-schema-options.html#option_mysqld_performance-schema-instrument

MySQL Query Response Time
-------------------------

Query response time distribution is a feature available in Percona Server.
It provides information about changes in query response time
for different groups of queries,
often allowing to spot performance problems
before they lead to serious issues.

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

Sample Configuration
====================

Considering all the recommendations,
you can try using some or all of the following MySQL configuration settings,
if you are running Percona Server::

   # Consider all queries regardless of execution time
   long_query_time=0
   
   # Capture every 100th query
   log_slow_rate_limit=100
   
   # Sample by queries, rather than session
   log_slow_rate_type=query
   
   # Store maximum information about each captured query
   log_slow_verbosity=full
   
   # Always capture queries with execution time over 1 second, ignoring sampling
   slow_query_log_always_write_time=1
   
   # Global slow query log settings apply to existing connections
   slow_query_log_use_global_control=all
   
   # Enable all InnoDB metrics (low overhead)
   innodb_monitor_enable=all
   
   # Enable user statistics (can lead to high overhead in some cases)
   userstat=1

   # Enable collection of query response time (very high overhead!)
   query_response_time_stats=ON

.. rubric:: References

.. target-notes::
