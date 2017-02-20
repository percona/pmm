.. _using:

====================================================
Using the Percona Monitoring and Management Platform
====================================================

You can access the PMM web interface using the IP address of the host
where *PMM Server* is running.
For example, if *PMM Server* is running on a host with IP 192.168.100.1,
access the following address with your web browser: ``http://192.168.100.1``.

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
Starting from MySQL 5.6.6, Performance Schema is enabled by default.

Performance Schema is not as data-rich as the slow query log,
but it has all the critical data and is generally faster to parse.
If you are running Percona Server,
a :ref:`properly configured slow query log <slow-log-settings>`
will provide the most amount of information with the lowest overhead.
Otherwise, using :ref:`Performance Schema <perf-schema-settings>`
will likely provide better results.

**To use Performance Schema:**

1. Make sure that the ``performance_schema`` variable is set to ``ON``:

   .. code-block:: sql

      mysql> SHOW VARIABLES LIKE 'performance_schema';
      +--------------------+-------+
      | Variable_name      | Value |
      +--------------------+-------+
      | performance_schema | ON    |
      +--------------------+-------+

   If not, add the the following lines to :file:`my.cnf` and restart MySQL:

   .. code-block:: sql

      [mysql]
      performance_schema=ON

   .. note:: Performance Schema instrumentation is enabled by default
      in MySQL 5.6.6 and later versions.
      It is not available at all in MySQL versions prior to 5.6.

2. Configure QAN agent to collect data from Performance Schema:

   If the instance is already running:

   a. In the Query Analytics web UI, click the gear button at the top.
   b. Under **Query Analytics**, select **Performance Schema**
      in the **Collect from** drop-down list.
   c. Click **Apply** to save changes.

   If you are adding a new monitoring instance with the ``pmm-admin`` tool,
   use the ``--query-source perfschema`` option.
   For example:

   .. code-block:: bash

      sudo pmm-admin add mysql --user root --password root --create-user --query-source perfschema

For more information, run ``pmm-admin add mysql --help``.

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

.. _orchestrator:

Orchestrator
============

.. note:: Orchestrator was included into PMM for experimental purposes.
   It is a standalone tool, not integrated with PMM
   other than that you can access it from the landing page.

Orchestrator is a MySQL replication topology management and visualization tool.
You can access it using the ``/orchestrator`` URL after *PMM Server* address.

To use it, create a MySQL user for Orchestrator on all managed instances::

 GRANT SUPER, PROCESS, REPLICATION SLAVE, RELOAD ON *.* TO 'orc_client_user'@'%' IDENTIFIED BY 'orc_client_password’;

.. note:: The credentials in the previous example are default.
   If you use a different user name or password,
   you have to pass them when
   :ref:`running PMM Server <run-server>`
   using the following options::

    -e ORCHESTRATOR_USER=name -e ORCHESTRATOR_PASSWORD=pass

Then you can use the **Discover** page in the Orchestrator web interface
to add the instances to the topology.
