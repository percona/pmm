# MongoDB Unused Indexes

## Description
This check warns if there are unused indexes on any database collection in your instance (Need to enable "indexStats" metric collector).

## Resolution

Too many indexes on a MongoDB collection affect not only overall write performance, but also disk and memory resources. We should ideally have only the minimum indexes required.

Aside from knowing when to add an index to improve query performance, and how to modify indexes to satisfy changing query complexities, we also need to know how to identify unused indexes and cut their unnecessary overhead.

MongoDB maintains statistics about index usage on a per-server basis. You can review the index
statistics by running the following script on  mongos/mongod nodes in a sharded cluster or a replica set respectively:

> “var ldb=db.adminCommand( { listDatabases: 1 } ); for (i=0;i<ldb.databases.length;i++)  {  print('DATABASE ',ldb.databases[i].name);   if ( ldb.databases[i].name != 'admin' && ldb.databases[i].name != 'config' ) {  var db = db.getSiblingDB(ldb.databases[i].name);  var cpd = db.getCollectionNames();  for (j=0;j<cpd.length;j++) {  if ( cpd[j] !=  'system.profile' ) { print(cpd[j]);  var pui = db.runCommand({ aggregate : cpd[j] ,pipeline : [{$indexStats: {}}],cursor: { batchSize: 100 }  });  printjson(pui);  }  }  print('\n\n'); }  }”


The **accesses.ops** field contains the number of times the index was used since the server start.
We suggest evaluating this number and dropping indexes that are not used.

**Important:** Keep in mind that the index stats will be updated ONLY on the server executing the query. If some queries are sent only to secondary (or primary), this usage will not be recorded on other replica set members. 

Therfore, before deciding to drop an index, we need to analyze every member in a replica set.

For more information on unused indexes, see the [Identifying Unused Indexes in MongoDB](https://www.percona.com/blog/identifying-unused-indexes-in-mongodb/) blog post.





## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
