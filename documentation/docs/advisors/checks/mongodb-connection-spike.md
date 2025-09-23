# MongoDB Connections sudden spike

## Description
This check warns if the number of connections increases by more than 50% compared to the most recent or normal number of connections.

## Resolution

A sudden spike in MongoDB connections can be caused by multiple factors, including:

1. **Increased User Traffic:** A sudden increase in the number of users accessing your MongoDB database could result in a connection spike.

2. **Database Maintenance:** Database maintenance activities such as backups, indexing, or data migrations could cause connections to spike when a restart occurs or when application servers reconnect.

3. **Poor Connection Pooling:** Applications not using connection pooling properly can cause connections to spike because this creates new connections without properly closing old ones. ```

4. **New batch jobs:** New batch jobs added can cause a spike in the number of connections.

5. **Long-running Queries:** Long-running queries can also result in a spike in connections as they tie up resources, preventing new connections from being established. This could cause connections to queue.

6. **Poorly Configured Connection Limits:** MongoDB databases not configured with appropriate connection limits could cause connections to spike as new connections are created.

7. **Security Attacks:** Malicious actors can sometimes launch attacks to create a sudden surge of connections in an attempt to overwhelm and disrupt the database. An example of this would be Denial of Service (DOS) attacks. 


To diagnose the root cause of the sudden spike in connections, you should examine your logs and check related performance metrics to identify any unusual patterns or activity. This can help you determine if the issue is related to user traffic, application code, database configuration, or security issues.

Some of the key Performance metrics that you should examined are:
- Number of connections
- Query executors & Query targeting
- Opcounters
- Queues
- Locks time and count
- Operation execution time
- Replication lag
- MongoS logs in sharded clusters
- OS and system logs
- Network related logs
- System and Resource Utilization
  - CPU 
  - Memory
  - Disk - Disk latency, Disk IOPS




## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
