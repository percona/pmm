# MongoDB ClusterSync Details

This dashboard shows the health of Percona ClusterSync for MongoDB (PCSM), which replicates data from a source MongoDB cluster to a target cluster. It covers replication lag, change replication throughput, initial sync progress, high availability (HA) roles, and the resource usage of each PCSM process.

Use it to confirm that the target cluster keeps up with the source, and to find where a sync slows down.

![MongoDB ClusterSync Details dashboard](../../images/PMM_MongoDB_ClusterSync_Details.png)

## Prerequisites

To populate this dashboard, make sure that:

- PCSM 1.0.0 or later is running. PMM does not support earlier PCSM versions.
- Each PCSM instance is added to PMM as an external service. For the steps, see [Monitor Percona ClusterSync for MongoDB](../../install-pmm/install-pmm-client/connect-database/external.md#monitor-percona-clustersync-for-mongodb).

If PMM monitors at least one MongoDB service, open the dashboard from **MongoDB > ClusterSync**. Otherwise, go to **All dashboards > Browse all dashboards** and open **MongoDB ClusterSync Details** in the **MongoDB** folder.

## Filters

Use the **Environment**, **Cluster**, and **Service Name** filters at the top of the dashboard to choose which PCSM instances to show. If you add every instance of one HA group with the same `--cluster` value, select that cluster to see the whole group.

## Panels that show only the active instance

In an HA group, only the ACTIVE instance replicates data. The lag, change replication, and initial sync panels show the ACTIVE instance only. The **High Availability** and **Process** rows show every instance.

## Overview

The **Overview** row summarizes the health of the sync:

- **Exporter Up**: Shows whether PMM can collect metrics from each PCSM instance. **DOWN** means that PCSM is not running or its metrics endpoint is not reachable. **unknown** means that PMM has no data for the instance, for example because its host or PMM Client is down. Click a value to open the **Node Summary** dashboard for that host.
- **Active Instance**: Shows the PCSM instance that holds the HA lease and runs the sync.
- **Replication Lag**: Shows how far the target is behind the source, in logical seconds. PCSM also reports `0` before replication starts.
- **Initial Sync Lag**: Shows how far the initial sync is from catching up with the source, in logical seconds.
- **Lag Time**: Shows replication lag and initial sync lag over time.
- **Change Events Rate**: Shows change events read from the source and applied to the target per second. **Read** can stay above **Applied** when PCSM skips events, for example for excluded namespaces, so use the queue panels in the **Change Replication** row to confirm a backlog.

## High Availability

The **High Availability** row shows how the PCSM instances of an HA group share the work:

- **Instance Roles**: Shows the role of each PCSM instance over time. The ACTIVE instance replicates data, and STANDBY instances wait to take over.
- **Lease Term**: Shows the current HA lease term for each cluster. The term increases every time a different instance takes over.
- **Role Transitions**: Shows how many times each instance changed its role in the selected time range. An instance that starts and becomes ACTIVE counts one transition.

## Change Replication

The **Change Replication** row shows how fast the ACTIVE instance applies changes to the target:

- **Dispatcher Queue**: Shows events waiting in the queue between the change stream reader and the dispatcher that hands them to the replication workers.
- **Worker Event Queues** and **Worker Bulk Queues**: Show events and bulk writes waiting in the replication workers, as a total and for the fullest single worker. A growing queue means that the target cannot absorb writes as fast as they arrive.
- **Worker Events Applied**: Shows events applied per second by all replication workers.
- **Flush Duration (p99)** and **Flush Batch Size (p99)**: Show the 99th percentile duration and size of bulk writes to the target. Rising flush duration points to a slow target. Both panels show no data while the source has no writes.

The collapsed **Per-Worker Detail** row shows the event queue, events applied, and flush duration of individual replication workers. It shows one set of panels for each selected service, with the 10 busiest workers in each panel. PCSM starts one worker per CPU core by default.

## Initial Sync

The **Initial Sync** row shows the progress of the initial clone of the source data:

- **Copy Progress**: Shows the share of the estimated data size that PCSM has copied to the target. The value is accurate only for an uninterrupted clone, because PCSM restarts the counters when it restarts or fails over.
- **Copied vs Estimated Size**: Shows bytes inserted into the target against the estimated total size.
- **Copy Throughput**: Shows bytes per second read from the source and inserted into the target.
- **Documents Copied per Second**: Shows documents read from the source and inserted into the target.
- **Batch Duration**: Shows the duration of the latest read batch and insert batch.

The copy counters belong to each PCSM process. If an instance becomes ACTIVE after a restart or failover, this row shows no clone data for it.

## Process

The collapsed **Process** row shows CPU, resident memory, file descriptors, goroutines, threads, Go heap, and garbage-collection pauses for each PCSM process.
