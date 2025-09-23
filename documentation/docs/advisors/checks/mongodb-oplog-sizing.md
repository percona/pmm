# MongoDB Oplog Sizing

## Description
This check warns if the oplog window is below 24 hours, and recommends an oplog size based on your instance.

## Resolution

**Replica set oplog**

The oplog (operations log) is a special capped collection that keeps a rolling record of all operations that modify the data stored in your databases.

MongoDB applies database operations on the primary and then records the operations on the primary's oplog. The secondary members then copy and apply these operations in an asynchronous process. All replica set members contain a copy of the oplog, in the local.oplog.rs collection, which allows them to maintain the current state of the database.

**Oplog window**

Oplog entries are time-stamped. The oplog window refers to the amount of time that the oplog can store Write operations before it reaches its maximum size and starts overwriting old entries. The length of the oplog window depends on the rate of Write operations in the system and the size of the oplog.

**Calculate oplog size**

It's important to ensure that the oplog size is large enough to accommodate the expected rate of write operations and provide an adequate oplog window. If the oplog window is too short, it can lead to data loss or replication lag in the replica set.

Usually, we recommend keeping the oplog window of 24-48 hrs. If it goes below the recommended value and stays there for quite some time, then you might need to resize the oplog. 

The default oplog size for the WiredTiger storage engine is 5% of physical memory in which the lower bound is 990 MB and the upper bound is 50 GB. To determine the oplog size, calculate it using the below method below: 

For 24 hour window -
> oplog_size = Oplog rate (GB/Hr) * 24

For 48 hour window -
> oplog_size = Oplog rate (GB/Hr) * 48

**Set the oplog size**

You can explicitly change the oplog size by using either of the following methods:

1. Set the size in the mongodb configuration file -

> replication:
>    oplogSizeMB: <int>

2. Set the size using mongo shell -

> db.adminCommand({replSetResizeOplog: 1, size: 2048})


For more information on oplog, see the [MongoDB documentation](https://www.mongodb.com/docs/manual/core/replica-set-oplog/).


## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
