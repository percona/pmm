---
name: mongodb-incidents
description: pmm_mongodb_* availability, restart, replica set (no primary, primary changed, unusual state, replication lag, oplog window), WiredTiger read/write tickets, chunk migrations, dbPath disk, SSL cert expiry and CVE-2025-14847 zlib alerts; MongoDB outages, elections, lag, saturation.
---

# MongoDB incidents

## When to use
Any pmm_mongodb_* alert except the PBM backup ones (those belong to **pmm-backup-management**), or MongoDB symptoms: elections, members stuck, lag, queued operations, disk or certificate problems.
Slow operations, COLLSCAN or index design: use **mongodb-query-analysis**.

## Steps
1. Scope: `pmm_list_firing_alerts()` (`service_name`, `node_name`, `rs_nm`/`set`, `cluster`); `pmm_list_services(service_type=mongodb)` for every member of that replica set, not only the alerting one; `pmm_list_agents(service_id=<id>)` for `mongodb_exporter` status.
2. Metrics: `pmm_observability_map(engine=mongodb, intent=<intent>, service_id=<id>)`, then `pmm_metrics_snapshot` on the panel exprs. Add these (scope with `service_id` or `rs_nm`/`set`/`cluster`):

| Alert | intent / dashboard | Extra PromQL |
|---|---|---|
| down, restarted | availability | `mongodb_up`, `mongodb_instance_uptime_seconds` |
| replset_no_primary, primary_changed, unusual_state | replication; `mongodb-replicaset-summary` panels 1587, 1642 | `mongodb_mongod_replset_my_state` per member, `mongodb_rs_members_state` (labels `member_idx`, `member_state`) |
| replication_lag, oplog_window | replication; `mongodb-replicaset-summary` panels 1038, 1025, 1020 (flow control) | `mongodb_mongod_replset_member_replication_lag{state="SECONDARY"}`, `(mongodb_mongod_replset_oplog_head_timestamp - mongodb_mongod_replset_oplog_tail_timestamp) / 3600`, `increase(mongodb_oplog_stats_wt_cache_bytes_written_from_cache[1h])` (oplog bytes/h) |
| read_tickets, write_tickets | latency + workload; `mongodb-wiredtiger` panels 55, 40, 57 | `mongodb_mongod_wiredtiger_concurrent_transactions_available_tickets` / `_out_tickets` (by `txn_rw`), `mongodb_mongod_global_lock_current_queue` (by `type`), `mongodb_mongod_wiredtiger_cache_bytes{type="total"} / mongodb_mongod_wiredtiger_cache_max_bytes` |
| too_many_chunk_migrations | `mongodb-cluster-summary` panels 1216, 1313 | `mongodb_mongos_sharding_changelog_10min_total{event=~".*moveChunk.*"}`, `mongodb_mongos_sharding_balancer_enabled`, `mongodb_mongos_sharding_shard_chunks_total` (by `shard`) |
| dbpath_disk_space | - | `mongodb_dbstats_fsUsedSize / mongodb_dbstats_fsTotalSize`, `mongodb_dbstats_dataSize` and `mongodb_dbstats_indexSize` (by `database`) |
| host_ssl_cert_expiry | - | `(mongodb_ss_security_SSLServerCertificateExpirationDate - mongodb_date) / 86400000` (days) |
| cve_2025_14847_zlib | - | `mongodb_version_info` (label `mongodb`), `increase(mongodb_ss_network_compression_zlib_compressor_bytesIn[24h])` |

3. Logs: `pmm_clickhouse_query(database=otel)` with the node filter and `LogAttributes['log.file.name'] ILIKE '%mongo%'`. Match `Body ILIKE` on `%election%`, `%transition to%`, `%rollback%`, `%heartbeat%`, `%too stale%`, `%No space left%`, `%SSL%`, `%Fatal%`.
4. mongosh checks for the user: `rs.status()` (per member `stateStr`, `health`, `optimeDate`, `lastHeartbeatMessage`), `rs.printSecondaryReplicationInfo()`, `rs.printReplicationInfo()`, `db.currentOp({active: true, secs_running: {$gt: 5}})`, `db.serverStatus().wiredTiger.concurrentTransactions` (`queues.execution` on 7.0+), `sh.status()`.

