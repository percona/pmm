# List of available alert templates

The table below lists all the alert templates available in Percona Monitoring and Management (PMM).

## Template catalog

- [Operating System templates](#os_alerts)
- [PMM templates](#pmm_alerts)
- [PMM High Availability templates](#pmm_ha_alerts)
- [PMM internal component templates](#pmm_component_alerts)
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
| PMM | **Backup failed [Technical Preview]** | Monitors backup processes and raises alerts on failures. Provides details about the failed backup artifact and affected service to ensure data safety and recovery readiness. This template is currently in [Technical Preview](../reference/glossary.md) and is intended for testing purposes only, as it is subject to change. | MySQL, MongoDB, PostgreSQL, ProxySQL |

<a id="pmm_ha_alerts"></a>
### PMM High Availability templates

These templates monitor a PMM Server [High Availability cluster](../install-pmm/install-HA-clustered.md). They never raise alerts on a standalone (non-HA) PMM installation, because the underlying HA metrics are only exposed when HA mode is enabled.

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| PMM HA | **PMM HA cluster has no active leader** | Alerts when no node in the cluster holds the Raft leader lease, which means that leader-only work such as advisor checks, backups, telemetry and scheduled tasks has stopped. | PMM |
| PMM HA | **PMM HA split-brain detected** | Alerts when more than one node claims Raft leadership at the same time, which means the nodes have formed separate Raft clusters instead of one. | PMM |
| PMM HA | **PMM HA leader is flapping** | Alerts when the Raft term on a node changes more than 5 times (default threshold) within 10 minutes, which indicates an unstable network or a leader that keeps restarting. | PMM |
| PMM HA | **PMM HA node unreachable** | Alerts when fewer nodes report HA metrics than the number configured in `PMM_HA_PEERS`, which indicates that at least one PMM Server node is down or isolated. | PMM |
| PMM HA | **PMM HA quorum at risk** | Alerts when the number of live Raft voters has fallen to or below the smallest majority that still forms a quorum. Applies to clusters of three nodes or more. | PMM |

#### Enable the HA alerts

These templates are available as soon as PMM Server is installed, but like all other alert templates they do not create alert rules by themselves. To start receiving notifications:

1. Go to **Alerting > Alert rule templates** and find the template you want to use.
2. Select **New alert rule from template**.
3. Choose a folder and an evaluation group. Percona recommends grouping the HA rules together, for example in a **PMM HA** group.
4. Configure a [contact point](./contact_points.md) so that the alerts reach you.
5. Repeat for each of the five templates.

#### Coverage limitations

Keep the following in mind when you rely on these alerts:

- **A complete cluster outage cannot be detected from inside the cluster.** Each node's metrics are collected only by the monitoring agent running on that same node, so when every node is down there is nothing left to report it and all HA alerts fall silent. Monitor the load balancer endpoint from outside the cluster to cover this case.
- **Split-brain detection requires the isolated node to still reach shared storage.** If a network partition also cuts a node off from the shared VictoriaMetrics storage, its metrics never arrive and the second leader stays invisible.
- **An ordinary network partition does not cause a split brain.** Raft is designed to prevent two leaders: a node in a minority partition cannot win an election. If no side of the partition holds a majority, the cluster is left with no leader at all and *PMM HA no active leader* fires. If a majority survives, it elects a new leader within seconds and none of these alerts fire, including *PMM HA quorum at risk* and *PMM HA node unreachable*. The isolated node keeps writing metrics and keeps reporting itself as a voter, because each node reads its Raft membership from its own local copy, which the partition does not change. So from the metrics alone the cluster still looks complete. What does change is the isolated node's Raft term, which climbs as it repeatedly fails to win elections, so *PMM HA leader is flapping* is the alert most likely to fire in that situation. The split-brain alert covers the rarer case where nodes end up in separate clusters, for example when they cannot discover each other at startup and each bootstraps its own.
- **The node unreachable alert names one node at a time.** A node that stopped within the last 6 hours is named in the alert. A node that has never reported since the cluster started, or that has been down for longer than 6 hours, cannot be named, because no metrics remain to identify it; the alert then fires without a node name. These two cases do not combine: if any node can be named, the alert names it and does not separately report the ones it cannot. Use the **High Availability** page to see the full picture whenever more than one node is missing.
- **The quorum alert does not apply to one- and two-node clusters.** On those the condition would be permanently true, since with two nodes quorum is two and both nodes are always essential, so the template suppresses itself and stays silent. *PMM HA node unreachable* does cover them, so rely on it instead.
- **Changing `PMM_HA_PEERS` raises alerts until the rollout finishes.** The expected node count is the highest value any node reports, so while some nodes carry the new peer list and others still carry the old one, the cluster looks smaller than expected. *PMM HA node unreachable* and *PMM HA quorum at risk* can both fire for the duration of a rolling restart, in either direction. Let the rollout finish before treating either as a real failure.

<a id="pmm_component_alerts"></a>
### PMM internal component templates

These templates monitor the components that make up PMM Server itself, rather than the databases PMM monitors. They alert when a component stops responding to the health check that PMM already collects, so no extra configuration is needed.

| Area | Template name | Description | Database technology |
| :----|:------------- | :---------- | :------------------ |
| PMM | **PMM VictoriaMetrics is down** | Alerts when a VictoriaMetrics component stops responding. Covers the single instance used by a standalone PMM Server and the vmselect, vminsert, vmstorage and vmagent components used by a clustered deployment. | PMM |
| PMM | **PMM ClickHouse is down** | Alerts when the ClickHouse instance that stores Query Analytics data stops responding. Query Analytics stops collecting while this lasts; metrics are unaffected. | PMM |
| PMM | **PMM Grafana is down** | Alerts when a Grafana instance stops responding. The user interface, dashboards and alert rule evaluation all depend on it. | PMM |
| PMM | **PMM Query Analytics API is down** | Alerts when the qan-api2 component stops responding. Query Analytics stops accepting new query data and its page cannot load. | PMM |

#### Coverage limitations

These alerts are evaluated by Grafana querying VictoriaMetrics, which means two of them cannot observe their own total failure. Keep the following in mind:

- **VictoriaMetrics being down completely cannot be detected by this alert.** The rule has to query VictoriaMetrics to run, so when VictoriaMetrics is gone the query cannot execute and the expression never evaluates. What the alert does cover is a single component failing while the rest still answers queries, which is the common case in a clustered deployment. A complete outage instead shows up as *every* alert rule reporting a datasource error at once. Monitor PMM Server from outside to cover this case.
- **Grafana being down everywhere cannot be detected either**, because Grafana is what evaluates the alert rules. In a clustered deployment the alert still catches Grafana failing on one node while another node keeps evaluating.
- **ClickHouse and Query Analytics are fully covered.** VictoriaMetrics stays up to record their failure, so a total outage of either is detected normally.
- **In a clustered deployment, coverage depends on the Helm chart.** A standalone PMM Server scrapes these components itself. In a cluster, VictoriaMetrics and ClickHouse run as separate workloads scraped by the chart, so these alerts only see them if the chart uses the same job names. Where a component is not scraped at all, the alert stays silent rather than firing falsely.
- **These alerts cover PMM Server only.** For a PMM Client that has stopped reporting, use *PMM agent down* instead.

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
| PBM | **MongoDB stale PBM backup** | Monitors the time of the last successful backup. If it is older than the configured threshold, it sends an alert. | MongoDB | 

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
