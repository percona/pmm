# MongoDB WiredTiger Details

This dashboard shows internal WiredTiger storage engine metrics for a MongoDB instance, covering cache usage, concurrency, checkpoint behavior, write-ahead log activity, and document-level operation counts. Use it to diagnose storage engine bottlenecks when query latency or throughput problems cannot be explained by query patterns alone.

![MongoDB WiredTiger Details](../../images/PMM_MongoDB_WiredTiger_Details.jpg)

## Overview

### WiredTiger Cache Usage

Shows the current amount of data in the WiredTiger cache in bytes. This is the uncompressed in-memory representation, which is larger than the on-disk size. Compare this with **WiredTiger Max Cache Size** to see how full the cache is.

### WiredTiger Max Cache Size

Shows the configured maximum size for the WiredTiger cache. The default is 50% of available RAM minus 1 GB, with a minimum of 256 MB. To change it, set `storage.wiredTiger.engineConfig.cacheSizeGB` in the config file or pass `--wiredTigerCacheSizeGB` on the command line. If **WiredTiger Cache Usage** is consistently close to this value, the cache is under pressure and eviction will be occurring.

### Memory Cached

Shows the amount of memory in the OS filesystem cache on the host. WiredTiger stores data in its own cache, but the OS filesystem cache also holds recently accessed data files. A large filesystem cache can reduce disk reads even when the WiredTiger cache is under pressure.

### Memory Available

Shows the percentage of system memory currently available for use. Turns orange at 90% and red at 95%. If available memory is critically low, the OS may begin evicting filesystem cache pages, which will increase disk reads for data not in the WiredTiger cache.

## WiredTiger Transactions

Shows the rate of WiredTiger internal transactions per second, broken down by type: begin, commit, and rollback.

A high rollback rate relative to commit rate can indicate write conflicts under concurrent workloads. Use this alongside **Queued Operations** to understand whether contention is causing transactions to fail and retry.

## WiredTiger Cache Activity

Shows the rate of data transfer between the WiredTiger cache and storage in bytes per second. Two series are shown: **Read into** (data loaded from storage into the cache) and **Written from** (dirty data flushed from the cache to disk).

Writes from the cache always go to disk. Reads into the cache may be served from the OS filesystem cache if the data is already in RAM. A sustained high **Read into** rate means the working set is not fitting in the WiredTiger cache and data is being loaded frequently.

## WiredTiger Block Activity

Shows the rate of data handled by the WiredTiger block manager per second, broken down by operation type (read, write, map read).

The block manager is the layer below the cache that handles physical reads and writes to data files. High block write rates indicate frequent flushing of dirty cache pages. Elevated read rates mean the cache is being populated from disk, which points to working set pressure.

## WiredTiger Sessions

Shows the number of open WiredTiger internal cursors and sessions over time.

Each MongoDB operation opens one or more WiredTiger cursors. A steadily growing cursor count can indicate cursors are not being closed promptly, which may eventually cause resource exhaustion.

## WiredTiger Concurrency Tickets Available

Shows the number of available WiredTiger concurrency tickets for read (positive Y axis) and write (negative Y axis) operations over time. Each simultaneous operation in the WiredTiger engine consumes one ticket. Available tickets equal the total pool minus tickets currently in use.

When available tickets approach zero for either read or write, new operations must wait before entering the storage engine. This is one of the most direct indicators of storage engine saturation. If tickets are consistently depleted, check **Queued Operations** and consider whether the workload or hardware is the limiting factor.

## Queued Operations

Shows the number of operations waiting to acquire a global lock, broken down by read and write queues over time.

Any value above zero means lock contention is occurring. A queue that grows and stays elevated points to long-running write operations blocking other work. Use this alongside **WiredTiger Concurrency Tickets Available** to distinguish between lock queue pressure and ticket exhaustion.

## WiredTiger Checkpoint Time

Shows the time spent in the WiredTiger checkpoint phase, displayed as a per-second average of a cyclical event that runs approximately every 60 seconds by default.

Checkpoints flush dirty cache pages to disk to create a consistent on-disk snapshot. A rising trend in checkpoint time means each checkpoint is taking longer, usually because there are more dirty pages to flush or the disk cannot keep up. Long checkpoints can cause brief latency spikes for write operations. Check **Disk I/O and Swap Activity** to confirm whether disk throughput is the constraint.

## WiredTiger Cache Eviction

Shows the rate of cache page evictions per second, broken down by type (modified and unmodified pages).

Eviction happens when the cache fills up and WiredTiger needs to make room for new data. Unmodified page evictions are inexpensive. Modified (dirty) page evictions require writing to disk first and are more expensive. A sustained high rate of dirty page evictions means the cache is consistently full and cannot absorb write bursts without stalling application threads.

## WiredTiger Cache Capacity

Shows the configured maximum cache size (**Max**) alongside the current cache usage (**Used**) over time.

Use this to see how cache utilization trends over the selected time range. A **Used** line that hugs the **Max** line means the cache is always full and eviction pressure is constant. This is a signal to increase the cache size if memory allows, or to reduce the working set.

