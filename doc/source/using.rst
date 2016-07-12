.. _using:

====================================================
Using the Percona Monitoring and Management Platform
====================================================

You can access the PMM web interface using the IP address of the host
where the *PMM Server* container is running.
For example, http://192.168.100.1.

The landing page has links to corresponding PMM tools:

.. contents::
   :local:
   :depth: 1

These tools provide comprehensive insight
into the performance of a MySQL host.

.. _using-qan:

Query Analytics
===============

The *Query Analytics* tool enables database administrators
and application developers to analyze MySQL queries over periods of time
and find performance problems.
Query Analytics helps you optimize database performance
by making sure that queries are executed as expected
and within the shortest time possible.
In case of problems, you can see which queries may be the cause
and get detailed metrics for them.

The following image shows the *Query Analytics* app.

.. image:: images/query-analytics.png
   :width: 640

The summary table contains top 10 queries ranked by **%GTT**
(percent of grand total time),
which is the percentage of time
that the MySQL server spent executing a specific query,
compared to the total time it spent executing all queries
during the selected period of time.

You can select the period of time at the top,
by selecting a predefined interval
(last hour, 3 hours, 6 hours, 12 hours, last day, or 5 days),
or select a specific inteval using the calendar icon.

If you have multiple MySQL hosts with *PMM Client* installed,
you can switch between those hosts using the drop-down list at the top.

To configure the QAN agent running on a MySQL host with *PMM Client*,
click the gear icon at the top.
For more information, see :ref:`configure`.

Query Details
-------------

You can get details for a query if you click it in the summary table.
The details contain all metrics specific to that particular query,
such as, bytes sent, lock time, rows sent, and so on.
You can see when the query was first and last seen,
get an example of the query, as well as its fingerprint.

The details section enables you to run ``EXPLAIN`` on the selected query
directly from the PMM web interface (simply specify the database).

.. image:: images/qan-realtime-explain.png
   :width: 640

At the bottom, you can run Table Info for the selected query.
This enables you to get ``SHOW CREATE TABLE``, ``SHOW INDEX``,
and ``SHOW TABLE STATUS`` for each table used by the query
directly from the PMM web interface.

.. image:: images/qan-create-table.png
   :width: 640

.. _perf-schema:

Performance Schema
------------------

The default source of query data for PMM is the slow query log.
It is available in MySQL 5.1 and later versions.
Starting from MySQL 5.6 (including Percona Server 5.6 and later),
you can select to parse query data from the Performance Schema.

Performance Schema is not as data-rich as the slow query log,
but it has all the critical data and is generally faster to parse.
It is recommended to use the slow query log when running Percona Server
and the QAN agent is properly configured to avoid overhead.
Otherwise, it is likely that using Performance Schema will
provide better results.

For more information about configuring QAN agent, see :ref:`configure`.

**To use Performance Schema:**

1. Enable it on the server by starting ``mysqld``
   with the ``performance_schema`` variable set to ``ON``.
   For example, use the following lines in :file:`my.cnf`:

   .. code-block:: none

      [mysql]
      performance_schema=ON

   .. note:: Performance Schema instrumentation is enabled by default
      on MySQL 5.6.6 and later versions.

2. Configure QAN agent to collect data from Performance Schema:

   a. In the Query Analytics web UI, click the gear button at the top.
   b. Under **Query Analytics**, select **Performance Schema**
      in the **Collect from** drop-down list.
   c. Click **Apply** to save changes.

.. _using-mm:

Metrics Monitor
===============

The *Metrics Monitor* tool provides a historical view of metrics
that are critical to a database server.
Time-based graphs are separated into dashboards by themes:
some are related to MySQL or MongoDB, others provide general system metrics.

To access the dashboards, provide default user credentials:

* User: ``admin``
* Password: ``admin``

On the Home screen, select a dashboard
from the list of available Percona Dashboards.
For example, the following image shows the **MySQL Overview** dashboard:

.. image:: images/metrics-monitor.png 
   :width: 640

