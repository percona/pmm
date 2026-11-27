---
name: node-capacity-saturation
description: pmm_node_high_cpu_load, pmm_node_low_free_memory, pmm_node_swap_filled_up, plus disk space running out, disk IO saturation or latency, high load average and network errors on a monitored host or database node.
---

# Node capacity and saturation (CPU, memory, swap, disk, network)

## When to use
The alerts above or "the host/DB server is slow, full or swapping". When the node hosts a database, finish here by naming the consumer, then go to that engine's skill for query-level work (MySQL: **mysql-query-analysis**).

## Steps
1. **Scope.** Alert label node_name → `pmm_list_nodes()` → node_id, node_type (generic, container, remote_rds, …). `pmm_list_services(node_id=<id>)` → database services on the node. Remote/RDS nodes: metrics come from the cloud provider; host shell steps do not apply.
2. **Panels.** `pmm_observability_map(engine=node, intent=<cpu_memory | disk_io | network | overview>)`; substitute `$node_name` in the returned expr. Render with `node_id` only when a graph adds evidence. Per-mountpoint space: dashboard `node-disk` (Disk Details), panel 1018 Mountpoint Usage.
3. **Snapshot** the alert window ±2h (`pmm_metrics_snapshot`, `{node_name="<n>"}` in every selector):
   - **CPU**: `avg by (mode) (rate(node_cpu_seconds_total{mode!="idle"}[5m]))` (user / system / iowait / steal / softirq split); `node_load1` against the CPU count `count(node_cpu_seconds_total{mode="idle"})`; `node_procs_running`, `node_procs_blocked`.
   - **Memory**: `node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes`; `rate(node_vmstat_pswpin[5m])`, `rate(node_vmstat_pswpout[5m])`; DB cache sizing, e.g. `mysql_global_variables_innodb_buffer_pool_size`.
   - **Swap**: `1 - node_memory_SwapFree_bytes / node_memory_SwapTotal_bytes` together with the swap-in/out rates above.
   - **Disk space**: `node_filesystem_avail_bytes{fstype!~"tmpfs|rootfs|selinuxfs|autofs|rpc_pipefs"} / node_filesystem_size_bytes{fstype!~"tmpfs|rootfs|selinuxfs|autofs|rpc_pipefs"}` by mountpoint; time to full: `predict_linear(node_filesystem_free_bytes{mountpoint="<mp>"}[6h], 86400) < 0` means full within a day.
   - **Disk IO**: `rate(node_disk_io_time_seconds_total[5m])` per device (1.0 = busy 100 %); latency `rate(node_disk_read_time_seconds_total[5m]) / rate(node_disk_reads_completed_total[5m])` and the write equivalent (`node_disk_write_time_seconds_total`, `node_disk_writes_completed_total`); queue depth `rate(node_disk_io_time_weighted_seconds_total[5m])`.
   - **Network**: `rate(node_network_receive_errs_total{device!="lo"}[5m])`, `rate(node_network_receive_drop_total{device!="lo"}[5m])` (and transmit), `rate(node_netstat_Tcp_RetransSegs[5m])`.
4. **Consumer.** For each DB service on the node pull top QAN queries for the same window (`pmm.metrics` by service_id, ordered by `SUM(m_query_time_sum)`); a spike in one fingerprint that lines up with the resource curve is the cause. PMM has no per-process CPU/memory metrics; for other processes give the user `top -o %CPU`, `ps aux --sort=-%mem | head -15`, `sudo iotop -o` (self-managed).
5. **Kernel / IO errors** (`pmm_clickhouse_query`, database=otel): `ResourceAttributes['node_name'] = '<n>'` AND `(Body ILIKE '%out of memory%' OR Body ILIKE '%killed process%' OR Body ILIKE '%I/O error%' OR Body ILIKE '%No space left%')`, ORDER BY Timestamp DESC LIMIT 50.

## Interpretation
- **pmm_node_high_cpu_load**: 100 − idle % across all CPUs (5m rate) > 80 for 5m. iowait and steal count as busy, so check modes first: user = queries; system = kernel, context switches, many connections; iowait = disk bound (use the disk signals); steal = noisy neighbour or exhausted burst credits; softirq = network packet load.
- Load average above CPU count with modest CPU use = threads blocked on IO (`node_procs_blocked` > 0).
- **pmm_node_low_free_memory**: MemAvailable < 20 % for 5m. MemAvailable already counts reclaimable page cache, so this is real pressure: oversized DB caches, too many connections × per-session buffers, co-located processes. Next stop is the OOM killer (step 5).
- **pmm_node_swap_filled_up**: swap > 80 % used for 5m. Swap used but swap-in/out ≈ 0 = stale pages, low urgency. Sustained swap-in = thrashing, queries slow down.
- Disk latency: sustained > 10–20 ms per IO on SSD/NVMe is high; utilization near 1.0 with rising latency = saturated; a flat plateau of IOPS/throughput on a cloud volume = provisioned limit or burst credits gone.
- Disk growth usual suspects: binary/relay logs (lagging replica), WAL or oplog, slow/general logs, local backups, temp tables in tmpdir, undo/ibtmp1 growth from long transactions.
- Network: rising errors/drops = NIC, driver, MTU or full ring buffers; retransmits = congestion or packet loss on the path.

## Remediation
Self-managed host commands need operator shell; k8s: change the Operator CR (resources, storage); RDS/Aurora: instance class, storage and parameter group in the provider console. PMM never executes these.
- **CPU**: fix the top queries (indexes, rewrites), lower concurrency (pool sizes, a proxy), scale CPU; steal → non-burstable instance type.
- **Memory**: shrink DB caches to fit (MySQL `SET GLOBAL innodb_buffer_pool_size = <bytes>;` online; PostgreSQL `shared_buffers` needs a restart; MongoDB `storage.wiredTiger.engineConfig.cacheSizeGB`), reduce max connections, move co-tenants, add RAM.
- **Swap** (self-managed): `sudo sysctl -w vm.swappiness=1` (persist in `/etc/sysctl.d/`); clear stale swap only when MemAvailable exceeds swap used: `sudo swapoff -a && sudo swapon -a`.
- **Disk space** (self-managed): `df -h; df -i`, `sudo du -xh <mount> | sort -h | tail -20`; `PURGE BINARY LOGS BEFORE NOW() - INTERVAL 3 DAY;` once replicas have read them; rotate or truncate oversized logs (`sudo truncate -s 0 <file>` only if not needed); move old backups; grow the volume (`sudo lvextend -r -L +50G /dev/<vg>/<lv>`, or after a cloud resize `sudo growpart /dev/<disk> <part>` then `sudo xfs_growfs <mount>` / `sudo resize2fs /dev/<part>`). k8s: expand the PVC through the CR. RDS: increase storage or enable storage autoscaling.
- **Disk IO**: remove the IO source (full scans, on-disk temp tables, unbatched writes), set `innodb_io_capacity` to what the device sustains, move to faster or provisioned-IOPS storage, split data and logs.
- **Network** (self-managed): `ip -s link show <dev>`, `ethtool -S <dev>`, check MTU consistency along the path; cloud: instance bandwidth limits.

## Related skills
mysql-availability (OOM-killed or disk-full MySQL), mysql-query-analysis.
