.. _dashboard.mysql-amazon-aurora-metrics:

|mysql| |amazon-aurora| Metrics
================================================================================

This dashboard provides metrics for analyzing |amazon-aurora| instances.

.. contents::
   :local:

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-transaction-commits:

`Amazon Aurora Transaction Commits <dashboard.mysql-amazon-aurora-metrics.html#amazon-aurora-transaction-commits>`_
-------------------------------------------------------------------------------------------------------------------

This graph shows number of commits which the |amazon-aurora| engine performed as
well as the average commit latency. Graph Latency does not always correlates
with number of commits performed and can quite high in certain situations.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-load:

`Amazon Aurora Load <dashboard.mysql-amazon-aurora-metrics.html#amazon-aurora-transaction-commits>`_
----------------------------------------------------------------------------------------------------

This graph shows what statements contribute most load on the system as well
as what load corresponds to |amazon-aurora| transaction commit.

- Write Transaction Commit Load: Load in Average Active Sessions per second for
  COMMIT operations
- UPDATE load: load in Average Active Sessions per second for UPDATE queries
- SELECT load: load in Average Active Sessions per second for SELECT queries
- DELETE load: load in Average Active Sessions per second for DELETE queries
- INSERT load: load in Average Active Sessions per second for INSERT queries

.. note: An *active session* is a connection that has submitted work to the
   database engine and is waiting for a response from it. For example, if you
   submit an SQL query to the database engine, the database session is active
   while the database engine is processing that query.

.. _dashboard.mysql-amazon-aurora-metrics.aurora-memory-used:

`Aurora Memory Used <dashboard.mysql-amazon-aurora-metrics.html#aurora-memory-used>`_
-------------------------------------------------------------------------------------

This graph shows how much memory is used by |amazon-aurora| lock manager as well
as amount of memory used by |amazon-aurora| to store Data Dictionary.

- Aurora Lock Manager Memory: the amount of memory used by the Lock Manager,
  the module responsible for handling row lock requests for concurrent
  transactions.

- Aurora Dictionary Memory: the amount of memory used by the Dictionary, the
  space that contains metadata used to keep track of database objects, such as
  tables and indexes.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-statement-latency:

`Amazon Aurora Statement Latency <dashboard.mysql-amazon-aurora-metrics.html#amazon-aurora-statement-latency>`_
---------------------------------------------------------------------------------------------------------------

This graph shows average latency for most important types of statements. Latency
spikes are often indicative of the instance overload.

- DDL Latency: Average time to execute DDL queries
- DELETE Latency: average time to execute DELETE queries
- UPDATE Latency: average time to execute UPDATE queries
- SELECT Latency: average time to execute SELECT queries
- INSERT Latency: average time to execute INSERT queries

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-special-command-counters:

`Amazon Aurora Special Command Counters <dashboard.mysql-amazon-aurora-metrics.html#amazon-aurora-special-command-counters>`_
-----------------------------------------------------------------------------------------------------------------------------

|amazon-aurora| |mysql| allows a number of commands which are not available from
standard |mysql|. This graph shows usage of such commands. Regular
:code:`unit_test` calls can be seen in default |amazon-aurora| install, the rest
will depend on your workload.

show_volume_status
   The number of executions per second of the command |sql.show-volume-status|. The
   |sql.show-volume-status| query returns two server status variables: Disks and
   Nodes. These variables represent the total number of logical blocks of data
   and storage nodes, respectively, for the DB cluster volume.

awslambda
   The number of AWS Lambda calls per second. AWS Lambda is an event-drive,
   serverless computing platform provided by AWS. It is a compute service that
   run codes in response to an event. You can run any kind of code from Aurora
   invoking Lambda from a stored procedure or a trigger.
 
alter_system
   The number of executions per second of the special query ALTER SYSTEM, that
   is a special query to simulate an instance crash, a disk failure, a disk
   congestion or a replica failure. It is a useful query for testing the system.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-problems:

`Amazon Aurora Problems <dashboard.mysql-amazon-aurora-metrics.html#amazon-aurora-problems>`_
---------------------------------------------------------------------------------------------

This metric shows different kinds of internal |amazon-aurora| |mysql| problems
which should be zero in case of normal operation.

- Reserved mem Exceeded Incidents
- Missing History on Replica Incidents
- Thread deadlocks: number of deadlocks per second

.. include:: ../.res/replace.txt

