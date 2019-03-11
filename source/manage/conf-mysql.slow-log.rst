.. _conf-mysql.slow-log:

`Slow Log Settings <pmm.conf-mysql.slow-log-settings>`_
==========================================================================================

If you are running |percona-server|, a properly configured slow query log will
provide the most amount of information with the lowest overhead.  In other
cases, use :ref:`Performance Schema <perf-schema>` if it is supported.

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

.. include:: .res/replace.txt
