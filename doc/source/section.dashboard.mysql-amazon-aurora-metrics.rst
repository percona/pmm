.. _dashboard.mysql-amazon-aurora-metrics:

|mysql| |amazon-aurora| Metrics
================================================================================

This dashboard provides metrics for analyzing |amazon-aurora| instances.

.. contents::
   :local:

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-transaction-commits:

|amazon-aurora| Transaction Commits
--------------------------------------------------------------------------------

This graph shows number of commits which the |amazon-aurora| engine performed as
well as the average commit latency. Graph Latency does not always correlates
with number of commits performed and can quite high in certain situations.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-load:

|amazon-aurora| Load
--------------------------------------------------------------------------------

This graph shows what statements contribute most load on the system as well
as what load corresponds to |amazon-aurora| transaction commit.

- UPDATE load: load in Average Active Sessions per second for UPDATE queries
- SELECT load: load in Average Active Sessions per second for SELECT queries
- DELETE load: load in Average Active Sessions per second for DELETE queries
- INSERT load: load in Average Active Sessions per second for INSERT queries

.. _dashboard.mysql-amazon-aurora-metrics.aurora-memory-used:

Aurora Memory Used
--------------------------------------------------------------------------------

This graph shows how much memory is used by |amazon-aurora| lock manager as well
as amount of memory used by |amazon-aurora| to store Data Dictionary.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-statement-latency:

|amazon-aurora| Statement Latency
--------------------------------------------------------------------------------

This graph shows average latency for most important types of statements. Latency
spikes are often indicative of the instance overload.

- DELETE Latency: average time in milliseconds to execute DELETE queries
- UPDATE Latency: average time in milliseconds to execute UPDATE queries
- SELECT Latency: average time in milliseconds to execute SELECT queries
- INSERT Latency: average time in milliseconds to execute INSERT queries

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-special-command-counters:

Amazon Aurora Special Command Counters
--------------------------------------------------------------------------------

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

Amazon Aurora Problems
--------------------------------------------------------------------------------

This metric shows different kinds of internal |amazon-aurora| |mysql| problems
which should be zero in case of normal operation.

- Reserved mem Exceeded Incidents:
- Missing History on Replica Incidents:
- Thread deadlocks: number of deadlocks per second.

.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt
