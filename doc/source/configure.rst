.. _configure:

==========================================================
Configuring the Percona Monitoring and Management Platform
==========================================================

PMM is designed in such a way as to provide
full insight into MySQL and MongoDB server performance
without additional configuration.
However, the underlying components
(those developed by Percona and third-party components)
are open-source and provide certain ways
to configure and modify their behavior.
You might even go as far as modifying the source code of some components
to suit your needs,
but most users should not be concerned with anything
beyond the basic configuration that PMM exposes via the web interface.

For more information about configuring MySQL for the best results with PMM,
see :ref:`Configuring MySQL for best results <conf-mysql>`.

This document describes settings exposed in the web interface.

.. contents::
   :depth: 1

Configuring Query Analytics
===========================

Most important Query Analytics settings
can be configured through the QAN web app.
To access the QAN settings, click the gear icon at the top.

At the top of the configuration screen,
use the drop-down menu to select a specific MySQL instance or agent.

MySQL Connection
----------------

This section contains settings of the connection
to the selected MySQL instance.

:ID: Unique identifier of this MySQL instance used by QAN

:Name: Name of this MySQL instance used by QAN.
 By default, it is the same as the host name.
 You can change it to anything that makes sense to you,
 as long as it is unique.

:User: QAN agent's MySQL user name.
 The default is ``pmm``.
 You can change it if you created a different MySQL user for QAN agent.

:Password: QAN agent's MySQL user password.
 The default is ``percona2016``.
 You can change it if you created a different MySQL user for QAN agent.

:Allow old passwords: Enable this if your MySQL server
 uses old password encryption

:Host: Select this and specify the host name of your MySQL server

:Port: If you select to use a host name for connecting to MySQL server,
 you should also specify the port it uses

:Socket: Select this if you want to connect via a socket.
 This is the default.
 It is usually located at :file:`/var/run/mysqld/mysqld.sock`.

:Version: The full MySQL server version

:Agent: Select the agent to use with this connection.
 Click **Test Connection** to make sure
 the agent can communicate with QAN API via this connection.

Query Analytics
---------------

This section contains settings that define how QAN agent collects data.
By default, most of these settings are not set explicitly,
meaning that the agent will use built-in values.
These can vary based on the version of the MySQL instance and agent.

If you want to configure these settings,
choose your agent from the drop-down list
and make sure **Manual** configuration is selected.
For any setting that you want to set manually, click the pin icon.

:Collect interval: This determines how often
 the agent collects and sends data.
 Set a higher value if you do not need frequent updates.

:Send real query examples: Enable this if you want the agent to send
 examples of queries with real data used in the database being monitored.
 If you have sensitive data, disable this to mask values in query examples.

:Collect from: Choose between the slow query log and Performance Schema.
 For more information, see :ref:`perf-schema`.

If you select the slow query log,
there are additional settings you can configure:

:Long query time: A query is added to the slow query log
 only if its execution time exceeds this value.
 Faster queries are not logged.

:Max slow log size: Specify the maximum size
 of the slow query log before it rotates.
 You can specify units, for example, ``1GB``, ``200MB``, and so on.
 Set this to ``0`` if you do not want to rotate the log
 and always write to one file.

:Remove old slow logs: Enable this to remove the old log file
 after it is rotated.

The following settings are available only if you are
monitoring Percona Server and collecting from the slow query log:

:Slow log verbosity: Select how verbose you want the slow query log to be.
 ``Full`` means that all query details are logged,
 ``Standard`` means only the important metrics are logged,
 and ``Minimal`` is only for the most basic query metrics.

:Rate limit: Use this to limit the amount of queries logged.
 It is usually not necessary to log every slow query.
 For example, if there are thousands of slow queries,
 you can set this to ``20`` to log every 20th slow query.
 By default, it is set to ``100`` (log every 100th slow query).
 If you are not getting enough slow queries for good analysis,
 set ``0`` or ``1`` to disable rate-limiting
 and log every slow query encountered.

:Log slow admin statements: Enable this
 to include slow administrative statements to the slow query log.
 Administrative statements are ``ALTER TABLE``,
 ``ANALYZE TABLE``, ``CHECK TABLE``, ``CREATE INDEX``,
 ``DROP INDEX``, ``OPTIMIZE TABLE``, and ``REPAIR TABLE``.
 For more information,
 see the |log_slow_admin_statements|_ replication slave variable reference.

 .. |log_slow_admin_statements| replace:: ``log_slow_admin_statements``
 .. _log_slow_admin_statements: http://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_log_slow_admin_statements

:Log slow slave statements: Enable this
 to include slow queries executed on the slave.
 It applies to ``START SLAVE`` statements.
 For more information,
 see the |log_slow_slave_statements|_ MySQL system variable reference.

 .. |log_slow_slave_statements| replace:: ``log_slow_slave_statements``
 .. _log_slow_slave_statements: http://dev.mysql.com/doc/refman/5.7/en/replication-options-slave.html#sysvar_log_slow_slave_statements

Agent Status and Log
--------------------

QAN agent regularly sends its status and log messages to QAN API.
This is a handy way to remotely troubleshoot the agent
without logging in to the MySQL host where the agent is running.

To access the agent's status and online log,
select the agent from the drop-down list
at the top of the QAN settings page.
You can specify a custom name for the agent
and see the unique identifier that QAN uses for this agent.

The **Status** section contains latest status
and configuration parameter values fetched from the agent.
You can see which services are running and which are idle,
when was the last data sent, and other internal information.

The **Log** section contains a list of log messages
that the agent sent to QAN API.

Configuring Metrics Monitor
===========================

There are standard Grafana settings
that you can access using the **Manage Dashboards** gear icon
in the header toolbar.

Prometheus web interface can be accessed by adding ``/prometheus/``
to the PMM server address.

Consul web interface can be accessed by adding ``/consul/``
to the PMM server address.

.. note:: It is not recommended to configure any settings
   in the Consul web interface, because it can crash PMM.
   Access to the Consul web interface is provided only for visibility,
   and possibly some low-level configuration suggestions from experts.

.. rubric:: References

.. target-notes::
