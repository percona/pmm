# Dashboards overview

Dashboards are a compilation of visualizations, including charts and metrics, that enable you to view performance metrics from node to single query for multiple databases in a centralized location.

A dashboard is a group of one or more panels organized and arranged into rows. Panels refer to individual components or visual elements that display specific data or visualizations within the dashboard's layout. These panels are the building blocks that collectively form a dashboard, providing a means to present and visualize data in various formats. Dashboards are grouped into folders. You can customize these by renaming them or creating new ones.

Dashboards provide insightful and actionable data, enabling you to gain an overview of your system status quickly. These dashboards enable you to drill down into specific time frames, apply filters, and analyze data trends for troubleshooting and performance optimization. Customizable dashboards and real-time alerting facilitate seamless monitoring of database performance.


## Available dashboards

Performance Monitoring and Management (PMM) offers a range of dashboards you can access. Some of these dashboards are as follows:

=== "Insight"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [Advanced Data Exploration]                                                              | Explore and analyze metrics with custom queries
    | [Home Dashboard]                                                                         | Overview of monitored environments and quick access to key dashboards
    | [Prometheus Exporter Status]                                                             | Monitor exporter health and availability
    | [Prometheus Exporters Overview]                                                          | Resource usage (CPU, memory) across all exporters
    | [VictoriaMetrics]                                                                        | VictoriaMetrics performance and storage metrics
    | [VictoriaMetrics Agents Overview]                                                        | VictoriaMetrics agents status and data collection

=== "PMM"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [PMM Inventory]                                                                          | Manage monitored services, nodes, and agents
    | [Environment Overview]                                                                   | High-level view of all monitored environments
    | [Environment Summary]                                                                    | Aggregated metrics across environments

=== "OS"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [CPU Utilization Details]                                                                | CPU usage, load averages, and core utilization
    | [Disk Details]                                                                           | Disk I/O, latency, and space utilization
    | [Network Details]                                                                        | Network traffic, errors, and interface statistics
    | [Memory Details]                                                                         | Memory usage, swap, and caching
    | [Node Temperature Details]                                                               | Hardware temperature monitoring
    | [Nodes Compare]                                                                          | Side-by-side comparison of multiple nodes
    | [Nodes Overview]                                                                         | Summary view of all monitored nodes
    | [Node Summary]                                                                           | System information and resource usage for a single node
    | [NUMA Details]                                                                           | NUMA node memory allocation and performance
    | [Processes Details]                                                                      | Process-level CPU, memory, and I/O metrics

=== "MySQL"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [MySQL Amazon Aurora Details]                                                            | Aurora-specific metrics and replication
    | [MySQL Command/Handler Counters Compare]                                                 | Compare command execution patterns across instances
    | [MySQL InnoDB Compression Details]                                                       | InnoDB compression efficiency and performance
    | [MySQL InnoDB Details]                                                                   | InnoDB storage engine metrics and buffer pool statistics
    | [MySQL MyISAM/Aria Details]                                                              | MyISAM and Aria storage engine performance
    | [MySQL MyRocks Details]                                                                  | MyRocks storage engine metrics
    | [MySQL Instance Summary]                                                                 | MySQL instance health and performance overview
    | [MySQL Instances Compare]                                                                | Compare metrics across multiple MySQL instances
    | [MySQL Instances Overview]                                                               | Summary of all monitored MySQL instances
    | [MySQL Wait Event Analyses Details]                                                      | Identify and analyze wait events and bottlenecks
    | [MySQL Performance Schema Details]                                                       | Performance Schema instrumentation and metrics
    | [MySQL Query Response Time Details]                                                      | Query response time distribution analysis
    | [MySQL Replication Summary]                                                              | Replication status, lag, and topology
    | [MySQL Group Replication Summary]                                                        | Group replication health and performance
    | [MySQL Table Details]                                                                    | Table-level statistics and performance
    | [MySQL User Details]                                                                     | User connection and activity monitoring

=== "MongoDB"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [Experimental MongoDB Collection Overview]                                               | Collection-level statistics (experimental)
    | [Experimental MongoDB Collection Details]                                                | Detailed collection metrics (experimental)
    | [Experimental MongoDB Oplog Details]                                                     | Oplog operations and replication (experimental)
    | [MongoDB Cluster Summary]                                                                | Sharded cluster health and performance overview
    | [MongoDB Instance Summary]                                                               | MongoDB instance metrics and operations
    | [MongoDB Instances Compare]                                                              | Compare metrics across MongoDB instances
    | [MongoDB ReplSet Summary]                                                                | Replica set health, lag, and member status
    | [MongoDB InMemory Details]                                                               | InMemory storage engine performance
    | [MongoDB MMAPv1 Details]                                                                 | MMAPv1 storage engine metrics
    | [MongoDB WiredTiger Details]                                                             | WiredTiger storage engine performance and caching

