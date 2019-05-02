.. _dashboard.postgres-overview:

|dbd.postgres-overview|
================================================================================

This dashboard provides basic information about |postgresql| hosts.

.. contents::
   :local:

.. _dashboard.postgres-overview.connected:

`Connected <dashboard.postgres-overview.html#connected>`_
----------------------------------------------------------------------------------------------

Reports whether PMM Server can connect to the |postgresql| instance.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.version:

`Version <dashboard.postgres-overview.html#version>`_
----------------------------------------------------------------------------------

The version of the |postgresql| instance.

|view-all-metrics| |this-dashboard|


.. _dashboard.postgres-overview.shared-buffers:

`Shared Buffers <dashboard.postgres-overview.html#shared-buffers>`_
---------------------------------------------------------------------------------------------------------

Defines the amount of memory the database server uses for shared memory
buffers. Default is ``128MB``. Guidance on tuning is ``25%`` of RAM, but
generally doesn't exceed ``40%``.

|view-all-metrics| |this-dashboard|

.. seealso::

   |postgresql| Server status variables: shared_buffers
      https://www.postgresql.org/docs/current/static/runtime-config-resource.html#GUC-SHARED-BUFFERS

.. _dashboard.postgres-overview.disk-page-buffers:

`Disk-Page Buffers <dashboard.postgres-overview.html#disk-page-buffers>`_
-----------------------------------------------------------------------------------------------------

The setting ``wal_buffers`` defines how much memory is used for caching the
write-ahead log entries. Generally this value is small (``3%`` of 
``shared_buffers`` value), but it may need to be modified for heavily loaded
servers.

|view-all-metrics| |this-dashboard|

.. seealso::

   |postgresql| Server status variables: wal_buffers
      https://www.postgresql.org/docs/current/static/runtime-config-wal.html#GUC-WAL-BUFFERS

   |postgresql| Server status variables: shared_buffers
      https://www.postgresql.org/docs/current/static/runtime-config-resource.html#GUC-SHARED-BUFFERS

.. _dashboard.postgres-overview.memory-size-for-each-sort:

`Memory Size for each Sort <dashboard.postgres-overview.html#memory-size-for-each-sort>`_
-----------------------------------------------------------------------------------------------------------------------

The parameter work_mem defines the amount of memory assigned for internal sort
operations and hash tables before writing to temporary disk files. The default
is ``4MB``.

|view-all-metrics| |this-dashboard|

.. seealso::

   |postgresql| Server status variables: work_mem
      https://www.postgresql.org/docs/current/static/runtime-config-resource.html#GUC-WORK-MEM

.. _dashboard.postgres-overview.disk-cache-size:

`Disk Cache Size <dashboard.postgres-overview.html#disk-cache-size>`_
--------------------------------------------------------------------------------------------------

|postgresql|'s ``effective_cache_size`` variable tunes how much RAM you expect
to be available for disk caching. Generally adding Linux free+cached will give
you a good idea. This value is used by the query planner whether plans will fit
in memory, and when defined too low, can lead to some plans rejecting certain
indexes.

|view-all-metrics| |this-dashboard|

.. seealso::

   |postgresql| Server status variables: effective_cache_size
      https://www.postgresql.org/docs/current/static/runtime-config-query.html#GUC-EFFECTIVE-CACHE-SIZE

.. _dashboard.postgres-overview.autovacuum:

`Autovacuum <dashboard.postgres-overview.html#autovacuum>`_
----------------------------------------------------------------------------------------

Whether autovacuum process is enabled or not. Generally the solution is to
vacuum more often, not less.

|view-all-metrics| |this-dashboard|

.. seealso::

   |postgresql| Server status variables: autovacuum
      https://www.postgresql.org/docs/current/static/routine-vacuuming.html#AUTOVACUUM

.. _dashboard.postgres-overview.connections:

`PostgreSQL Connections <dashboard.postgres-overview.html#connections>`_
-----------------------------------------------------------------------------------------------------

Max Connections
   The maximum number of client connections allowed. Change this value with
   care as there are some memory resources that are allocated on a per-client
   basis, so setting ``max_connections`` higher will generally increase overall
   |postgresql| memory usage.

Connections
   The number of connection attempts (successful or not) to the |postgresql|
   server.

Active Connections
   The number of open connections to the |postgresql| server.

|view-all-metrics| |this-dashboard|

.. seealso::

   |postgresql| Server status variables: max_connections
      https://www.postgresql.org/docs/current/static/runtime-config-connection.html#GUC-MAX-CONNECTIONS

.. _dashboard.postgres-overview.tuples:

`PostgreSQL Tuples <dashboard.postgres-overview.html#tuples>`_
-------------------------------------------------------------------------------------------

Tuples
   The total number of rows processed by |postgresql| server: fetched, returned,
   inserted, updated, and deleted.

