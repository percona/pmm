---
name: valkey-redis-incidents
description: pmm_valkey_down, pmm_redis_down; Valkey/Redis outages and restarts, memory near maxmemory and evictions, connected or rejected clients, replication link down or lagging, slowlog and command latency, RDB/AOF persistence failures.
---

# Valkey / Redis incidents

## When to use
`pmm_valkey_down` or `pmm_redis_down`, or Valkey/Redis symptoms: OOM or MISCONF errors, evictions, connection refusals, latency spikes, replicas resyncing.

## Steps
1. Scope: `pmm_list_services(service_type=valkey)`. Redis instances are registered with the same type and the same `valkey_exporter`. `pmm_list_agents(service_id=<id>)` gives the exporter status.
2. Metrics: `pmm_observability_map(engine=valkey, intent=<intent>, service_id=<id>)`, then `pmm_metrics_snapshot` on its exprs. Extra PromQL (scope with `service_id="<id>"`):

| Symptom | intent | Extra PromQL |
|---|---|---|
| down, restart | availability | `redis_up`, `redis_uptime_in_seconds`, `redis_instance_info` |
| memory, evictions | memory | `redis_memory_used_bytes / redis_memory_max_bytes` (max 0 = no limit), `rate(redis_evicted_keys_total[5m])`, `rate(redis_keyspace_misses_total[5m])`, `redis_config_key_value{key="maxmemory-policy"}`, `redis_db_keys_expiring` vs `redis_db_keys` |
| clients | connections | `redis_connected_clients` vs `redis_config_maxclients`, `redis_blocked_clients`, `redis_client_recent_max_output_buffer_bytes`, `redis_evicted_clients_total` |
| latency | latency, workload | `redis_latency_percentiles_usec{quantile="99.9"}` (by `cmd`), `topk(10, rate(redis_commands_duration_seconds_total[5m]))` (by `cmd`), `rate(redis_errors_total[5m])` |
| slow commands | slow_queries | `redis_slowlog_length`, `redis_config_value{key="slowlog-log-slower-than"}` |
| replication | replication | `redis_connected_slaves`, `redis_master_repl_offset - on(service_name) group_left(slave_ip, slave_port) redis_connected_slave_offset_bytes` (bytes behind, per replica), `redis_replica_resyncs_full`, `redis_replica_partial_resync_denied`, `redis_replication_backlog_bytes` |
| persistence | none: dashboard `valkey-persistence-details` | `redis_rdb_last_bgsave_status`, `redis_aof_last_write_status`, `redis_aof_delayed_fsync`, `time() - redis_rdb_last_save_timestamp_seconds`, `redis_rdb_changes_since_last_save`, `redis_aof_last_cow_size_bytes` |
| cluster mode | none: dashboard `valkey-cluster-details` | `redis_cluster_state`, `redis_cluster_slots_fail`, `redis_cluster_slots_pfail` |

   Rejected-connection and replica link-status series are not on the PMM dashboards. Look them up with `pmm_list_metric_names(prefix="redis_master_")` and `pmm_list_metric_names(prefix="redis_rejected")`, and use only the names that come back.
3. Node context: memory and swap on the node (`pmm_observability_map(engine=node, intent=cpu_memory)`). Valkey forks for RDB and AOF rewrites and needs copy-on-write headroom.
4. Commands for the user (`valkey-cli`, or `redis-cli` for Redis): `INFO memory`, `INFO clients`, `INFO stats` (`rejected_connections`, `evicted_keys`, `latest_fork_usec`), `INFO replication` (`master_link_status`, `master_last_io_seconds_ago`), `INFO persistence`, `SLOWLOG GET 20`, `LATENCY LATEST`, `LATENCY DOCTOR`, `MEMORY DOCTOR`, `CLIENT LIST`.

