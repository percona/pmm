## Grafana dashboards for efficient database monitoring

The list of featured dashboards:

- Advanced Data Exploration
- CPU Utilization Details
- Disk Details
- HAProxy Instance Summary
- Home Dashboard
- Memory Details
- MongoDB Cluster Summary
- MongoDB InMemory Details
- MongoDB Instance Summary
- MongoDB Instances Compare
- MongoDB Instances Overview
- MongoDB MMAPv1 Details
- MongoDB ReplSet Summary
- MongoDB WiredTiger Details
- MySQL Amazon Aurora Details
- MySQL Command Handler Counters Compare
- MySQL Group Replication Summary
- MySQL InnoDB Compression Details
- MySQL InnoDB Details
- MySQL Instance Summary
- MySQL Instances Compare
- MySQL Instances Overview
- MySQL MyISAM Aria Details
- MySQL MyRocks Details
- MySQL Performance Schema Details
- MySQL Query Response Time Details
- MySQL Replication Summary
- MySQL Table Details
- MySQL User Details
- MySQL Wait Event Analyses Details
- NUMA Details
- Network Details
- Node Summary
- Node Temperature Details
- Nodes Compare
- Nodes Overview
- PXC Galera Cluster Summary
- PXC Galera Node Summary
- PXC Galera Nodes Compare
- PostgreSQL Instance Summary
- PostgreSQL Instances Compare
- PostgreSQL Instances Overview
- Processes Details
- Prometheus Exporter Status
- Prometheus Exporters Overview
- ProxySQL Instance Summary
- VictoriaMetrics
- VictoriaMetrics Agents Overview
- Valkey/Redis Clients
- Valkey/Redis Cluster Details
- Valkey/Redis Command Detail
- Valkey/Redis Load
- Valkey/Redis Memory
- Valkey/Redis Network
- Valkey/Redis Overview
- Valkey/Redis Persistence Details
- Valkey/Redis Replication
- Valkey/Redis Slowlog


These dashboards are part of [Percona Monitoring and Management](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html).

See a live demonstration at <https://pmmdemo.percona.com>.

## Reusing dashboards outside of PMM

Dashboards can be converted to be used on a dedicated prometheus instance.

Example:

- misc/convert-dash-from-PMM.py dashboards/Disk_Details.json

## Contributing

We welcome contributions to this repository! Detailed information in [CONTRIBUTING.md](CONTRIBUTING.md)
