# Understand PMM High Availability Cluster

!!! danger "VictoriaMetrics limitations"
    PMM HA Cluster does not support:
    
    - **Prometheus data imports**: Cannot import existing Prometheus files
    - **Metrics downsampling**: No automatic historical data optimization
    
    If your strategy requires these features, evaluate carefully before testing.
 
Standard PMM monitoring goes offline for minutes during server failures. PMM HA Cluster keeps monitoring running by immediately switching to secondary servers when failures occur.


PMM HA Cluster keeps your database monitoring running continuously, even when servers fail or during maintenance windows.

Unlike [Single-Instance HA](../install-pmm/HA-kubernetes-single-instance.md) where a server failure means minutes of monitoring downtime, PMM HA Cluster immediately fails over to secondary servers, keeping your monitoring online.

Whether a server crashes, you're upgrading software, or scaling your infrastructure, your monitoring stays active with no blind spots or missed incidents.

### Key benefits

- **Continuous monitoring**: Three PMM server replicas ensure monitoring never stops, even during failures
- **Fast automatic failover**: Traffic immediately switches to healthy servers with no data loss
- **Automatic traffic routing**: HAProxy routes traffic to the active leader and handles failover transparently
- **Resilient data storage**: Distributed databases (ClickHouse, VictoriaMetrics, PostgreSQL) eliminate single points of failure
- **Scales with your needs**: Add more capacity as your database infrastructure grows

### Choose your Kubernetes deployment type

| Consideration | Single-instance | HA Cluster |
|--------------|----------------------|-----------------------------|
| **Production status** | ✅ Production-ready | ✅ Production-ready |
| **Failover time** | 2-5 minutes | Immediate |
| **Setup complexity** | ● Low | ●●● Medium |
| **Resource overhead** | 1x baseline | 3-5x baseline |
| **Setup time** | ~5 minutes | ~20 minutes |
| **PMM instances** | 1 pod | 3 pods with leader election |
| **Automatic failover routing** | No | Yes (HAProxy routes to active leader) |
| **Databases** | Built-in | External clusters |
| **Anti-affinity** | No | Yes (pods distributed across nodes) |

## Before you begin

### Check prerequisites

- **Kubernetes**: 1.32 or higher
- **Helm**: 3.17.0 or higher
- **kubectl**: configured to access your cluster
- **Persistent Volume Provisioner**: available in your cluster

### Check if your platform is supported

!!! info "Supported platforms"
    PMM HA Cluster has been validated on **Amazon EKS (Kubernetes 1.32+)** and **OpenShift 4.21+** (including ROSA HCP). Other platforms (GKE, AKS, on-premise) may work but are untested. VMware Tanzu is not supported.

## Plan your resources

Choose the size of your Kubernetes cluster based on how many nodes you plan to monitor. Count each host that runs one database service with Query Analytics turned on as one node.

If you run three Kubernetes worker nodes and keep data for the default 30 days, use the following sizes. Each size handles the number of monitored nodes in its row:

| Monitored nodes | Helm values to use | Each of the 3 worker nodes | Total storage |
|-----------------|--------------------|----------------------------|---------------|
| Up to 100 | The chart's default values | 16 CPUs, 32 GB memory | About 0.5 TB |
| Up to 500 | `examples/values-500-nodes.yaml` | 32 CPUs, 64 GB memory | About 1.6 TB |
| Up to 1,000 | `examples/values-1000-nodes.yaml` | 48 CPUs, 96 GB memory | About 3.2 TB |

For more than 100 nodes, pass the matching values file from the [`pmm-ha` chart](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm-ha) when you install, for example `helm install pmm-ha percona/pmm-ha --namespace pmm -f values-500-nodes.yaml`. These sizes also need a storage class that allows volume expansion (`allowVolumeExpansion: true`).

Choose a larger size than your node count suggests if:

- you monitor MySQL with per-table statistics, or MongoDB with many collections. These produce two to three times more metrics per node. To reduce them, [turn off table statistics](install-pmm-client/connect-database/mysql/improve_perf.md#disable-table-statistics).
- you keep data for longer than 30 days. Storage grows with retention, for example 90 days needs about three times the metrics storage.

## Learn the architecture

The PMM HA architecture diagram below shows how components interact and communicate. 

The architecture consists of:

- three PMM server replicas with automatic leader election
- HAProxy for routing traffic to the active leader and handling failover
- operator-managed database clusters (ClickHouse, VictoriaMetrics, and PostgreSQL) for resilient data storage

![HA Clustered diagram](../images/HA-diagram.jpg)
### How operators manage databases

PMM HA requires three Kubernetes operators to manage distributed databases:

- VictoriaMetrics Operator: Manages metrics storage
- Altinity ClickHouse Operator: Manages query analytics data
- Percona PostgreSQL Operator: Manages Grafana metadata

These operators handle scaling, failover, and replication automatically. 

When you deploy PMM HA, the operators keep the databases healthy and available, which is how PMM survives node failures and recovers immediately.

You'll need to install these operators before deploying PMM HA.

### Learn high availability mechanisms
PMM HA uses several mechanisms to ensure continuous operation:

- **Leader election**: PMM servers use Raft consensus protocol for leader election (ports 9096, 9097)
- **Automatic failover**: HAProxy detects when the active leader becomes unhealthy and routes traffic to the new leader
- **Pod anti-affinity**: Kubernetes scheduler distributes components across different nodes
- **Health checks**: Comprehensive readiness and liveness probes on all components
- **Rolling updates**: Upgrades restart each pod sequentially. Expect a brief interruption (around 5-10 seconds) when the active leader pod restarts and a new leader is elected.

- Exporters in pull mode get no metrics, because nothing in the HA deployment scrapes them. Exporters on PMM Client nodes use push mode by default, so this affects only those set to pull mode, for example with `--metrics-mode=pull`. To switch an exporter to push mode, run `pmm-admin inventory change agent <AGENT_TYPE> <AGENT_ID> --push-metrics` (see [`pmm-admin inventory change agent`](../use/commands/pmm-admin/inventory.md#pmm-admin-inventory-change-agent)).
- **[PMM-14680](https://perconadev.atlassian.net/browse/PMM-14680)**: PostgreSQL database instance nodes and services display with an incorrect 'pmm-' prefix in their names.
- **[PMM-14734](https://perconadev.atlassian.net/browse/PMM-14734)**: HA cluster status always displays as "Healthy" even when follower or leader pods are deleted or not ready.
- **[PMM-14742](https://perconadev.atlassian.net/browse/PMM-14742)**: The Inventory page shows inconsistent numbers of PostgreSQL services (4-5 instead of the expected 6 services).
- **[PMM-14787](https://perconadev.atlassian.net/browse/PMM-14787)**: Data retention settings are not applied correctly, allowing metrics older than the configured retention period to remain visible.

## Ready to deploy?

Now that you understand how PMM HA Cluster works, you can deploy it on your Kubernetes cluster. 

The installation process uses Helm to set up all three replicas, configure HAProxy load balancing, and deploy the distributed databases automatically.

[Install PMM HA Cluster →](../install-pmm/install-HA-clustered.md){.md-button}

## Need to upgrade?

When a new PMM release is available, see [Upgrade PMM HA Cluster using Helm](../pmm-upgrade/upgrade_helm_ha.md) for the step-by-step procedure.
