.. _conf-mysql-slow-log:
.. _pmm.conf-mysql-slow-log-settings:

#################
Slow Log Settings
#################

If you are running Percona Server for MySQL, a properly configured slow query log will provide the most amount of information with the lowest overhead.  In other cases, use :ref:`Performance Schema <perf-schema>` if it is supported.

*****************************
Configuring the Slow Log File
*****************************

The first and obvious variable to enable is ``slow_query_log`` which controls the global Slow Query on/off status.

Secondly, verify that the log is sent to a FILE instead of a TABLE. This is controlled with the ``log_output`` variable.

By definition, the slow query log is supposed to capture only *slow queries*. These are the queries the execution time of which is above a certain threshold. The threshold is defined by the ``long_query_time`` variable.

In heavily-loaded applications, frequent fast queries can actually have a much bigger impact on performance than rare slow queries.  To ensure comprehensive analysis of your query traffic, set the ``long_query_time`` to **0** so that all queries are captured.

*********
Fine tune
*********

Depending on the amount of traffic, logging could become aggresive and resource consuming. However, Percona Server for MySQL provides a way to throttle the level of intensity of the data capture without compromising information. The most important variable is ``log_slow_rate_limit``, which controls the *query sampling* in Percona Server for MySQL. Details on that variable can be found `here <https://www.percona.com/doc/percona-server/LATEST/diagnostics/slow_extended.html#log_slow_rate_limit>`__.

A possible problem with query sampling is that rare slow queries might not get captured at all.  To avoid this, use the ``slow_query_log_always_write_time`` variable to specify which queries should ignore sampling.  That is, queries with longer execution time will always be captured by the slow query log.

**********************
Slow log file rotation
**********************

PMM will take care of rotating and removing old slow log files, only if you set the ``--size-slow-logs`` variable via pmm-admin as described in :ref:`pmm.ref.pmm-admin`.

When the limit is reached, PMM will remove the previous old slow log file, rename the current file with the sufix ``.old``, and execute the MySQL command ``FLUSH LOGS``. It will only keep one old file. Older files will be deleted on the next iteration.