## WiredTiger Cache Pages

Shows the number of pages in the WiredTiger cache over time, broken down by state (clean, dirty, internal).

A high dirty page count relative to total pages means a large fraction of the cache contains modified data that has not yet been flushed to disk. If dirty pages stay elevated, checkpoint and eviction pressure will follow.

## WiredTiger Log Operations

Shows the rate of WiredTiger write-ahead log (WAL) operations per second, broken down by type (write, sync, read, compress, compress failure, compress uncompressed, read).

The WiredTiger WAL provides durability for writes. High sync rates indicate frequent fsync calls, which can limit write throughput on slow storage. High compress failure rates mean WAL data is not compressing well, which increases log volume.

## WiredTiger Log Activity

Shows the rate of data moved through the WiredTiger write-ahead log in bytes per second, broken down by operation type.

Rising log write rates indicate increasing write activity. If log sync bytes are high relative to log write bytes, individual writes are being fsynced frequently rather than batched, which can reduce write throughput.

## WiredTiger Log Records

Shows the rate of records appended to the WiredTiger internal log per second, broken down by type (compressed and uncompressed).

Use this alongside **WiredTiger Log Activity** to understand whether rising log volume is driven by more records or larger records.

## Document Changes

Shows the rate of document-level changes per second, broken down by operation type: `inserted`, `updated`, `deleted`, and `returned` (query results), plus replicated write operations (`repl_inserted`, `repl_updated`, `repl_deleted`) and TTL index deletions (`ttl_deleted`).

Use this to understand overall data throughput and its composition. A spike in `ttl_deleted` means a large batch of documents expired at once. High `repl_*` rates on a secondary mean replication is catching up. Compare insert and delete rates to track net data growth.

## Scanned and Moved Objects

Shows the rate of objects scanned per second, broken down into data objects (`scanned_objects`) and index entries (`scanned`). Also shows the rate of documents moved per second (`moved`).

High scan rates relative to documents returned point to collection scans that would benefit from better indexes. The `moved` metric applies to MMAPv1 only: documents are moved when they grow beyond their allocated space. If you see a non-zero `moved` rate, the instance is running MMAPv1.

## Page Faults

Shows the rate of OS memory page faults per second on the host. Page faults are not exclusive to MongoDB and can be caused by any process on the host.

For WiredTiger instances, page faults typically happen when the OS filesystem cache does not contain data that MongoDB needs to read. A sustained high rate means the working set is larger than available memory and MongoDB is reading frequently from disk. Check **Memory Available** and **Memory Cached** in the Overview to confirm the host is under memory pressure.

## MongoDB Summary

### MongoDB Uptime

Shows how long the MongoDB instance has been running since its last restart. Red means under 5 minutes, orange means under 1 hour, green means over 1 hour.

A recent restart may cause temporarily elevated cache miss rates and page faults as the WiredTiger cache warms up.

### QPS

Shows the current query rate in operations per second, excluding commands.

### Latency

Shows the average command latency in microseconds.

### Service

Links to the **MongoDB Instance Summary** for the selected service.

## Node Summary

### System Uptime

Shows how long the host has been running since last boot. Red means under 5 minutes, green means over 1 hour.

### Load Average

Shows the 1-minute load average. Turns orange at 10 and red at 20. Values above the number of CPU cores indicate the system is overloaded.

### RAM

Shows the total physical memory on the host.

### Virtual Memory

Shows total memory including swap (RAM + swap).

### Disk Space

Shows total disk capacity across all partitions. Click to open **Disk Details**. Note this value can be over-reported on systems where the same storage is counted multiple times.

### Min Space Available

Shows the lowest free disk space percentage across all partitions. Red below 5%, orange at 5%, green above 20%.

### Node

Links to the **Node Summary** dashboard for the host.

## CPU Usage

Shows CPU utilization over time as a stacked chart, broken down by mode: user, system, iowait, steal, and others.

High iowait alongside elevated WiredTiger eviction or checkpoint times confirms that disk I/O is the bottleneck. High steal values in virtualized environments mean the host is competing for CPU with other tenants.

## CPU Saturation and Max Core Usage

Shows normalized CPU load (load average divided by CPU count) and the utilization of the most-loaded CPU core over time.

A normalized load above 1.0 means processes are waiting for CPU. If the max core utilization is high while normalized load is moderate, work is concentrated on a single core, which is common for single-threaded operations or heavily skewed workloads.

## Disk I/O and Swap Activity

Shows disk read and write throughput alongside swap in and swap out activity over time. Click to open **Disk Performance**.

For WiredTiger instances, high disk reads indicate cache misses and working set pressure. High disk writes indicate checkpoint flushing or heavy write throughput. Any swap activity is a serious warning: if the host is swapping, available memory for both the WiredTiger cache and the OS filesystem cache is critically low, and performance will degrade significantly.

## Network Traffic

Shows inbound and outbound network throughput in bytes per second, excluding loopback traffic.

Unexpected spikes can indicate replication traffic, a client sending or receiving large result sets, or a backup in progress.
