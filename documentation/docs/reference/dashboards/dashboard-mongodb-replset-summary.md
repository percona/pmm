# MongoDB ReplSet Summary

![!image](../../images/PMM_MongoDB_ReplSet_Summary.jpg)

## Overview
Displays key metrics for individual nodes, such as their role, CPU usage, memory consumption, disk space, network traffic, uptime, and current MongoDB version.

### Feature Compatibility Version
Shows the Feature Compatibility Version (FCV) currently active in your MongoDB deployment. The FCV controls which database features are available and affects data file format compatibility between MongoDB versions.

This panel helps you confirm that your cluster is running the expected FCV—especially useful after upgrades, when the FCV may lag behind the MongoDB binary version.

Monitoring FCV is important when planning upgrades or downgrades, as setting a newer FCV can enable advanced features but may prevent rolling back to older MongoDB versions.

### Nodes
Displays the total number of nodes in the replica set, including all members regardless of state. 

Monitoring this value ensures the replica set maintains the expected number of nodes for proper replication and fault tolerance.

### DBs
Shows the total number of user-created databases, excluding system databases (like admin, local, and config). This metric helps track database growth and understand the scale of your MongoDB deployment.

### Last Election
Displays the time elapsed since the most recent primary election.

Frequent elections can indicate connectivity issues or node failures. A stable replica set should show a relatively high value.

If the value is low, it may indicate a problem that needs investigation.

### State
Shows the current replica set state of this MongoDB node. MongoDB replica set members can be in various states including PRIMARY (handling all write operations), SECONDARY (replicating data from the primary), ARBITER (participating in elections but not storing data), or several transitional states.

This status indicator helps you quickly identify the role of each node and spot any nodes experiencing issues. Color coding makes it easy to distinguish primaries (green) from secondaries (yellow) and problem states.

### CPU Usage
Shows CPU usage as a percentage from 0% to 100%. It updates every minute, turning from green to red when usage exceeds 80%. This helps quickly spot high CPU load, which could affect system performance, and monitor how hard the CPU is working at a glance.

### Memory Used
Displays the percentage of total system memory currently in use. It updates regularly, showing green up to 80% of usage and red beyond that threshold.

Use this for a quick visual indicator of memory consumption to monitor available memory without swapping as it's an easy way to assess how close the system is to its memory limits.

### Disk IO Utilization
Shows how busy the disk is handling read/write requests. The meter turns red above 80%, warning of potential slowdowns. It updates regularly, giving administrators a quick way to check if the disk is keeping up with demand or if it's becoming a bottleneck in system performance.

### Disk Space Utilization
Shows how much of the total disk space is currently in use. The meter turns red when usage exceeds 80%, warning of low free space. It updates regularly, giving you a quick way to check if the disk is nearing capacity. This helps prevent "disk full" errors that could disrupt services or system operation.

### Disk IOPS
Shows how many read and write operations the disk performs each second. The blue color helps spot spikes in disk activity. These spikes could mean the disk is struggling to keep up, which might slow down the system. It's a quick way for you to check if the disk is working too hard.

### Network Traffic
Combines both incoming (received) and outgoing (transmitted) data, excluding local traffic. It gives you a quick view of overall network activity, helping spot unusual spikes or drops in data flow that might affect system performance.

### Uptime
Shows how long the system has been running without a restart. As uptime increases, the color changes from red to orange to green, giving a quick visual indicator of system stability. Red indicates very recent restarts (less than 5 minutes), orange shows short uptimes (5 minutes to 1 hour), and green represents longer uptimes (over 1 hour). This helps you easily spot recent system restarts or confirm continuous operation.

### Version
Displays the current version of MongoDB running on the system. This information is crucial for ensuring the system is running the intended version and for quickly identifying any nodes that might need updates.

## States

