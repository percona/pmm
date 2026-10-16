---
name: mysql-pxc-galera
description: Percona XtraDB Cluster / Galera (PMM ships no alert template for it). wsrep flow control pauses, node desynced or donor, node or whole cluster non-Primary, cluster size drop, SST/IST, certification conflicts.
---

# Percona XtraDB Cluster / Galera (wsrep)

## When to use
Writes stall cluster-wide, a PXC node rejects queries ("WSREP has not yet prepared node"), cluster size dropped, a node is desynced/donor, deadlock errors on commit in a multi-writer setup. Node fully down → **mysql-availability** first. Async replication into or out of PXC → **mysql-replication-topology**.

## Steps
1. **Scope.** `pmm_list_services(service_type=mysql)`; PXC nodes usually share the `cluster` label. Confirm Galera and the real cluster name: `execute_prometheus_instant_query("mysql_galera_variables_info")` → label `wsrep_cluster_name` per service_name. Work on **every** node of that cluster, not only the alerting one.
2. **Dashboards** (no observability-map route for Galera): `pmm_search_dashboards('PXC')` → `pxc-node-summary` (per node) and `pxc-cluster-summary` (per cluster). Node summary panels: 51 Cluster Status, 49 Local State, 53 Desync Mode, 50 Ready, 54 Cluster Size, 39 Replication Queues, 45 Flow Control, 41 Writing Conflicts, 47 Available Downtime before SST. If an id is missing, use `pmm_list_dashboard_panels`.
3. **Snapshot** (`pmm_metrics_snapshot`, alert/user window, `{service_name=~"<n1>|<n2>|<n3>"}`, max_series = node count):
   - State: `mysql_global_status_wsrep_cluster_status` (1 = Primary), `mysql_global_status_wsrep_local_state` (4 Synced, 2 Donor/Desynced, 1 Joining, 3 Joined), `mysql_global_status_wsrep_ready`, `mysql_global_status_wsrep_cluster_size`, `mysql_global_variables_wsrep_desync`.
   - Flow control: `rate(mysql_global_status_wsrep_flow_control_paused_ns[5m]) / 1e9` (fraction of time writes were paused), `rate(mysql_global_status_wsrep_flow_control_sent[5m])` (who asks to pause), `mysql_global_status_wsrep_local_recv_queue` vs `mysql_global_status_wsrep_flow_control_interval_high`, `mysql_global_status_wsrep_local_send_queue`.
   - Conflicts / network: `rate(mysql_global_status_wsrep_local_cert_failures[5m])`, `rate(mysql_global_status_wsrep_local_bf_aborts[5m])`, `mysql_galera_evs_repl_latency_avg_seconds`, `mysql_galera_gcache_size_bytes`.
4. **Slow node resources.** For the node with the highest FC sent / recv queue, check host CPU, iowait and disk latency (**node-capacity-saturation**, `engine=node`, `intent=disk_io`), and its QAN top queries for big transactions or DDL.
5. **Error log** on nodes that changed state (`pmm_clickhouse_query`, database=otel): `ResourceAttributes['node_name'] = '<node>'` AND `Body ILIKE '%WSREP%'`, ORDER BY Timestamp DESC LIMIT 100. Look for non-Primary, partition / "evs::proto ... inactive", SST/IST start/fail, "Flow-control", TOI DDL.
6. **For the user to run** on each node: `SHOW GLOBAL STATUS WHERE Variable_name IN ('wsrep_cluster_status','wsrep_local_state_comment','wsrep_cluster_size','wsrep_last_committed','wsrep_local_recv_queue','wsrep_flow_control_paused');`

## Interpretation
- **Flow control** paused fraction > 0.1 sustained = cluster write throughput is throttled by the slowest node. The node sending FC with a high recv queue is the culprit: slow disk/CPU, too few applier threads, very large transactions, DDL in TOI, or it is a donor/backup source.
- **Desync**: local_state 2 with desync=1 = manual or backup-tool desync; with desync=0 = SST donor. local_state 1/3 = node rejoining after a restart (IST, or SST if its downtime exceeded what gcache covers).
- **Non-Primary** on one node = it lost contact with the majority (network, peer ports, partition); it refuses queries (wsrep_ready=0). Non-Primary on all nodes = quorum lost cluster-wide (e.g. 2 of 3 nodes gone at once, even split).
- **cluster_size drop** = a node left; find why on that node (availability skill).
- **Cert failures / BF aborts** rising = concurrent writes to the same rows on different nodes; applications see 1213 deadlocks at COMMIT.

## Remediation
Self-managed: SQL / shell below. k8s (PXC Operator): put config in the `PerconaXtraDBCluster` CR (`spec.pxc.configuration`) and use the Operator's crash-recovery procedure; do not bootstrap pods by hand. Run with approval.
- **Flow control**: fix the slow node's resources; more appliers `SET GLOBAL wsrep_applier_threads = <n>;` (older: `wsrep_slave_threads`); split large transactions (≈10k rows per commit); run schema changes with `pt-online-schema-change` instead of TOI at peak. Temporary relief: `SET GLOBAL wsrep_provider_options = 'gcs.fc_limit=256';` (revert later).
- **Desync left on**: `SET GLOBAL wsrep_desync = OFF;` after the backup ends. Donor/joiner: wait for SST/IST to finish; watch joiner datadir growth (`sudo du -sh /var/lib/mysql`). Frequent SST: increase `gcache.size` in my.cnf (`wsrep_provider_options="gcache.size=2G"`).
- **One node non-Primary**: restore connectivity between nodes on 4567 (group), 4568 (IST), 4444 (SST) and 3306; then restart that node (`sudo systemctl restart mysql`); it rejoins via IST/SST.
- **Whole cluster non-Primary, nodes running**: on the single node with the highest `wsrep_last_committed`: `SET GLOBAL wsrep_provider_options = 'pc.bootstrap=YES';`; the others then join it.
- **All nodes down**: pick the node with the highest `seqno` in `/var/lib/mysql/grastate.dat` (or `mysqld --wsrep-recover`), set `safe_to_bootstrap: 1` there, start it with `sudo systemctl start mysql@bootstrap.service`, then start the others normally. Never bootstrap more than one node (split brain).
- **Write conflicts**: send all writes to one node (ProxySQL/HAProxy writer hostgroup or backend), retry 1213 in the application.

## Related skills
proxysql-incidents, haproxy-incidents (writer routing), node-capacity-saturation.