## Interpretation
- **down**: exporter enabled but `mongodb_up` is not 1 for 1m (also fires when the exporter stops reporting). Exporter `RUNNING` with `mongodb_up=0`: mongod unreachable, or auth/TLS fails. Exporter not running: **exporter-scrape-failures**.
- **restarted**: uptime below 300 s. Ranked causes: OOM kill (node memory), crash (`Fatal assertion` in the log), pod reschedule, planned restart or upgrade.
- **replset_no_primary**: no writable member, so writes fail. Causes: voting majority unreachable (partition, members down), members in RECOVERING/STARTUP2, or only priority-0 members left.
- **primary_changed**: an election in the last 10m. Planned if it matches `rs.stepDown()` or a rolling restart; otherwise the primary crashed or stalled (CPU, I/O, swap) or heartbeats timed out (10 s default). Repeated changes mean flapping: check node saturation and network.
- **unusual_state**: a member is not PRIMARY(1), SECONDARY(2), ARBITER(7) or DOWN(8). Others: STARTUP(0), RECOVERING(3), STARTUP2(5, initial sync), UNKNOWN(6), ROLLBACK(9), REMOVED(10). RECOVERING plus `too stale` in the log: the member fell off the oplog and needs a resync. ROLLBACK: writes not replicated before a failover are being undone.
- **replication_lag**: a secondary more than 600 s behind (default). Causes: primary write bursts (oplog GB/hour rising), slower secondary disk/CPU, network, index builds or heavy reads on the secondary. Flow control activity means the majority commit point lags.
- **oplog_window**: the oplog holds less than 24 h (default). A member offline longer than the window cannot catch up and needs a full resync. If oplog GB/hour jumped, the burst shrank the window, not an undersized oplog.
- **read/write tickets**: fewer than 50 available, so operations queue and latency rises. Ranked causes: slow operations holding tickets (COLLSCANs, see mongodb-query-analysis), WiredTiger cache above ~95% full (application threads evict), checkpoint or disk stalls, too many concurrent clients. On 7.0+ the pool is dynamic: compare available with out tickets.
- **too_many_chunk_migrations**: more than 40 `moveChunk.start` in 10 minutes: balancing after a shard was added, a hot or monotonic shard key, zone changes, jumbo chunks retrying. Migrations add I/O and latency.
- **dbpath_disk_space**: the data filesystem is more than 85% used. WiredTiger does not return freed space to the OS without `compact` or a resync. The oplog has a fixed size.
- **host_ssl_cert_expiry**: fewer than 30 days left. After expiry, TLS handshakes fail between clients and members.
- **cve_2025_14847_zlib**: a vulnerable version plus zlib traffic in the last 24 h. Fixed versions: 4.4.30+, 5.0.32+, 6.0.27+, 7.0.28+, 8.0.17+, 8.2.3+. 3.6, 4.0 and 4.2 have no fix.

## Remediation
- **No primary**: restore network and the down members first; check votes and priorities in `rs.conf()`. Last resort, with approval: `rs.reconfig(cfg, {force: true})` on a survivor (can lose writes).
- **Elections**: fix the primary's stall (node-capacity-saturation) or the network; change `settings.electionTimeoutMillis` only with evidence.
- **Too stale member**: initial sync. Self-managed: stop it, empty `dbPath`, start it; Operator: delete the pod's PVC and pod. Confirm the other members are healthy first.
- **Lag**: spread bulk writes, size secondaries like the primary, no index builds at peak.
- **Oplog window**: resize online on each member (secondaries first): `db.adminCommand({replSetResizeOplog: 1, size: <MB>, minRetentionHours: 48})` (`minRetentionHours` needs 4.4+). Persist it with `replication.oplogSizeMB` (self-managed), the Operator CR, or the Atlas cluster config.
- **Tickets**: kill long operations with approval (`db.killOp(<opid>)`) and fix their indexes; size `storage.wiredTiger.engineConfig.cacheSizeGB` against RAM. Raising the ticket count is not a first fix.
- **Chunk migrations**: `sh.stopBalancer()` / `sh.startBalancer()`, or an `activeWindow` on the `balancer` document in `config.settings`. Review the shard key and jumbo chunks (`sh.status()`).
- **Disk**: grow the volume; drop unused indexes and collections; reclaim space with `db.runCommand({compact: "<coll>"})` (check its locking for your version) or a member resync. Never delete files in `dbPath`.
- **Certificate**: renew, then `db.adminCommand({rotateCertificates: 1})` (5.0+, self-managed) or a rolling restart. Operator: update the TLS secret (cert-manager renews it). Atlas manages certificates.
- **CVE zlib**: upgrade to a fixed version, or remove `zlib` from `net.compression.compressors` (mongod.conf or Operator CR) and client connection strings, then rolling restart.
