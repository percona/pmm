# Dashboards overview

Dashboards are a compilation of visualizations, including charts and metrics, that enable you to view performance metrics from node to single query for multiple databases in a centralized location. 

A dashboard is a group of one or more panels organized and arranged into rows. Panels refer to individual components or visual elements that display specific data or visualizations within the dashboard's layout. These panels are the building blocks that collectively form a dashboard, providing a means to present and visualize data in various formats. Dashboards are grouped into folders. You can customize these by renaming them or creating new ones. 

Dashboards provide insightful and actionable data, enabling you to gain an overview of your system status quickly. These dashboards enable you to drill down into specific time frames, apply filters, and analyze data trends for troubleshooting and performance optimization. Customizable dashboards and real-time alerting facilitate seamless monitoring of database performance.


## Available dashboards

Performance Monitoring and Management (PMM) offers a range of dashboards you can access. Some of these dashboards are as follows:

| Category          | Dashboard                                                                                | Elements { data-sort-method='number'} |
|-------------------|------------------------------------------------------------------------------------------|:-------------------------------------:|
| Insight           | [Advanced Data Exploration]                                                              | 7
| Insight           | [Home Dashboard]                                                                         | 26
| Insight           | [Prometheus Exporter Status]                                                             | 57
| Insight           | [Prometheus Exporters Overview]                                                          | 27
| Insight           | [VictoriaMetrics]                                                                        | 52
| Insight           | [VictoriaMetrics Agents Overview]                                                        | 58
| PMM               | [PMM Inventory]                                                                          | 3
| PMM               | [Environment Overview]                                                                   | 0
| PMM               | [Environment Summary]                                                                    | 0
| OS                | [CPU Utilization Details]                                                                | 21
| OS                | [Disk Details]                                                                           | 34
| OS                | [Network Details]                                                                        | 70
| OS                | [Memory Details]                                                                         | 116
| OS                | [Node Temperature Details]                                                               | 6
| OS                | [Nodes Compare]                                                                          | 74
| OS                | [Nodes Overview]                                                                         | 115
| OS                | [Node Summary]                                                                           | 67
| OS                | [NUMA Details]                                                                           | 72
| OS                | [Processes Details]                                                                      | 35
| Prometheus        | [Prometheus Exporter Status]                                                             | 57
| Prometheus        | [Prometheus Exporters Overview]                                                          | 27
| MySQL             | [MySQL Amazon Aurora Details]                                                            | 20
| MySQL             | [MySQL Command/Handler Counters Compare]                                                 | 11
| MySQL             | [MySQL InnoDB Compression Details]                                                       | 41
| MySQL             | [MySQL InnoDB Details]                                                                   | 339
| MySQL             | [MySQL MyISAM/Aria Details]                                                              | 55
| MySQL             | [MySQL MyRocks Details]                                                                  | 101
| MySQL             | [MySQL Instance Summary]                                                                 | 90
| MySQL             | [MySQL Instances Compare]                                                                | 70
| MySQL             | [MySQL Instances Overview]                                                               | 96
| MySQL             | [MySQL Wait Event Analyses Details]                                                      | 42
| MySQL             | [MySQL Performance Schema Details]                                                       | 48
| MySQL             | [MySQL Query Response Time Details]                                                      | 49
| MySQL             | [MySQL Replication Summary]                                                              | 50
| MySQL             | [MySQL Group Replication Summary]                                                        | 18
| MySQL             | [MySQL Table Details]                                                                    | 45
| MySQL             | [MySQL User Details]                                                                     | 62
| MongoDB           | [Experimental MongoDB Collection Overview]                                                             | 100
| MongoDB           | [Experimental MongoDB Collection Details]                                                             | 100
| MongoDB           | [Experimental MongoDB Oplog Details]                                                             | 100
| MongoDB           | [MongoDB Cluster Summary]                                                                | 55
| MongoDB           | [MongoDB Instance Summary]                                                               | 42
| MongoDB           | [MongoDB Instances Compare]                                                              | 19
| MongoDB           | [MongoDB ReplSet Summary]                                                                | 130
| MongoDB           | [MongoDB InMemory Details]                                                               | 46
| MongoDB           | [MongoDB MMAPv1 Details]                                                                 | 52
| MongoDB           | [MongoDB WiredTiger Details]                                                             | 54
| PostgreSQL        | [PostgreSQL Instances Overview]                                                          | 114
| PostgreSQL        | [Experimental PostgreSQL Vacuum Monitoring]                                              | 114
| PostgreSQL        | [PostgreSQL Instance Summary]                                                            | 67
| PostgreSQL        | [PostgreSQL Instances Compare]                                                           | 89
| ProxySQL          | [ProxySQL Instance Summary]                                                              | 55
| High-availability | [PXC/Galera Node Summary]                                                                | 32
| High-availability | [PXC/Galera Cluster Summary]                                                             | 19
| High-availability | [Experimental PXC/Galera Cluster Summary]                                                 | 7
| High-availability | [PXC/Galera Nodes Compare]                                                               | 55
| High-availability | [HAProxy Instance Summary]                                                               | 113

