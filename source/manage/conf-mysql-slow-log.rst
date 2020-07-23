.. _conf-mysql-slow-log:
.. _pmm.conf-mysql-slow-log-settings:

#################
Slow Log Settings
#################

If you are running Percona Server, a properly configured slow query log will
provide the most amount of information with the lowest overhead.  In other
cases, use :ref:`Performance Schema <perf-schema>` if it is supported.

By definition, the slow query log is supposed to capture only *slow queries*.
These are the queries the execution time of which is above a certain
threshold. The threshold is defined by the ``long_query_time`` variable.

In heavily loaded applications, frequent fast queries can actually have a much
bigger impact on performance than rare slow queries.  To ensure comprehensive
analysis of your query traffic, set the ``long_query_time`` to **0** so that all
queries are captured.

However, capturing all queries can consume I/O bandwidth and cause the
*slow query log* file to quickly grow very large. To limit the amount of
queries captured by the *slow query log*, use the *query sampling* feature
available in Percona Server.

A possible problem with query sampling is that rare slow queries might not get
captured at all.  To avoid this, use the ``slow_query_log_always_write_time``
variable to specify which queries should ignore sampling.  That is, queries with
longer execution time will always be captured by the slow query log.