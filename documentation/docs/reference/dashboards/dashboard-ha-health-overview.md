# PMM HA Health Overview

Use the PMM HA Health Overview dashboard to spot failed or degraded components in your PMM HA Cluster and find the pods that need attention. It covers PMM Server, PostgreSQL, ClickHouse, VictoriaMetrics, and HAProxy, and shows which nodes currently hold the leader and primary roles.

To open the dashboard, go to **PMM HA > Overview**.

![PMM HA Health Overview Dashboard](../../images/PMM_HA_Health_Overview.png)

For each component, the dashboard compares the number of alive instances with the number expected and colors the result:

- **Green**: more than two-thirds of the expected instances are alive
- **Yellow**: between one-third and two-thirds are alive, so the component is degraded
- **Red**: one-third or fewer are alive, or PMM found no instances

The **PMM HA** badge in the side menu uses the same thresholds.

!!! note "Helm chart requirement"
    The dashboard needs the `pmm-ha` Helm chart that ships with PMM 3.10.0 or later. With an earlier chart, the **Namespace** filter and the resource and storage usage panels might be empty.

## Overview

### PMM

In HA Cluster mode, multiple PMM Server replicas (three by default) provide full redundancy through Raft consensus. Raft needs a quorum, which is a majority of replicas (`floor(N/2)+1`), to elect a leader and serve requests. If one replica fails, failover stays transparent to users as long as quorum holds, but you should investigate quickly to restore full redundancy.

