.. _mm-dashboards:

==========================
Metrics Monitor Dashboards
==========================

This section contains a reference of dashboards
available in Metrics Monitor.

.. contents::
   :local:

MySQL Overview
==============

This dashboard provides basic information about MySQL hosts.

.. list-table::
   :header-rows: 1
   :widths: 30 20 50

   * - Name
     - Importance
     - Description

   * - MySQL Uptime
     - INFO
     - The amount of time since the MySQL server process was started.

   * - Current QPS
     - IMPORTANT
     - The number of queries executed by the server during the last second,
       *including those executed within stored programs*.

       This variable does not include the following commands:

       * ``COM_PING``
       * ``COM_STATISTICS``

   * - InnoDB Buffer Pool Size
     - IMPORTANT
     - Absolute value of the InnoDB buffer pool
       used for caching data and indexes in memory.
       This should be big enough to store the working set
       and never exceed the available memory on the database host.

   * - Buffer Pool Size of Total RAM
     - IMPORTANT
     - Ratio between InnoDB buffer pool size and total memory.
       In most cases, the InnoDB buffer pool should be between 60% and 90%
       of available memory on a dedicated database host,
       but it depends on many factors.

   * - MySQL Connections
     - IMPORTANT
     - **Max Connections** is the maximum permitted number
       of simultaneous client connections.
       This is the value of the ``max_connections`` variable.

       **Max Used Connections** is the maximum number of connections
       that have been in use simultaneously since the server was started.

       **Connections** is the number of connection attempts
       (successful or not) to the MySQL server.

   * - MySQL Active Threads
     - INFO
     - **Threads Connected** is the number of open connections.

       **Threads Running** is the number of threads not sleeping.

   * - MySQL Questions
     - INFO
     - The number of queries sent to the server by clients,
       *excluding those executed within stored programs*.

       This variable does not count the following commands:

       * ``COM_PING``
       * ``COM_STATISTICS``
       * ``COM_STMT_PREPARE``
       * ``COM_STMT_CLOSE``
       * ``COM_STMT_RESET``

MySQL Query Response Time
=========================

This dashboard provides information about query response time distribution.

.. list-table::
   :header-rows: 1
   :widths: 30 20 50

   * - Name
     - Importance
     - Description

   * - Average Query Response Time
     - INFO
     - Average query response time is calculated
       as the total execution time of queries
       divided by the number of queries.

   * - Query Response Time Distribution
     - INFO
     - Shows how many fast, neutral, and slow queries are executed per second.

   * - Average Query Response Time
       (Read/Write Split)
     - INFO
     - Compare read and write query response time.

   * - Read Query Response Time Distribution
     - INFO
     - Shows how many fast, neutral, and slow read queries
       are executed per second.

   * - Write Query Response Time Distribution
     - INFO
     - Shows how many fast, neutral, and slow write queries
       are executed per second.

MongoDB Overview
================

This dashboard provides basic information about MongoDB instances.

.. list-table::
   :header-rows: 1
   :widths: 30 20 50

   * - Name
     - Importance
     - Description

   * - Command Operations
     - INFO
     - Shows how many times a command is executed per second on average
       during the selected interval.

       Look for peaks and drops and correlate them with other graphs.

   * - Connections
     - IMPORTANT
     - Keep in mind the hard limit on the maximum number of connections
       set by your distribution.

       Anything over 5,000 should be a concern,
       because the application may not close connections correctly.

   * - Cursors
     - INFO
     - Helps identify why connections are increasing.
       Shows active cursors compared to cursors being automatically killed
       after 10 minutes due to an application not closing the connection.

   * - Document Operations
     - INFO
     - When used in combination with **Command Operations**,
       this graph can help identify *write aplification*.
       For example, when one ``insert`` or ``update`` command
       actually inserts or updates hundreds, thousands,
       or even millions of documents.

   * - Queued Operations
     - CRITICAL
     - Any number of queued operations for long periods of time
       is an indication of possible issues.
       Find the cause and fix it before requests get stuck in the queue.

   * - getLastError Write Time

       getLastError Write Operations
     - INFO
     - This is useful for write-heavy workloads
       to understand how long it takes to verify writes
       and how many concurrent writes are occurring.

   * - Asserts
     - INFO
     - Asserts are not important by themselves,
       but you can correlate spikes with other graphs.

   * - Memory Faults
     - CRITICAL
     - Memory faults indicate that requests are processed from disk
       either because an index is missing
       or there is not enough memory for the data set.
       Consider increasing memory or sharding out.

MongoDB ReplSet
===============

This dashboard provides information about replica sets and their members.

.. list-table::
   :header-rows: 1
   :widths: 30 20 50

   * - Name
     - Importance
     - Description

   * - ReplSet State
     - INFO
     - Shows the role of the selected member instance
       (PRIMARY or SECONDARY)

   * - ReplSet Members
     - INFO
     - Shows the number of members in the replica set

   * - ReplSet Last Election
     - INFO
     - Shows how long ago the last election occurred

   * - ReplSet Lag
     - INFO
     - Shows the current replication lag for the selected member

   * - Storage Engine
     - INFO
     - Shows the storage engine used on the instance

   * - Oplog Insert Time
     - INFO
     - Shows how long it takes to write to the oplog.
       Without it the write will not be successful.

       This is more useful in mixed replica sets
       (where instances run different storage engines).

   * - Oplog Recovery Window
     - CRITICAL
     - Shows the time range in the oplog
       and the oldest backed up operation.

       For example, if you take backups every 24 hours,
       each one should contain at least 36 hours of backed up operations,
       giving you 12 hours of restore window.

   * - Replication Lag
     - INFO
     - Shows the delay between an operation occurring on the primary
       and that same operation getting applied on the selected member

   * - Elections
     - INFO
     - Elections happen when a primary becomes unavailable.
       Look at this graph over longer periods (weeks or months)
       to determine patterns and correlate elections with other events.

   * - Member State Uptime
     - INFO
     - Shows how long various members were in PRIMARY and SECONDARY roles

   * - Max Heartbeat Time
     - IMPORTANT
     - Shows the heartbeat return times sent by the current member
       to other members in the replica set.

       Long heartbeat times can indicate network issues
       or that the server is too busy.

   * - Max Member Ping Time
     - INFO
     - This can show a correlation with the replication lag value