## Interpretation
- **down** (both templates): the exporter is enabled but `redis_up` is not 1 for 1m, so the two alerts can fire together for one outage. Exporter `RUNNING` with `redis_up=0` means the server is down or unreachable, AUTH/ACL/TLS changed, `maxclients` is reached (the exporter cannot connect either), or the server is busy (long Lua script or command, fork on a large dataset). An uptime reset means a restart: OOM kill, crash, or a planned restart. Exporter not running is a monitoring fault: use **exporter-scrape-failures**.
- **Memory**: used near `maxmemory` with `noeviction` makes writes fail with OOM errors. With an `allkeys-*`/`volatile-*` policy it causes evictions and a rise in misses. `volatile-*` with few expiring keys behaves like `noeviction`. No `maxmemory` (max 0) lets the node OOM killer end the process; a fork can need up to the dataset size again in copy-on-write memory.
- **Clients**: connected clients near `maxclients` means new connections are rejected. Blocked clients come from BLPOP/BRPOP/WAIT-style commands. Growing output buffers mean slow consumers (pub/sub, MONITOR, big replies) and can trigger client eviction or memory pressure.
- **Latency and slowlog**: commands run on one thread, so one O(N) command stalls everyone: `KEYS`, `SMEMBERS`/`HGETALL`/`LRANGE 0 -1` on big keys, `DEL` of big keys, `FLUSHALL`, heavy Lua. Correlate a slowlog rise with per-`cmd` duration. Other causes: fork latency (`latest_fork_usec`), node swapping, transparent huge pages.
- **Replication**: a growing offset gap means the replica cannot keep up or the link is down. Rising `resyncs_full` or `partial_resync_denied` means the backlog is too small for the outage or write rate, or replicas restart. Each full sync forks the primary and sends a full RDB over the network.
- **Persistence**: `rdb_last_bgsave_status=0` means BGSAVE failed (disk full, permissions, fork ENOMEM). With `stop-writes-on-bgsave-error yes` (the default), writes then fail with MISCONF. `aof_last_write_status=0` means AOF writes fail (disk). A rising `aof_delayed_fsync` means slow disk under `appendfsync everysec`.

## Remediation
- **Memory**: set a limit and a policy, leaving fork headroom (about 60 to 75% of RAM when persistence is on): `CONFIG SET maxmemory 6gb`, `CONFIG SET maxmemory-policy allkeys-lru`, then `CONFIG REWRITE` (self-managed). Find big keys on a replica with `valkey-cli --bigkeys` / `--memkeys`. Add TTLs and split large keys.
- **Clients**: fix pool sizing or connection leaks in the application. Raise `maxclients` (bounded by `ulimit -n`). Set `timeout` for idle clients, and `client-output-buffer-limit` for pub/sub and replica classes.
- **Latency**: replace `KEYS` with `SCAN`, use `UNLINK` instead of `DEL`, enable `lazyfree-lazy-eviction` / `lazyfree-lazy-expire`, and break up big keys. Self-managed: disable transparent huge pages and avoid swap. Capture `SLOWLOG GET 50` before `SLOWLOG RESET`.
- **Replication**: raise `repl-backlog-size` (for example `CONFIG SET repl-backlog-size 512mb`) to cover outages, check network and replica `INFO replication`, and stagger replica restarts.
- **Persistence**: free disk on the data directory. Self-managed: set `sysctl vm.overcommit_memory=1` so forks do not fail. Retry with `BGSAVE` and check `INFO persistence`. `CONFIG SET stop-writes-on-bgsave-error no` is only an emergency measure, accepted together with the risk of losing data.
- **Down**:
  - Self-managed: `systemctl status valkey` (or `redis`), `journalctl -u valkey --since '-1h'`, `dmesg -T | grep -i oom`.
  - Kubernetes: `kubectl describe pod` and `kubectl logs --previous`.
  - Managed services (ElastiCache, MemoryDB): the provider console. `CONFIG` is restricted there; use the parameter group.

## Related skills
node-capacity-saturation (node memory, swap, disk), exporter-scrape-failures (exporter down).