Read Tuple Activity
   The number of rows read from the database: as returned so fetched ones.

Tuples Changed per 5min
   The number of rows changed in the last 5 minutes: inserted, updated, and
   deleted ones.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.transactions:

`PostgreSQL Transactions <dashboard.postgres-overview.html#transactions>`_
------------------------------------------------------------------------------------------------------

Transactions
   The total number of transactions that have been either been committed or
   rolled back.

Duration of Transactions
   Maximum duration in seconds any active transaction has been running.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.temp.files:

`Temp Files <dashboard.postgres-overview.html#temp-files>`_
----------------------------------------------------------------------------------------

Number of Temp Files
   The number of temporary files created by queries.

Size of Temp files
   The total amount of data written to temporary files by queries in bytes.

.. note:: All temporary files are taken into account by these two gauges,
   regardless of why the temporary file was created (e.g., sorting or hashing),
   and regardless of the ``log_temp_files`` setting.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.conflicts.and.locks:

`Conflicts and Locks <dashboard.postgres-overview.html#conflicts-and-locks>`_
----------------------------------------------------------------------------------------------------------

Conflicts/Deadlocks
   The number of queries canceled due to conflicts with recovery in the database
   (due to dropped tablespaces, lock timeouts, old snapshots, pinned buffers,
   or deadlocks).

Number of Locks
   The number of deadlocks detected by |postgresql|.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.buffers.and.blocks.operations:

`Buffers and Blocks Operations <dashboard.postgres-overview.html#buffers-and-blocks-operations>`_
------------------------------------------------------------------------------------------------------------------------------

Operations with Blocks
   The time spent reading and writing data file blocks by backends, in
   milliseconds.

.. note:: Capturing read and write time statistics is possible only if
   ``track_io_timing`` setting is enabled. This can be done either in
   configuration file or with the following query executed on the running
   system::

      ALTER SYSTEM SET track_io_timing=ON;
      SELECT pg_reload_conf();

Buffers
   The number of buffers allocated by |postgresql|.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.canceled.queries:

`Canceled Queries <dashboard.postgres-overview.html#canceled-queries>`_
-----------------------------------------------------------------------------------------------------

The number of queries that have been canceled due to dropped tablespaces, lock
timeouts, old snapshots, pinned buffers, and deadlocks.

.. note:: Data shown by this gauge are based on the
   ``pg_stat_database_conflicts`` view.
 
|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.cache.hit.ratio:

`Cache Hit Ratio <dashboard.postgres-overview.html#cache-hit-ratio>`_
--------------------------------------------------------------------------------------------------

The number of times disk blocks were found already in the buffer cache, so that
a read was not necessary.

.. note:: This only includes hits in the |postgresql| buffer cache, not the
   operating system's file system cache.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.checkpoint.stats:

`Checkpoint Stats <dashboard.postgres-overview.html#checkpoint-stats>`_
----------------------------------------------------------------------------------------------------

The total amount of time that has been spent in the portion of checkpoint
processing where files are either written or synchronized to disk,
in milliseconds.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.postgresql.settings:

`PostgreSQL Settings <dashboard.postgres-overview.html#postgresql-settings>`_
----------------------------------------------------------------------------------------------------------

The list of all settings of the |postgresql| server.

|view-all-metrics| |this-dashboard|

.. _dashboard.postgres-overview.system.summary:

`System Summary <dashboard.postgres-overview.html#system-summary>`_
-------------------------------------------------------------------------------------------------

This section contains the following system parameters of the |postgresql|
server: CPU Usage, CPU Saturation and Max Core Usage, Disk I/O Activity, and
Network Traffic.

|view-all-metrics| |this-dashboard|

.. seealso::

   Configuring |postgresql| for Monitoring
      :ref:`pmm.qan.postgres.conf`
   |postgresql| Server status variables: wal_buffers
      https://www.postgresql.org/docs/current/static/runtime-config-wal.html#GUC-WAL-BUFFERS
   |postgresql| Server status variables: shared_buffers
      https://www.postgresql.org/docs/current/static/runtime-config-resource.html#GUC-SHARED-BUFFERS
   |postgresql| Server status variables: work_mem
      https://www.postgresql.org/docs/current/static/runtime-config-resource.html#GUC-WORK-MEM
   |postgresql| Server status variables: effective_cache_size
      https://www.postgresql.org/docs/current/static/runtime-config-query.html#GUC-EFFECTIVE-CACHE-SIZE
   |postgresql| Server status variables: autovacuum
      https://www.postgresql.org/docs/current/static/routine-vacuuming.html#AUTOVACUUM
   |postgresql| Server status variables: max_connections
      https://www.postgresql.org/docs/current/static/runtime-config-connection.html#GUC-MAX-CONNECTIONS

.. |this-dashboard| replace:: :ref:`dashboard.postgres-overview`

.. include:: ../.res/replace.txt

