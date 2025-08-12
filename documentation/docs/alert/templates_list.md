# List of available alert templates

The table below lists all the alert templates available in Percona Monitoring and Management (PMM). 

## Template catalog

- [Operating System templates](#os_alerts)
- [PMM templates](#pmm_alerts)
- [MongoDB templates](#mongodb_alerts)
- [PBM templates](#pbm_alerts)
- [MySQL templates](#mysql_alerts)
- [PostgreSQL templates](#postgresql_alerts)
- [ProxySQL templates](#proxysql_alerts)

<a id="os_alerts"></a>
### Operating System (OS) templates

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| OS | **Node high CPU load** | Monitors node CPU usage and alerts when it surpasses 80% (default threshold). Provides details about specific nodes experiencing high CPU load, indicating potential performance issues or scaling needs. | MySQL, MongoDB, PostgreSQL |
| OS | **Memory available less than a threshold** | Tracks available memory on nodes and alerts when free memory drops below 20% (default threshold). Helps prevent system instability due to memory constraints. | MySQL, MongoDB, PostgreSQL |
| OS | **Node high swap filling up** | Monitors node swap usage and alerts when it exceeds 80% (default threshold). Indicates potential memory pressure and performance degradation, allowing for timely intervention. | MySQL, MongoDB, PostgreSQL |

<a id="pmm_alerts"></a>
### PMM templates

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| PMM | **PMM agent down** | Monitors PMM Agent status and alerts when an agent becomes unreachable, indicating potential host or agent issues. | MySQL, MongoDB, PostgreSQL, ProxySQL |

<a id="mongodb_alerts"></a>
### MongoDB templates

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| MongoDB | **MongoDB down** | Detects when a MongoDB instance becomes unavailable, enabling rapid response to maintain database accessibility. When monitoring MongoDB sharded clusters, alerts using this template detect outages in any cluster component (configuration servers, Mongos routers, data-bearing nodes, and arbiters). | MongoDB |
| MongoDB | **MongoDB restarted** | Detects recent MongoDB restarts, alerting if an instance has been restarted within the last 5 minutes (default threshold). Facilitates investigation of unexpected downtime and potential issues.  | MongoDB |
| MongoDB | **MongoDB DBPath disk space utilization** | Monitors disk space usage in MongoDB's data directory and alerts when it exceeds set thresholds. Helps prevent storage-related issues and ensures adequate space for database operations.  | MongoDB |
| MongoDB | **MongoDB host SSL certificate expiry** | Tracks SSL certificate expiration dates for MongoDB hosts and alerts when certificates are approaching expiry. Enables timely certificate renewal to maintain secure connections.  | MongoDB |
| MongoDB | **MongoDB oplog window** | Monitors the oplog window size and alerts when it falls below the recommended threshold (typically 24-48 hours). Ensures sufficient time for secondary nodes to replicate data and maintain cluster consistency.  | MongoDB |
| MongoDB | **MongoDB read tickets** | Tracks read ticket availability in the WiredTiger storage engine and alerts when it falls below set thresholds. Helps optimize read performance and identify potential bottlenecks.  | MongoDB |
| MongoDB | **MongoDB replication lag is high** | Monitors replication lag and alerts when it exceeds acceptable thresholds. Crucial for maintaining data consistency across replicas and identifying synchronization issues.  | MongoDB |
| MongoDB | **MongoDB ReplicaSet has no primary** | Detects when a replica set loses its primary node and alerts users. Indicates that the cluster is in read-only mode, potentially affecting write operations and overall database functionality.  | MongoDB |
| MongoDB | **MongoDB member is in unusual state** | Identifies and alerts when replica set members enter unusual states such as Recovering, Startup, or Rollback. Helps maintain cluster health and performance by enabling quick intervention.  | MongoDB |
| MongoDB | **MongoDB write tickets** | Monitors write ticket availability in the WiredTiger storage engine and alerts when it falls below set thresholds. Aids in optimizing write performance and identifying potential bottlenecks.  | MongoDB |
| MongoDB | **MongoDB too many chunk migrations** | Monitors amount of chunk migrations in a MongoDB sharded cluster and alerts if they are more than set thresholds.  | MongoDB |

<a id="pbm_alerts"></a>
### PBM (Percona Backup for MongoDB) templates

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| PBM | **MongoDB PBM Agent down** | Monitors the status of Percona Backup for MongoDB (PBM) Agents and alerts when an Agent becomes unresponsive. This indicates potential issues with the host system or with the PBM Agent itself.  | MongoDB |
| PBM | **MongoDB PBM backup has failed** | Monitors the status of backups and alerts if they fail.  | MongoDB |
| PBM | **MongoDB PBM backup duration** |Monitors the time taken to complete a backup and alerts when it exceeds set thresholds. If the backup did not complete, no alerts are sent.  | MongoDB |
| PBM | **MongoDB PBM backup size** | Monitors the amount of disk space taken by a completed backup and alerts when it exceeds set thresholds. If the backup did not complete, no alerts are sent.  | MongoDB |

<a id="mysql_alerts"></a>
### MySQL templates

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| MySQL | **MySQL down** | Monitors MySQL instance availability and alerts when any MySQL service becomes unreachable. Enables quick response to maintain database services.  | MySQL |
| MySQL | **MySQL replication running IO** | Tracks MySQL replication I/O thread status and alerts if it stops running on a replica. Crucial for ensuring data is being received from the primary server.  | MySQL |
| MySQL | **MySQL replication running SQL** | Monitors MySQL replication SQL thread status and alerts if it stops running on a replica. Essential for verifying that received data is being applied correctly to maintain data consistency.  | MySQL |
| MySQL | **MySQL restarted** | Detects recent MySQL restarts, alerting if an instance has been restarted within the last 5 minutes (default threshold). Aids in investigating unexpected downtime and potential issues.  | MySQL |
| MySQL | **MySQL connections in use** | Tracks MySQL connection usage and alerts when the percentage of active connections exceeds 80% of the maximum allowed (default threshold). Helps prevent performance degradation due to connection overload.  | MySQL |

<a id="postgresql_alerts"></a>
### PostgreSQL templates

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| PostgreSQL | **PostgreSQL down** | Detects when PostgreSQL instances become unavailable, enabling quick response to maintain database services. Provides details about affected services and nodes.  | PostgreSQL |
| PostgreSQL | **PostgreSQL restarted** | Identifies recent PostgreSQL restarts, alerting if an instance has been restarted within the last 5 minutes (default threshold). Aids in investigating unexpected downtime and potential issues.  | PostgreSQL |
| PostgreSQL | **PostgreSQL connections in use** | Monitors PostgreSQL connection usage and alerts when the percentage of active connections exceeds 80% of the maximum allowed (default threshold). Helps prevent performance degradation due to excessive connections.  | PostgreSQL |
| PostgreSQL | **PostgreSQL index bloat is high** | Detects excessive index bloat and alerts users. Helps identify performance degradation due to bloated indexes, enabling timely maintenance to improve query performance.  | PostgreSQL |
| PostgreSQL | **PostgreSQL high number of dead tuples** | Monitors the accumulation of dead tuples in relations and alerts when they exceed set thresholds. Indicates potential issues with vacuum settings and helps optimize storage and query performance.  | PostgreSQL |
| PostgreSQL | **PostgreSQL has a high number of statement timeouts** | Tracks and alerts on frequent query cancellations due to statement timeouts. Helps identify various issues such as high load, poorly written queries, or inadequate resource allocation.  | PostgreSQL |
| PostgreSQL | **PostgreSQL table bloat is high** | Detects excessive table bloat and alerts users. Indicates a need to adjust vacuum settings for specific relations or globally, helping to maintain optimal query performance and storage efficiency.  | PostgreSQL |
| PostgreSQL | **PostgreSQL high rate of transaction rollbacks** | Monitors the ratio of transaction rollbacks to commits and alerts on high rates. Helps identify potential application or database issues leading to frequent transaction failures.  | PostgreSQL |
| PostgreSQL | **PostgreSQL tables not auto analyzed** | Identifies tables that are not being auto-analyzed and alerts users. Crucial for maintaining accurate statistics and generating proper query execution plans.  | PostgreSQL |
| PostgreSQL | **PostgreSQL tables not auto vacuumed** | Detects tables that are not being auto-vacuumed and alerts users. Essential for managing bloat, optimizing storage, and maintaining overall database health.  | PostgreSQL |
| PostgreSQL | **PostgreSQL unused replication slot** | Identifies and alerts on unused replication slots. Helps prevent excessive WAL retention and potential disk space issues, especially when replicas are offline.  | PostgreSQL |

<a id="proxysql_alerts"></a>
### ProxySQL templates

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| ProxySQL | **ProxySQL server status** | Monitors ProxySQL server status and alerts when a server transitions to OFFLINE_SOFT (3) or OFFLINE_HARD (4) state. Includes critical details such as server endpoint, hostgroup, and associated ProxySQL service. This alert is essential for maintaining high availability and preventing database access disruptions.  | ProxySQL |