[Advanced Data Exploration]: ../../reference/dashboards/dashboard-advanced-data-exploration.md
[Home Dashboard]: dashboard-home.md
[DB Cluster Summary]: ../../reference/dashboard-cluster-summary.md
[Prometheus Exporter Status]: ../../reference/dashboard-prometheus-exporter-status.md
[Prometheus Exporters Overview]: ../../reference/dashboard-prometheus-exporters-overview.md
[VictoriaMetrics]: ../../reference/dashboard-victoriametrics.md
[VictoriaMetrics Agents Overview]: ../../reference/dashboard-victoriametrics-agents-overview.md
[PMM Inventory]: dashboard-inventory.md
[Environment Overview]: ../../reference/dashboard-env-overview.md
[Environment Summary]: ../../reference/dashboard-environent-summary.md
[CPU Utilization Details]: ../../reference/dashboard-cpu-utilization-details.md
[Disk Details]: ../../reference/dashboard-disk-details.md
[Network Details]: ../../reference/dashboard-network-details.md
[Memory Details]: ../../reference/dashboard-memory-details.md
[Node Temperature Details]: ../../reference/dashboard-node-temperature-details.md
[Nodes Compare]: ../../reference/dashboard-nodes-compare.md
[Nodes Overview]: ../../reference/dashboard-nodes-overview.md
[Node Summary]: ../../reference/dashboard-node-summary.md
[NUMA Details]: ../../reference/dashboard-numa-details.md
[Processes Details]: ../../reference/dashboard-processes-details.md
[Prometheus Exporter Status]: ../../reference/dashboard-prometheus-exporter-status.md
[Prometheus Exporters Overview]: ../../reference/dashboard-prometheus-exporters-overview.md
[MySQL Amazon Aurora Details]: ../../reference/dashboard-mysql-amazon-aurora-details.md
[MySQL Command/Handler Counters Compare]: ../../reference/dashboard-mysql-command-handler-counters-compare.md
[MySQL InnoDB Compression Details]: ../../reference/dashboard-mysql-innodb-compression-details.md
[MySQL InnoDB Details]: ../../reference/dashboard-mysql-innodb-details.md
[MySQL MyISAM/Aria Details]: ../../reference/dashboard-mysql-myisam-aria-details.md
[MySQL MyRocks Details]: ../../reference/dashboard-mysql-myrocks-details.md
[MySQL Instance Summary]: ../../reference/dashboard-mysql-instance-summary.md
[MySQL Instances Compare]: ../../reference/dashboard-mysql-instances-compare.md
[MySQL Instances Overview]: ../../reference/dashboard-mysql-instances-overview.md
[MySQL Wait Event Analyses Details]: ../../reference/dashboard-mysql-wait-event-analyses-details.md
[MySQL Performance Schema Details]: ../../reference/dashboard-mysql-performance-schema-details.md
[MySQL Query Response Time Details]: ../../reference/dashboard-mysql-query-response-time-details.md
[MySQL Replication Summary]: ../../reference/dashboard-mysql-replication-summary.md
[MySQL Group Replication Summary]: ../../reference/dashboard-mysql-group-replication-summary.md
[MySQL Table Details]: ../../reference/dashboard-mysql-table-details.md
[MySQL User Details]: ../../reference/dashboard-mysql-user-details.md
[MySQL TokuDB Details]: ../../reference/dashboard-mysql-tokudb-details.md
[Experimental MongoDB Collection Overview]: ../../reference/dashboard-mongodb-experimental_collection_overview.md
[Experimental MongoDB Collection Details]: ../../reference/dashboard-mongodb-experimental_collection_details.md
[Experimental MongoDB Oplog Details]: ../../reference/dashboard-mongodb-experimental_oplog.md
[MongoDB Cluster Summary]: ../../reference/dashboard-mongodb-cluster-summary.md
[MongoDB Instance Summary]: ../../reference/dashboard-mongodb-instance-summary.md
[MongoDB Instances Overview]: ../../reference/dashboard-mongodb-instances-overview.md
[MongoDB Instances Compare]: ../../reference/dashboard-mongodb-instances-compare.md
[MongoDB ReplSet Summary]: ../../reference/dashboard-mongodb-replset-summary.md
[MongoDB InMemory Details]: ../../reference/dashboard-mongodb-inmemory-details.md
[MongoDB MMAPv1 Details]: ../../reference/dashboard-mongodb-mmapv1-details.md
[MongoDB WiredTiger Details]: ../../reference/dashboard-mongodb-wiredtiger-details.md
[Experimental PostgreSQL Vacuum Monitoring]: dashboard-postgresql-vacuum-monitoring-experimental.md
[PostgreSQL Instances Overview]: ../../reference/dashboard-postgresql-instances-overview.md
[PostgreSQL Instance Summary]: ../../reference/dashboard-postgresql-instance-summary.md
[PostgreSQL Instances Compare]: ../../reference/dashboard-postgresql-instances-compare.md
[ProxySQL Instance Summary]: ../../reference/dashboard-proxysql-instance-summary.md
[PXC/Galera Node Summary]: ../../reference/dashboard-pxc-galera-node-summary.md
[PXC/Galera Cluster Summary]: ../../reference/dashboard-pxc-galera-cluster-summary.md
[Experimental PXC/Galera Cluster Summary]: ../../reference/dashboard-pxc-galera-cluster-summary-experimental.md
[PXC/Galera Nodes Compare]: ../../reference/dashboard-pxc-galera-nodes-compare.md
[HAProxy Instance Summary]: ../../reference/dashboard-haproxy-instance-summary.md