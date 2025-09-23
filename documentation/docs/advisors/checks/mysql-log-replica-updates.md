# MySQL server replicating events are not logged

## Description

The **log_replica_updates** value determines if replica updates, received from a source, are logged in the replica's binary log:

* 0 - replica updates are not logged

* 1 - the default value, the replica updates are logged
  
Specifying **--skip-log-bin** which disables binary logging, also disables replica update logging. If enabling binary logging but you need to disable replica update logging, specify **--log-replica-updates=OFF** at replica server startup.

Enabling **log_replica_updates** enables replication servers to be chained. For example, you might want to set up replication servers using this arrangement:

A -> B -> C

Here, A serves as the source for the replica B, and B serves as the source for the replica C. For this to work, B must be both a source and a replica. With binary logging enabled and **log_replica_updates** enabled, updates received from A are logged by B to its binary log, and can be passed on to C.

## Resolution

Change the configuration setting:
Log_replica_updates = 1

!! Warning this parameter is NOT dynamic server restart needed !!

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