=== "PostgreSQL"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [PostgreSQL Instances Overview]                                                          | High-level overview of all PostgreSQL instances
    | [Experimental PostgreSQL Vacuum Monitoring]                                              | Vacuum operations and table bloat (experimental)
    | [PostgreSQL Instance Summary]                                                            | PostgreSQL instance health and performance
    | [PostgreSQL Instances Compare]                                                           | Compare metrics across PostgreSQL instances

=== "Valkey/Redis"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [Valkey/Redis Overview]                                                                  | Deployment health and performance summary
    | [Valkey/Redis Clients]                                                                   | Client connections and blocked clients
    | [Valkey/Redis Cluster Details]                                                           | Cluster topology and replication offsets
    | [Valkey/Redis Command Details]                                                           | Command throughput and latency patterns
    | [Valkey/Redis Load]                                                                      | Workload distribution and I/O threading
    | [Valkey/Redis Memory]                                                                    | Memory usage and eviction patterns
    | [Valkey/Redis Network]                                                                   | Network bandwidth and traffic patterns
    | [Valkey/Redis Persistence]                                                               | RDB and AOF operations
    | [Valkey/Redis Replication]                                                               | Replication lag and synchronization status
    | [Valkey/Redis Slowlog]                                                                   | Slow command identification and bottleneck detection

=== "ProxySQL"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [ProxySQL Instance Summary]                                                              | ProxySQL performance, connection pooling, and query routing

=== "HA"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [PXC/Galera Node Summary]                                                                | Individual node health in PXC/Galera clusters
    | [PXC/Galera Cluster Summary]                                                             | Cluster-wide health and replication flow
    | [Experimental PXC/Galera Cluster Summary]                                                | Enhanced cluster monitoring (experimental)
    | [PXC/Galera Nodes Compare]                                                               | Compare metrics across PXC/Galera nodes
    | [HAProxy Instance Summary]                                                               | HAProxy load balancer performance and backend health


