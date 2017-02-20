.. _mm-dashboards:

==========================
Metrics Monitor Dashboards
==========================

This section contains a reference of dashboards
provided in Metrics Monitor.

MongoDB Dashboards
==================

MongoDB Overview
----------------

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
---------------

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


