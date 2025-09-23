# MongoDB fragmented collections

## Description
This check warns if the storage size is greater than the data size of a collection. This means that the collection is fragmented and needs running a compaction or an initial sync to reclaim disk space.

## Resolution

Fragmented collections are indicated by:
- wiredTiger storage engines with collections that have storage sizes greater than data size AND
- high disk usage```

This often happens if your application deletes many documents in your collections. The WiredTiger storage engine maintains lists of empty records in data files as it deletes documents. This space can be reused by WiredTiger, but will not be returned to the operating system. 

To reclaim disk space, either:

- Run compact on the collections
- Resync the node

**Run compact on the collections:**

The `compact` command rewrites and defragments all data and indexes in a collection. On WiredTiger databases, this command releases unneeded disk space to the operating system.

Below is the syntax -
> db.runCommand({compact: _collection name_})

**NOTE:**

  Starting in MongoDB 6.0.2 (and 5.0.12, and 4.4.17):
  
- A secondary node can replicate while compact is running.
- Reads are permitted.
- All other operations are permitted, except the below ones -
  - db.collection.drop()
  - db.collection.createIndex()
  - db.collection.createIndexes()
  - db.collection.dropIndex()
  - db.collection.dropIndexes()
  - collMod

> **Important:**
> 
>  Always have an up-to-date backup before performing server maintenance such as the compact operation.
> 
> For more details on the `compact` command, see the [MongoDB Documentation](https://www.mongodb.com/docs/manual/reference/command/compact/).

  
**Resync the node:**
  
Instead of running the `compact` command on a collection, you can resync the node in a rolling fashion so that there will be no downtime nor impact on the application.

This approach is the safest way to reclaim the disk space but it can be time consuming if your data size is huge.

For more information on reclaiming disk space, see the [Percona Blog](https://www.percona.com/blog/how-to-reclaim-disk-space-in-percona-server-for-mongodb/).


## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