[Advanced Data Exploration]: ../../reference/dashboards/dashboard-advanced-data-exploration.md
[Home Dashboard]: ../../reference/dashboards/dashboard-home.md
[Prometheus Exporter Status]: ../../reference/dashboards/dashboard-prometheus-exporter-status.md
[Prometheus Exporters Overview]: ../../reference/dashboards/dashboard-prometheus-exporters-overview.md
[VictoriaMetrics]: ../../reference/dashboards/dashboard-victoriametrics.md
[VictoriaMetrics Agents Overview]: ../../reference/dashboards/dashboard-victoriametrics-agents-overview.md
[PMM Inventory]: ../../use/dashboard-inventory.md
[Environment Overview]: ../../reference/dashboards/dashboard-env-overview.md
[Environment Summary]: ../../reference/dashboards/dashboard-environment-summary.md
[CPU Utilization Details]: ../../reference/dashboards/dashboard-cpu-utilization-details.md
[Disk Details]: ../../reference/dashboards/dashboard-disk-details.md
[Network Details]: ../../reference/dashboards/dashboard-network-details.md
[Memory Details]: ../../reference/dashboards/dashboard-memory-details.md
[Node Temperature Details]: ../../reference/dashboards/dashboard-node-temperature-details.md
[Nodes Compare]: ../../reference/dashboards/dashboard-nodes-compare.md
[Nodes Overview]: ../../reference/dashboards/dashboard-nodes-overview.md
[Node Summary]: ../../reference/dashboards/dashboard-node-summary.md
[NUMA Details]: ../../reference/dashboards/dashboard-numa-details.md
[Processes Details]: ../../reference/dashboards/dashboard-processes-details.md
[Prometheus Exporter Status]: ../../reference/dashboards/dashboard-prometheus-exporter-status.md
[Prometheus Exporters Overview]: ../../reference/dashboards/dashboard-prometheus-exporters-overview.md
[MySQL Amazon Aurora Details]: ../../reference/dashboards/dashboard-mysql-amazon-aurora-details.md
[MySQL Command/Handler Counters Compare]: ../../reference/dashboards/dashboard-mysql-command-handler-counters-compare.md
[MySQL InnoDB Compression Details]: ../../reference/dashboards/dashboard-mysql-innodb-compression-details.md
[MySQL InnoDB Details]: ../../reference/dashboards/dashboard-mysql-innodb-details.md
[MySQL MyISAM/Aria Details]: ../../reference/dashboards/dashboard-mysql-myisam-aria-details.md
[MySQL MyRocks Details]: ../../reference/dashboards/dashboard-mysql-myrocks-details.md
[MySQL Instance Summary]: ../../reference/dashboards/dashboard-mysql-instance-summary.md
[MySQL Instances Compare]: ../../reference/dashboards/dashboard-mysql-instances-compare.md
[MySQL Instances Overview]: ../../reference/dashboards/dashboard-mysql-instances-overview.md
[MySQL Wait Event Analyses Details]: ../../reference/dashboards/dashboard-mysql-wait-event-analyses-details.md
[MySQL Performance Schema Details]: ../../reference/dashboards/dashboard-mysql-performance-schema-details.md
[MySQL Query Response Time Details]: ../../reference/dashboards/dashboard-mysql-query-response-time-details.md
[MySQL Replication Summary]: ../../reference/dashboards/dashboard-mysql-replication-summary.md
[MySQL Group Replication Summary]: ../../reference/dashboards/dashboard-mysql-group-replication-summary.md
[MySQL Table Details]: ../../reference/dashboards/dashboard-mysql-table-details.md
[MySQL User Details]: ../../reference/dashboards/dashboard-mysql-user-details.md
[MySQL TokuDB Details]: ../../reference/dashboards/dashboard-mysql-tokudb-details.md
[Experimental MongoDB Collection Overview]: ../../reference/dashboards/dashboard-mongodb-experimental_collection_overview.md
[Experimental MongoDB Collection Details]: ../../reference/dashboards/dashboard-mongodb-experimental_collection_details.md
[Experimental MongoDB Oplog Details]: ../../reference/dashboards/dashboard-mongodb-experimental_oplog.md
[MongoDB Cluster Summary]: ../../reference/dashboards/dashboard-mongodb-cluster-summary.md
[MongoDB Instance Summary]: ../../reference/dashboards/dashboard-mongodb-instance-summary.md
[MongoDB Instances Overview]: ../../reference/dashboards/dashboard-mongodb-instances-overview.md
[MongoDB Instances Compare]: ../../reference/dashboards/dashboard-mongodb-instances-compare.md
[MongoDB ReplSet Summary]: ../../reference/dashboards/dashboard-mongodb-replset-summary.md
[MongoDB InMemory Details]: ../../reference/dashboards/dashboard-mongodb-inmemory-details.md
[MongoDB MMAPv1 Details]: ../../reference/dashboards/dashboard-mongodb-mmapv1-details.md
[MongoDB WiredTiger Details]: ../../reference/dashboards/dashboard-mongodb-wiredtiger-details.md
[Experimental PostgreSQL Vacuum Monitoring]: ../../reference/dashboards/dashboard-postgresql-vacuum-monitoring-experimental.md
[PostgreSQL Instances Overview]: ../../reference/dashboards/dashboard-postgresql-instances-overview.md
[PostgreSQL Instance Summary]: ../../reference/dashboards/dashboard-postgresql-instance-summary.md
[PostgreSQL Instances Compare]: ../../reference/dashboards/dashboard-postgresql-instances-compare.md
[ProxySQL Instance Summary]: ../../reference/dashboards/dashboard-proxysql-instance-summary.md
[Valkey/Redis Overview]: ../../reference/dashboards/dashboard-valkey-redis-overview.md
[Valkey/Redis Clients]: ../../reference/dashboards/dashboard-valkey-redis-clients.md
[Valkey/Redis Cluster Details]: ../../reference/dashboards/dashboard-valkey-redis-cluster-details.md
[Valkey/Redis Command Details]: ../../reference/dashboards/dashboard-valkey-redis-command-detail.md
[Valkey/Redis Load]: ../../reference/dashboards/dashboard-valkey-redis-load.md
[Valkey/Redis Memory]: ../../reference/dashboards/dashboard-valkey-redis-memory.md
[Valkey/Redis Network]: ../../reference/dashboards/dashboard-valkey-redis-network.md
[Valkey/Redis Persistence]: ../../reference/dashboards/dashboard-valkey-redis-persistence-details.md
[Valkey/Redis Replication]: ../../reference/dashboards/dashboard-valkey-redis-replication.md
[Valkey/Redis Slowlog]: ../../reference/dashboards/dashboard-valkey-redis-slowlog.md
[PXC/Galera Node Summary]: ../../reference/dashboards/dashboard-pxc-galera-node-summary.md
[PXC/Galera Cluster Summary]: ../../reference/dashboards/dashboard-pxc-galera-cluster-summary.md
[Experimental PXC/Galera Cluster Summary]: ../../reference/dashboards/dashboard-pxc-galera-cluster-summary-experimental.md
[PXC/Galera Nodes Compare]: ../../reference/dashboards/dashboard-pxc-galera-nodes-compare.md
[HAProxy Instance Summary]: ../../reference/dashboards/dashboard-haproxy-instance-summary.md