If the **Leader PMM Instance** panel shows **No active instance**, no replica holds the leader role and cluster-wide tasks, such as advisor checks and scheduled backups, have stopped. Check the [**PMM Pods**](#pmm-pods) table to find the affected replicas.

### PostgreSQL

PostgreSQL stores PMM's critical data including user accounts, dashboard configurations, alerting rules, and inventory information.

If the primary fails, Patroni automatically promotes a replica within seconds, but you'll temporarily lose write capability during failover (typically under 10 seconds).

Check the [**PostgreSQL Pods**](#postgresql-pods) panel in the **PostgreSQL Pod's Status** section to identify whether the primary or a replica is down.

### ClickHouse

ClickHouse stores all Query Analytics (QAN) data. When pods fail, QAN dashboards will show incomplete query performance metrics, you won't be able to analyze slow queries during the outage, and you'll have permanent gaps in your historical query analysis data.

To tell database (`pmmdb`) nodes from Keeper nodes, check the [**ClickHouse Pods**](#clickhouse-pods) table. The dashboard doesn't show ClickHouse shard topology.

Loss of keeper nodes (which handle coordination) is more critical than loss of a single database node (which handles storage).

### VictoriaMetrics

VictoriaMetrics uses multiple components working together, and the **VictoriaMetrics Components** panel shows the health of each one:

- `vminsert` receives metrics from monitored services
- `vmselect` processes dashboard queries
- `vmstorage` stores time series data
- `vmagent` scrapes metrics from targets
- `vmauth` handles authentication

To see which components are affected, check the **VictoriaMetrics Pods** panel in the [**VictoriaMetrics Pod's Status**](#victoriametrics-pods) section.

Failures in `vminsert` or `vmstorage` are most critical as they prevent metrics collection or cause data loss.

### HAProxy

HAProxy is how you get into your PMM cluster. It routes web traffic to healthy PMM Server replicas and handles automatic failover when the leader changes.

If HAProxy pods go down, you can't access the PMM web interface, API calls will fail, and automatic failover won't work.

HAProxy is essential for accessing PMM. If it shows unhealthy, investigate immediately.

To see which HAProxy pods are down and which backend services are unavailable, check the panels in the [**HAProxy Pod's Status**](#haproxy-instances) section.

## System health metrics

### Overall System Health

The **Overall System Health** gauge counts a pod as healthy only if it's ready and hasn't restarted during the selected time range. A pod that crashed and recovered still lowers the score until the restart falls outside the time range.

A single pod failure in a well-sized cluster usually drops the gauge to 95-98%, which needs no action. Below 90%, you likely have several failures that need urgent attention.

### Pod Count

If fewer pods are running than expected, find the missing pods in the pod status tables for each component, starting with [**PMM Pods**](#pmm-pods).

### Pod Restarts

Kubernetes restarts crashed pods automatically, so a cluster can look healthy while pods keep crashing. Five or more restarts in the selected time range (red) point to repeated failures. To find the unstable pods, check the [**Pods with Restarts**](#pods-with-restarts) table.

### Pods with Restarts

A pod turns red after three or more restarts in the selected time range. Focus on the pods that restart most often, and check whether the restarts match any changes you recently made.

If a pod keeps restarting across the selected time range, it's likely stuck in a crash loop. The usual causes are a configuration mistake or too little CPU or memory for the pod.

Run `kubectl logs <pod-name>` to see what's causing the crashes.

## Resource Usage

### CPU Usage

Gradual growth is normal as your monitoring workload increases. Sudden spikes may indicate problem queries, batch jobs, or other issues.

Compare CPU usage across similar pods, such as all your `vmselect` pods. If one pod is consistently higher than the others, traffic might not be balanced properly, or that pod has an issue.

High CPU on database pods like PostgreSQL, ClickHouse, or `vmstorage` during backups or maintenance is normal and expected.

### Memory Usage

Steady or slowly growing memory usage is normal. Sudden jumps usually mean large queries, data imports, or memory leaks.

If memory usage is getting close to the limits you've set, your pods risk being killed by Kubernetes for using too much memory (called OOM kills).

If you see pods restarting and they had high memory usage right before the restart, Kubernetes killed them for exceeding their memory limit.

You'll need to either increase the memory limits or figure out why memory usage is so high. Look at query patterns, cache sizes, or other factors that might be using more memory than expected.

## Storage Usage

### PostgreSQL Storage

Plan to add storage when the gauge turns yellow (70%). Red (85%) is urgent, because PostgreSQL stops accepting writes when the volume is full.

PostgreSQL stores metadata and configuration data. While it doesn't grow as fast as metrics or QAN data, if you run out of space, all PMM operations will stop.

If you're getting close to the limit, either expand the persistent volume or clean up old data like backup history and audit logs.

### ClickHouse Storage

Plan to expand storage when the gauge turns yellow (70%). Red (85%) requires immediate action.

ClickHouse stores all your Query Analytics data, which grows based on how many queries you're running and your retention settings.

If you're running low on space, you can reduce your QAN data retention period, expand the persistent volumes, or check that your slow query log isn't capturing too many queries.

If ClickHouse runs out of storage, it stops collecting QAN data, which creates gaps in your query performance analysis that you can't recover.

### VictoriaMetrics Storage

Plan to expand storage when the gauge turns yellow (70%). Red (85%) is critical, because VictoriaMetrics stops accepting new metrics when the volume is full.

VictoriaMetrics stores all your time series metrics. How fast it grows depends on how many services and instances you're monitoring, your metrics retention period (default is 30 days), and the cardinality of the metrics you're collecting.

When storage fills up, VictoriaMetrics can't store new metrics, which creates gaps in all your PMM dashboards. Keep an eye on growth trends and plan for more capacity before you run out of space.

## Service Availability

### Service Availability by Component

Use this chart to find when a problem started and whether a component has recurring stability issues. Match availability drops to known events, such as deployments or maintenance. A healthy PMM HA deployment stays at 100% except for short drops during planned maintenance or upgrades.

## PostgreSQL Pod's Status

### PostgreSQL Pods

In a healthy PostgreSQL cluster, one instance is the Primary, the remaining instances are Replicas, and all rows show UP and Healthy.

If the Primary shows DOWN, a failover is either in progress or just completed. Check that a Replica was promoted to become the new Primary.

If you see multiple Primaries, you have a split-brain condition that needs immediate attention.

If there's no Primary at all, your cluster can't accept any writes, which is a critical failure.

Patroni handles automatic failover for you. When the Primary fails, it automatically promotes a Replica to Primary within seconds.

After a failover, verify that the new Primary is handling writes correctly.

## PMM Pod's Status

### PMM Pods

In a healthy PMM cluster, one instance is the **Leader**, the remaining instances are **Follower**, and all rows show UP. Pods that are DOWN show **No data** in the role column.

If one pod shows DOWN, identify which replica is affected and investigate the cause. If fewer than a quorum of your configured replicas (`floor(N/2)+1`) remain alive, your deployment is at serious risk. Investigate immediately.

If all configured replicas show DOWN, your entire PMM system is unavailable.

During a failover, the previous leader can still show as **Leader** while DOWN for one to two minutes, until its metrics expire. This delay is expected. The current leader is the instance that shows both UP and **Leader**.


### ClickHouse Pods

ClickHouse uses two types of pods: Keeper nodes that handle coordination and metadata, and database nodes that handle storage. Only Keeper nodes have a Leader or Follower role.

For a healthy keeper ensemble, you should see one Leader with the rest as Followers, all showing UP. If the Leader goes DOWN, a new leader should be elected automatically within seconds.

For database pods, when all are UP, your QAN queries are working normally with full capacity.

If a database pod is DOWN, you have reduced storage capacity, but QAN still works with lower performance.

The keeper ensemble must maintain quorum to keep the cluster functioning, so losing multiple keeper nodes at once is more serious than losing a database node.

### VictoriaMetrics Pods

Check for failures in `vmstorage` pods first. Each `vmstorage` pod stores a subset of your metrics data, so too many failures cause data loss. Next, check `vminsert` pods. These pods receive incoming metrics, so failures create gaps in data collection.

Failures in `vmselect` pods reduce query capacity but don't cause data loss. Down `vmagent` or `vmauth` pods affect metric scraping and authentication respectively.

## HAProxy Pod's Status

### HAProxy Instances

If one pod shows DOWN, your load balancing is still working but with reduced capacity. Remaining pods handle the full traffic load. If all pods show DOWN, you can't access PMM at all.

Investigate DOWN pods quickly to restore full redundancy and prevent a single point of failure. Check pod logs to identify whether the issue is configuration, resource limits, or connectivity problems.

### HAProxy Backend Servers

HAProxy performs continuous health checks on each backend. When a backend fails its health check, HAProxy automatically stops routing traffic to it and uses the remaining healthy backends.

Backends that show DOWN during pod restarts or brief failures are expected. HAProxy's automatic routing prevents any impact on your users.

However, if backends stay DOWN for more than a few minutes, the underlying pods aren't recovering properly and you need to investigate.

To identify why backends remain unavailable, check the pod status tables in the previous sections. Common causes include resource exhaustion, configuration errors, or failed deployments.

## Dashboard usage tips

### Filters

The **Namespace** filter lists only namespaces that run PMM HA.

### Investigation workflow

When you notice problems, follow this approach to quickly figure out what's wrong:
{.power-number}

1. Start with the **Overall System Health** gauge. If it's below 95%, something's wrong.
2. Check the component panels in the **Overview** section to see which components have problems.
3. Review **Pod Restarts** to see if pods are still crashing
4. Look at the detailed pod status tables to find which specific pods failed.
5. Check **Resource Usage** to see if pods ran out of CPU, memory, or storage.
6. Review **Service Availability** timeline to see when things started going wrong.

#### Common patterns

- **Storage full**: Red storage gauges + pods crashing = expand storage immediately
- **Resource exhaustion**: High CPU/memory + pod restarts = increase resource limits
- **Network issues**: Multiple components partially down + high restart counts = investigate cluster networking
- **Single pod failure**: One component's health percentage is below 100% but no restarts = stuck pod requiring manual intervention

## Related alert rules

PMM HA Cluster creates [built-in alert rules](../../alert/templates_list.md#pmm_ha_alerts) for failure scenarios such as a missing leader, split-brain, and quorum at risk. Configure a [contact point](../../alert/contact_points.md) to receive notifications without watching this dashboard.
