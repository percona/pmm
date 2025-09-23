# MongoDB sharding - inconsistent indexes across sharded collections

## Description
This check warns if there are inconsistent indexes across shards for sharded collections. Missing or inconsistent indexes across the shards can have a negative impact on performance.


A sharded collection has an inconsistent index if the collection does not have the exact same indexes (including the index options) on each shard that contains chunks for the collection. Although inconsistent indexes should not occur during normal operations, they can occur in some situations. Examples of those situations are the following:

- When a user is creating an index with a unique key constraint and one shard contains a chunk with duplicate documents. In such cases, the create index operation may succeed on the shards without duplicates but fail on the shard with duplicates.

- When a user is creating an index across the shards in a rolling manner - manually building the index individually across the shards one by one. The index could be inconsistent across all shards if the manual process either fails to build the index for an associated shard or incorrectly builds an index with different specifications for one or more shards. Indexes are often created this way when the shards are very large or when there are a high number of shards.

Starting with MongoDB 4.2.6 and 4.4, the config server primarily checks for index inconsistencies across the shards for sharded collections by default.
Additionally, running the **serverStatus** command on the config server primary will return the field **shardedIndexConsistency** to report the number of sharded collections with index inconsistencies.

If **shardedIndexConsistency** reports any index inconsistencies, you can identify the missing index or any inconsistent properties across the collection’s shards by running a specific aggregation pipeline that returns results on inconsistencies. You can find this pipeline script in the [Manage Indexes section of the MongoDB Documentation](https://www.mongodb.com/docs/manual/tutorial/manage-indexes/).

## Resolution

**The inconsistency where an index is missing from the collection on a particular shard(s)** 

Perform either of the following steps: 

- Perform a rolling index build for the collection on the affected shards to make sure indexes are built consistently across all involved shards.

(OR)

- Issue an index build **db.collection.createIndex()** from a mongos instance. The operation only builds the collection's index on the shards missing the index.

**The index properties differ across the shards,**

Drop the incorrect index from the collection on the affected shards and rebuild the index by using the above method.


## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
