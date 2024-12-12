# PMM alert templates

Alert templates provide a set of common events and expressions for alerting, serving as a foundation for creating alert rules.

Percona Monitoring and Management (PMM) offers three categories of alert templates to enhance database performance monitoring:

1. **Built-in templates**: templates that are available out-of-the-box with the PMM installation and are available to all PMM users.
2. **Percona Platform templates**: additional templates dynamically delivered to PMM if the instance is [connected to Percona Platform](../how-to/integrate-platform.md) using a Percona Account.
    When connected to the Platform, PMM automatically downloads these templates if the **Telemetry** option is enabled under **Configuration > Settings > Advanced Settings**.
3. **Custom templates**: user-created templates for specific needs not met by built-in or Percona Platform templates. These allow you to tailor alerts to your unique environment and requirements.
   For details on creating custom templates, see [Percona Alerting](../get-started/alerting.md#configure-alert-templates).

## Accessing alert templates

To check the alert templates for your PMM instance, go to PMM > **Alerting > Alert Rule Templates** tab.

## Available alert template

The table below lists all the alert templates available in Percona Monitoring and Management (PMM).

This list includes both built-in templates (accessible to all PMM users), and Percona customers templates.

To access the Percona customers templates, you must be a Percona customer and [connect PMM to Percona Platform](../how-to/integrate-platform.md) using a Percona Account.

| Area | Template name | Description | Available for | Database technology |
| :----|:------------- | :---------- | :------------ | :------------------ |
| OS | **Node high CPU load** | Monitors node CPU usage and alerts when it surpasses 80% (default threshold). Provides details about specific nodes experiencing high CPU load, indicating potential performance issues or scaling needs. | • Percona customers<br>• Community | MySQL, MongoDB, PostgreSQL |
| OS | **Memory available less than a threshold** | Tracks available memory on nodes and alerts when free memory drops below 20% (default threshold). Helps prevent system instability due to memory constraints. | • Percona customers<br>• Community | MySQL, MongoDB, PostgreSQL |
| OS | **Node high swap filling up** | Monitors node swap usage and alerts when it exceeds 80% (default threshold). Indicates potential memory pressure and performance degradation, allowing for timely intervention. | • Percona customers<br>• Community | MySQL, MongoDB, PostgreSQL |
| PMM | **PMM agent down** | Monitors PMM Agent status and alerts when an agent becomes unreachable, indicating potential host or agent issues. | • Percona customers<br>• Community | MySQL, MongoDB, PostgreSQL, ProxySQL |
| PMM | **Backup failed [Technical Preview]** | Monitors backup processes and alerts on failures, providing details about the failed backup artifact and service. Helps maintain data safety and recovery readiness. This template is currently in Technical Preview status and should be used for testing purposes only as it is subject to change. | • Percona customers<br>• Community | MySQL, MongoDB, PostgreSQL, ProxySQL |
| MongoDB | **MongoDB down** | Detects when a MongoDB instance becomes unavailable, enabling rapid response to maintain database accessibility. | • Percona customers<br>• Community | MongoDB |
| MongoDB | **Memory used by MongoDB connections** | Tracks MongoDB connection memory usage and alerts when it exceeds configurable thresholds. Helps identify and address potential performance issues caused by high memory consumption. | • Percona customers<br>• Community | MongoDB |
| MongoDB | **Memory used by MongoDB** | Monitors overall MongoDB memory usage and alerts when it exceeds 80% of total system memory. Provides details about specific MongoDB services and nodes experiencing high memory consumption, aiding in resource optimization. | • Percona customers<br>• Community | MongoDB |
| MongoDB | **MongoDB restarted** | Detects recent MongoDB restarts, alerting if an instance has been restarted within the last 5 minutes (default threshold). Facilitates investigation of unexpected downtime and potential issues. | • Percona customers<br>• Community | MongoDB |
| MongoDB | **MongoDB DBPath disk space utilization** | Monitors disk space usage in MongoDB's data directory and alerts when it exceeds set thresholds. Helps prevent storage-related issues and ensures adequate space for database operations. | Percona customers | MongoDB |
| MongoDB | **MongoDB host SSL certificate expiry** | Tracks SSL certificate expiration dates for MongoDB hosts and alerts when certificates are approaching expiry. Enables timely certificate renewal to maintain secure connections. | Percona customers | MongoDB |
| MongoDB | **MongoDB oplog window** | Monitors the oplog window size and alerts when it falls below the recommended threshold (typically 24-48 hours). Ensures sufficient time for secondary nodes to replicate data and maintain cluster consistency. | Percona customers | MongoDB |
| MongoDB | **MongoDB read tickets** | Tracks read ticket availability in the WiredTiger storage engine and alerts when it falls below set thresholds. Helps optimize read performance and identify potential bottlenecks. | Percona customers | MongoDB |
| MongoDB | **MongoDB replication lag is high** | Monitors replication lag and alerts when it exceeds acceptable thresholds. Crucial for maintaining data consistency across replicas and identifying synchronization issues. | Percona customers | MongoDB |
| MongoDB | **MongoDB ReplicaSet has no primary** | Detects when a replica set loses its primary node and alerts users. Indicates that the cluster is in read-only mode, potentially affecting write operations and overall database functionality. | Percona customers | MongoDB |
| MongoDB | **MongoDB member is in unusual state** | Identifies and alerts when replica set members enter unusual states such as Recovering, Startup, or Rollback. Helps maintain cluster health and performance by enabling quick intervention. | Percona customers | MongoDB |
| MongoDB | **MongoDB write tickets** | Monitors write ticket availability in the WiredTiger storage engine and alerts when it falls below set thresholds. Aids in optimizing write performance and identifying potential bottlenecks. | Percona customers | MongoDB |
| MongoDB | **MongoDB too many chunk migrations** | Monitors amount of chunk migrations in a MongoDB sharded cluster and alerts if they are more than set thresholds. | Percona customers | MongoDB |
| PBM | **MongoDB PBM Agent down** | Monitors the status of Percona Backup for MongoDB (PBM) Agents and alerts when an Agent becomes unresponsive. This indicates potential issues with the host system or with the PBM Agent itself. | • Percona customers<br>• Community | MongoDB |
| PBM | **MongoDB PBM backup duration** |Monitors the time taken to complete a backup and alerts when it exceeds set thresholds. If the backup did not complete, no alerts are sent. | • Percona customers<br>• Community | MongoDB |
| PBM | **MongoDB PBM backup size** | Monitors the amount of disk space taken by a completed backup and alerts when it exceeds set thresholds. If the backup did not complete, no alerts are sent. | • Percona customers<br>• Community | MongoDB |
| MySQL | **MySQL down** | Monitors MySQL instance availability and alerts when any MySQL service becomes unreachable. Enables quick response to maintain database services. | • Percona customers<br>• Community | MySQL |
| MySQL | **MySQL replication running IO** | Tracks MySQL replication I/O thread status and alerts if it stops running on a replica. Crucial for ensuring data is being received from the primary server. | • Percona customers<br>• Community | MySQL |
| MySQL | **MySQL replication running SQL** | Monitors MySQL replication SQL thread status and alerts if it stops running on a replica. Essential for verifying that received data is being applied correctly to maintain data consistency. | • Percona customers<br>• Community | MySQL |
| MySQL | **MySQL restarted** | Detects recent MySQL restarts, alerting if an instance has been restarted within the last 5 minutes (default threshold). Aids in investigating unexpected downtime and potential issues. | • Percona customers<br>• Community | MySQL |
| MySQL | **MySQL connections in use** | Tracks MySQL connection usage and alerts when the percentage of active connections exceeds 80% of the maximum allowed (default threshold). Helps prevent performance degradation due to connection overload. | • Percona customers<br>• Community | MySQL |
| PostgreSQL | **PostgreSQL down** | Detects when PostgreSQL instances become unavailable, enabling quick response to maintain database services. Provides details about affected services and nodes. | • Percona customers<br>• Community | PostgreSQL |
| PostgreSQL | **PostgreSQL restarted** | Identifies recent PostgreSQL restarts, alerting if an instance has been restarted within the last 5 minutes (default threshold). Aids in investigating unexpected downtime and potential issues. | • Percona customers<br>• Community | PostgreSQL |
| PostgreSQL | **PostgreSQL connections in use** | Monitors PostgreSQL connection usage and alerts when the percentage of active connections exceeds 80% of the maximum allowed (default threshold). Helps prevent performance degradation due to excessive connections. | • Percona customers<br>• Community | PostgreSQL |
| PostgreSQL | **PostgreSQL index bloat is high** | Detects excessive index bloat and alerts users. Helps identify performance degradation due to bloated indexes, enabling timely maintenance to improve query performance. | Percona customers | PostgreSQL |
| PostgreSQL | **PostgreSQL high number of dead tuples** | Monitors the accumulation of dead tuples in relations and alerts when they exceed set thresholds. Indicates potential issues with vacuum settings and helps optimize storage and query performance. | Percona customers | PostgreSQL |
| PostgreSQL | **PostgreSQL has a high number of statement timeouts** | Tracks and alerts on frequent query cancellations due to statement timeouts. Helps identify various issues such as high load, poorly written queries, or inadequate resource allocation. | Percona customers | PostgreSQL |
| PostgreSQL | **PostgreSQL table bloat is high** | Detects excessive table bloat and alerts users. Indicates a need to adjust vacuum settings for specific relations or globally, helping to maintain optimal query performance and storage efficiency. | Percona customers | PostgreSQL |
| PostgreSQL | **PostgreSQL high rate of transaction rollbacks** | Monitors the ratio of transaction rollbacks to commits and alerts on high rates. Helps identify potential application or database issues leading to frequent transaction failures. | Percona customers | PostgreSQL |
| PostgreSQL | **PostgreSQL tables not auto analyzed** | Identifies tables that are not being auto-analyzed and alerts users. Crucial for maintaining accurate statistics and generating proper query execution plans. | Percona customers | PostgreSQL |
| PostgreSQL | **PostgreSQL tables not auto vacuumed** | Detects tables that are not being auto-vacuumed and alerts users. Essential for managing bloat, optimizing storage, and maintaining overall database health. | Percona customers | PostgreSQL |
| PostgreSQL | **PostgreSQL unused replication slot** | Identifies and alerts on unused replication slots. Helps prevent excessive WAL retention and potential disk space issues, especially when replicas are offline. | Percona customers | PostgreSQL |
| ProxySQL | **ProxySQL server status** | Monitors ProxySQL server status and alerts when a server transitions to OFFLINE_SOFT (3) or OFFLINE_HARD (4) state. Includes critical details such as server endpoint, hostgroup, and associated ProxySQL service. This alert is essential for maintaining high availability and preventing database access disruptions. | • Percona customers<br>• Community | ProxySQL |
