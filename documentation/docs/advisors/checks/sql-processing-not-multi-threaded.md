# Replica SQL processing not multi-threaded

## Description

From MySQL 8.0.26, use **replica_parallel_workers** in place of **slave_parallel_workers**, which is deprecated from that release. 
In releases before MySQL 8.0.26, use **slave_parallel_workers**.

**replica_parallel_workers** enables multithreading on the replica and sets the number of applier threads for executing replication transactions in parallel. When the value is a number greater than 0, the replica is a multithreaded replica with the specified number of applier threads, plus a coordinator thread to manage them. If you are using multiple replication channels, each channel has this number of threads.

Before MySQL 8.0.27, the default for this system variable is 0, so replicas are not multithreaded by default. From MySQL 8.0.27, the default is 4, so replicas are multithreaded by default.

Retrying of transactions is supported when multithreading is enabled on a replica. When **replica_preserve_commit_order=ON** or **slave_preserve_commit_order=ON **is set, transactions on a replica are externalized on the replica in the same order as they appear in the replica's relay log. 

The way in which transactions are distributed among applier threads is configured by **replica_parallel_type** (from MySQL 8.0.26) or **slave_parallel_type** (before MySQL 8.0.26). From MySQL 8.0.27, these system variables also have appropriate defaults for multithreading.

To disable parallel execution, set **replica_parallel_workers** to 0, which gives the replica a single applier thread and no coordinator thread. 

With this setting, the **replica_parallel_type **or **slave_parallel_type** 
and **replica_preserve_commit_order** or **slave_preserve_commit_order** system variables have no effect and are ignored. 

From MySQL 8.0.27, if parallel execution is disabled when the **CHANGE REPLICATION SOURCE TO** option GTID_ONLY is enabled on a replica, the replica actually uses one parallel worker to take advantage of the method for retrying transactions without accessing the file positions. 

With one parallel worker, the **replica_preserve_commit_order** or **slave_preserve_commit_order **system variable also has no effect.

## Resolution

Adopt a more appropriate value like replica_parallel_workers=4 (default from MySQL 8.0.26) and execute: STOP REPLICA; START REPLICA.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
