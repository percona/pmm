# MongoDB high connections

## Description
This check returns warnings if there is an increase in the number of connections.


## Resolution
High connection count in MongoDB can be caused by various factors, including application design, resource utilization, and server configuration. 
Here are some common reasons and potential solutions for high connection counts in MongoDB:

-  **Connection pooling issues**

   Connection pooling is a technique which is used to create and manage a pool of database connections. Not configuring the connection pool correctly can lead to high connection counts in MongoDB.

   To resolve connection pool issues, you can check the maximum number of connections in the pool (the `maxPoolSize` option) and adjust it as needed. Additionally, you should check the connection timeout values to ensure that connections are being reused effectively.
   
-  **Connection leaks**

   Connection leaks occur when a connection is not closed correctly. Over time, this can lead to additional connections being created.  To prevent connection leaks, make sure that your application closes connections after they are no longer needed.

-  **Batch jobs**

   Regular batch jobs increase the number of connections and impact resource utilization. You can verify what kind of operations are used by using the [**mongostat**](https://www.mongodb.com/docs/database-tools/mongostat/) utility.

-  **Poorly optimized queries**

   Queries that perform full table scans or involve a large number of documents can consume a significant amount of resources. Since the queues will be increased, this can result in high connection counts. 
   
   To reduce connection counts caused by queries, it's essential to optimize your queries. You can use the MongoDB **explain()** method to understand query performance and identify potential areas for optimization.

-  **Application design issues**

If the application has been designed to create too many connections, it can lead to high connection counts in MongoDB. To minimize connection overhead, consider implementing connection sharing, where connections are reused across application instances.


-  **Server resource constraints**

Insufficient server resources such as CPU, memory, or disk I/O can lead to high connection counts in MongoDB. Check the system resource utilization and increase resources as needed. You may also want to consider adding more members to the replica set to distribute the load.


In summary, to reduce high connection counts in MongoDB, it's essential to configure connection pooling correctly, optimize queries, adjust the application design, allocate sufficient server resources and prevent connection leaks.


## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
