# MySQL Amazon Aurora Metrics

This dashboard provides metrics for analyzing Amazon Aurora instances.

## Amazon Aurora Transaction Commits

This graph shows number of commits which the Amazon Aurora engine performed as well as the average commit latency. Graph Latency does not always correlates with number of commits performed and can quite high in certain situations.

## Amazon Aurora Load

This graph shows what statements contribute most load on the system as well as what load corresponds to Amazon Aurora transaction commit.

* Write Transaction Commit Load: Load in Average Active Sessions per second for COMMIT operations
* UPDATE load: load in Average Active Sessions per second for UPDATE queries
* SELECT load: load in Average Active Sessions per second for SELECT queries
* DELETE load: load in Average Active Sessions per second for DELETE queries
* INSERT load: load in Average Active Sessions per second for INSERT queries

## Aurora Memory Used

This graph shows how much memory is used by Amazon Aurora lock manager as well as amount of memory used by Amazon Aurora to store Data Dictionary.

* Aurora Lock Manager Memory: the amount of memory used by the Lock Manager, the module responsible for handling row lock requests for concurrent transactions.
* Aurora Dictionary Memory: the amount of memory used by the Dictionary, the space that contains metadata used to keep track of database objects, such as tables and indexes.

## Amazon Aurora Statement Latency

This graph shows average latency for most important types of statements. Latency spikes are often indicative of the instance overload.

* DDL Latency: Average time to execute DDL queries
* DELETE Latency: average time to execute DELETE queries
* UPDATE Latency: average time to execute UPDATE queries
* SELECT Latency: average time to execute SELECT queries
* INSERT Latency: average time to execute INSERT queries

## Amazon Aurora Special Command Counters

Amazon Aurora MySQL allows a number of commands which are not available from standard MySQL. This graph shows usage of such commands. Regular `unit_test` calls can be seen in default Amazon Aurora install, the rest will depend on your workload.

show_volume_status
: The number of executions per second of the command **SHOW VOLUME STATUS**. The **SHOW VOLUME STATUS** query returns two server status variables: Disks and Nodes. These variables represent the total number of logical blocks of data and storage nodes, respectively, for the DB cluster volume.

awslambda
: The number of AWS Lambda calls per second. AWS Lambda is an event-drive, serverless computing platform provided by AWS. It is a compute service that run codes in response to an event. You can run any kind of code from Aurora invoking Lambda from a stored procedure or a trigger.

alter_system
: The number of executions per second of the special query ALTER SYSTEM, that is a special query to simulate an instance crash, a disk failure, a disk congestion or a replica failure. It is a useful query for testing the system.

## Amazon Aurora Problems

This metric shows different kinds of internal Amazon Aurora MySQL problems which should be zero in case of normal operation.

* Reserved mem Exceeded Incidents
* Missing History on Replica Incidents
* Thread deadlocks: number of deadlocks per second
