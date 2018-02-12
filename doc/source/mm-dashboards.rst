.. _mm-dashboards:

================================================================================
Metrics Monitor Dashboards
================================================================================

This section contains a reference of dashboards
available in Metrics Monitor.

.. contents::
   :local:

MySQL Overview
================================================================================

This dashboard provides basic information about MySQL hosts.

.. include:: .res/table/list-table.org
   :start-after: +dashboard.mysql-overview+
   :end-before: #+end-block

.. seealso::

   |mysql| Documentation: |innodb| buffer pool
      https://dev.mysql.com/doc/refman/5.7/en/innodb-buffer-pool.html
   |mysql| Server System Variables: key_buffer_size
      https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_key_buffer_size
   |percona-server| Documentation: Running |tokudb| in Production
      https://www.percona.com/doc/percona-server/LATEST/tokudb/tokudb_quickstart.html#considerations-to-run-tokudb-in-production
   Blog post: Adaptive Hash Index in |innodb|
      https://www.percona.com/blog/2016/04/12/is-adaptive-hash-index-in-innodb-right-for-my-workload/

MySQL Query Response Time
================================================================================

This dashboard provides information about query response time distribution.

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Name
     - Description

   * - Average Query Response Time
     - Average query response time is calculated
       as the total execution time of queries
       divided by the number of queries.

   * - Query Response Time Distribution
     - Shows how many fast, neutral, and slow queries are executed per second.

   * - Average Query Response Time
       (Read/Write Split)
     - Compare read and write query response time.

   * - Read Query Response Time Distribution
     - Shows how many fast, neutral, and slow read queries
       are executed per second.

   * - Write Query Response Time Distribution
     - Shows how many fast, neutral, and slow write queries
       are executed per second.

MongoDB Overview
================================================================================

This dashboard provides basic information about MongoDB instances.

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Name
     - Description

   * - Command Operations
     - Shows how many times a command is executed per second on average
       during the selected interval.

       Look for peaks and drops and correlate them with other graphs.

   * - Connections
     - Keep in mind the hard limit on the maximum number of connections
       set by your distribution.

       Anything over 5,000 should be a concern,
       because the application may not close connections correctly.

   * - Cursors
     - Helps identify why connections are increasing.
       Shows active cursors compared to cursors being automatically killed
       after 10 minutes due to an application not closing the connection.

   * - Document Operations
     - When used in combination with **Command Operations**,
       this graph can help identify *write aplification*.
       For example, when one ``insert`` or ``update`` command
       actually inserts or updates hundreds, thousands,
       or even millions of documents.

   * - Queued Operations
     - Any number of queued operations for long periods of time
       is an indication of possible issues.
       Find the cause and fix it before requests get stuck in the queue.

   * - getLastError Write Time

       getLastError Write Operations
     - This is useful for write-heavy workloads
       to understand how long it takes to verify writes
       and how many concurrent writes are occurring.

   * - Asserts
     - Asserts are not important by themselves,
       but you can correlate spikes with other graphs.

   * - Memory Faults
     - Memory faults indicate that requests are processed from disk
       either because an index is missing
       or there is not enough memory for the data set.
       Consider increasing memory or sharding out.

MongoDB ReplSet
================================================================================

This dashboard provides information about replica sets and their members.

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Name
     - Description

   * - ReplSet State
     - Shows the role of the selected member instance
       (PRIMARY or SECONDARY)
   * - ReplSet Members
     - Shows the number of members in the replica set
   * - ReplSet Last Election
     - Shows how long ago the last election occurred
   * - ReplSet Lag
     - Shows the current replication lag for the selected member
   * - Storage Engine
     - Shows the storage engine used on the instance
   * - Oplog Insert Time
     - Shows how long it takes to write to the oplog.
       Without it the write will not be successful.

       This is more useful in mixed replica sets
       (where instances run different storage engines).
   * - Oplog Recovery Window
     - Shows the time range in the oplog
       and the oldest backed up operation.

       For example, if you take backups every 24 hours,
       each one should contain at least 36 hours of backed up operations,
       giving you 12 hours of restore window.
   * - Replication Lag
     - Shows the delay between an operation occurring on the primary
       and that same operation getting applied on the selected member
   * - Elections
     - Elections happen when a primary becomes unavailable.
       Look at this graph over longer periods (weeks or months)
       to determine patterns and correlate elections with other events.
   * - Member State Uptime
     - Shows how long various members were in PRIMARY and SECONDARY roles
   * - Max Heartbeat Time
     - Shows the heartbeat return times sent by the current member
       to other members in the replica set.

       Long heartbeat times can indicate network issues
       or that the server is too busy.
   * - Max Member Ping Time
     - This can show a correlation with the replication lag value

Cross Server Graphs
================================================================================

.. include:: .res/table/list-table.org
   :start-after: +dashboard.cross-server-graphs+
   :end-before: #+end-block


.. include:: .res/replace/program.txt
.. include:: .res/replace/name.txt