### Node States
Shows the state timeline of MongoDB replica set members during the selected time range. Each node's state (PRIMARY, SECONDARY, ARBITER, etc.) is color-coded for easy monitoring, with green indicating healthy states and red showing potential issues. Use this to track role changes and identify stability problems across your replica set.

## Details

### Command Operations
Shows the rate of MongoDB operations per second, including both regular and replicated operations (query, insert, update, delete, getmore), as well as document deletions by TTL indexes. Use this metric to monitor database activity patterns and identify potential performance bottlenecks.

### Top Hottest Collections by Read
Shows the five MongoDB collections with the highest read operations per second. Use this to identify your most frequently accessed collections and optimize their performance.

### Top Hottest Collections by Write
Shows the five MongoDB collections with the highest write operations (inserts, updates, and deletes) per second. Use this to identify your most frequently modified collections and optimize their write performance.

### Query Efficiency
Shows the ratio of documents or index entries scanned versus documents returned. A ratio of 1 indicates optimal query performance where each scanned document matches the query criteria. 

Higher values suggest less efficient queries that scan many documents to find matches. Use this to identify queries that might need index optimization.

### Queued Operations
Shows the number of operations waiting because the database is busy with other operations. Use this to identify when MongoDB operations are being delayed due to resource conflicts.

### Reads & Writes
Shows both active and queued read/write operations in your MongoDB deployment. Use this to monitor database activity and identify when operations are being delayed due to high load.

### Connections
Shows the number of current and available MongoDB connections. Use this to monitor connection usage and ensure your deployment has sufficient capacity for new client connections.

### Query Execution Times
Shows the average latency in microseconds (µs) for read, write, and command operations. Use this metric to monitor query performance and identify slow operations that may need optimization.

## Collection Details

### Size of Collections
Shows storage size of MongoDB collections across different databases. Use this to monitor database growth and plan storage capacity needs.

### Number of Collections
Shows the total number of collections in each MongoDB database. Use this to track database organization and growth patterns.

## Replication

### Replication Lag
Shows how many seconds Secondary nodes are behind the Primary in replicating data. Higher values indicate potential issues with network latency or system resources. The red threshold line at 10 seconds helps identify when lag requires attention.

### Oplog Recovery Window 
Shows the time range (in seconds) between the newest and oldest operations in the oplog. Use this to ensure sufficient history is maintained for recovery and secondary synchronization.

### Oplog GB/Hour 
Shows the size of the MongoDB oplog generated by the Primary server. Use this to track oplog growth, plan storage needs, and detect high-write periods. Values are displayed in bytes with hourly intervals.

## Performance

### Flow Control
Shows the frequency and duration (in microseconds) of MongoDB write throttling. Use this to understand when your deployment is slowing down writes to keep replication lag under control.

### WiredTiger Concurrency Tickets Available
Shows how many more read and write operations your MongoDB deployment can handle simultaneously. Use this to monitor database concurrency limits and potential bottlenecks.

## Nodes Summary

### Nodes Overview
Shows key system metrics for each node: uptime, load average, memory usage, disk space, and more. Use this table to monitor the health and resource utilization of your infrastructure at a glance.

## CPU Usage
Shows CPU utilization as a percentage of total capacity, broken down by user and system activity. Use this to monitor CPU load and identify potential performance bottlenecks.

## CPU Saturation

### CPU Saturation and Max Core Usage
Shows how heavily your CPU is loaded with waiting processes and maximum core utilization. Use this to identify when your system needs more CPU capacity or when processes are competing for CPU time.

## Disk I/O and Swap Activity
Shows disk I/O operations (reads/writes) and memory swap activity for each MongoDB node, measuring data flow between storage and RAM. 

Use this metric to monitor storage performance, detect memory pressure, and identify when MongoDB's working set may exceed available RAM.

##  Network Traffic
Shows inbound and outbound network traffic for each MongoDB node, measuring data flow in bytes per second. 

Use this metric to monitor bandwidth usage, identify unusual traffic patterns, and detect potential network bottlenecks that could affect replication performance.
