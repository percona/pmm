.. _dashboard.mysql-query-response-time:

|mysql| Query Response Time
================================================================================

This dashboard provides information about query response time distribution. 

.. contents::
   :local:

.. _dashboard.mysql-query-response-time.average:

:ref:`Average Query Response Time <dashboard.mysql-query-response-time.average>`
--------------------------------------------------------------------------------

The Average Query Response Time graph shows information collected using
the Response Time Distribution plugin sourced from table
*INFORMATION_SCHEMA.QUERY_RESPONSE_TIME*. It computes this value across all
queries by taking the sum of seconds divided by the count of seconds.

|view-all-metrics| |this-dashboard|

.. seealso::

   |percona| Server Documentation: QUERY_RESPONSE_TIME table
      https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#QUERY_RESPONSE_TIME

.. _dashboard.mysql-query-response-time.distribution:

:ref:`Query Response Time Distribution <dashboard.mysql-query-response-time.distribution>`
------------------------------------------------------------------------------------------

Shows how many fast, neutral, and slow queries are executed per second.

Query response time counts (operations) are grouped into three buckets:

- 100ms - 1s
- 1s - 10s
- > 10s

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-query-response-time.average.read-write-split:

:ref:`Average Query Response Time (Read/Write Split) <dashboard.mysql-query-response-time.average.read-write-split>`
--------------------------------------------------------------------------------------------------------------------

Available only in |percona| Server for |mysql|, this metric provides
visibility of the split of READ vs WRITE query response time.

|view-all-metrics| |this-dashboard|

.. seealso::

   |percona| Server Documentation: Logging queries in separate READ and WRITE tables
      https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#logging-the-queries-in-separate-read-and-write-tables
   |percona| Server Documentation: QUERY_RESPONSE_TIME_READ
      https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#QUERY_RESPONSE_TIME_READ
   |percona| Server Documentation: QUERY_RESPONSE_TIME_WRITE
      https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#QUERY_RESPONSE_TIME_WRITE

.. _dashboard.mysql-query-response-time.read-distribution:

:ref:`Read Query Response Time Distribution <dashboard.mysql-query-response-time.read-distribution>`
----------------------------------------------------------------------------------------------------

Available only in Percona Server for MySQL, illustrates READ query response time
counts (operations) grouped into three buckets:

- 100ms - 1s
- 1s - 10s
- > 10s

|view-all-metrics| |this-dashboard|

.. seealso::

   |percona| Server Documentation: QUERY_RESPONSE_TIME_READ
      https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#QUERY_RESPONSE_TIME_READ

.. _dashboard.mysql-query-response-time.write-distribution:

:ref:`Write Query Response Time Distribution <dashboard.mysql-query-response-time.write-distribution>`
------------------------------------------------------------------------------------------------------

Available only in Percona Server for MySQL, illustrates WRITE query response
time counts (operations) grouped into three buckets:

- 100ms - 1s
- 1s - 10s
- > 10s

|view-all-metrics| |this-dashboard|

.. seealso::
   
   Configuring |mysql| for |pmm|
      :ref:`pmm.conf-mysql.query-response-time`
   |percona| Server Documentation: QUERY_RESPONSE_TIME_WRITE
      https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#QUERY_RESPONSE_TIME_WRITE

.. |this-dashboard| replace:: :ref:`dashboard.mysql-query-response-time`

.. include:: .res/replace.txt
