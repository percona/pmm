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

- **Kubernetes**: 1.32 or higher, because the Helm chart doesn't install on earlier versions. The [supported versions](#check-if-your-platform-is-supported) depend on your platform.
- **Helm**: 3.17.0 or higher
- **kubectl**: configured to access your cluster
- **Persistent Volume Provisioner**: available in your cluster

### Check if your platform is supported

You can deploy PMM as an HA Cluster on Amazon EKS and Red Hat OpenShift, including ROSA with hosted control planes. The Amazon EKS and OpenShift versions you can use depend on your `pmm-ha` Helm chart version, because each chart ships its own database operators.

To check that your platform version is supported, before you install or upgrade:
{.power-number}

1. Find your `pmm-ha` chart version. For a new installation, use the chart version you plan to install. For an existing installation, run `helm list -n <namespace>`.

2. In the following table, find the row for your chart version and check that your Amazon EKS or OpenShift version is between the versions listed for it. For example, chart 1.8.0 supports Amazon EKS 1.33, 1.34, 1.35 and 1.36:

    | `pmm-ha` chart version | PMM version | Amazon EKS | OpenShift |
    |------------------------|-------------|------------|-----------|
    | 1.8.0 | 3.10.0 | 1.33 to 1.36 | 4.18 to 4.22 |

    If your version is supported, continue with [Plan your resources](#plan-your-resources). If it isn't, upgrade your Amazon EKS or OpenShift cluster to a supported version, or use a chart version that supports your current version.

Other platforms, such as GKE, AKS or on-premises Kubernetes, might work but aren't tested. VMware Tanzu isn't supported.

## Plan your resources

Before installing PMM HA, ensure your Kubernetes cluster has sufficient capacity to run the distributed architecture. This section helps you calculate the resources you'll need based on your monitoring requirements.

### Minimum cluster requirements

At minimum, your cluster needs:

- **CPU**: 12+ cores
- **Memory**: 20-40 GB RAM
- **Storage**: 100+ GB with persistent volume provisioner

This baseline supports monitoring **1-10 database services** with standard retention periods. 

If you're planning a larger deployment, use the sizing guidelines below to calculate your resource needs.

### Sizing guidelines

Use this table to estimate resources based on your monitoring scale:

| Monitored services | PMM replicas | ClickHouse replicas | VictoriaMetrics storage | Total CPU | Total memory | Total storage |
|-------------------|--------------|---------------------|------------------------|-----------|--------------|---------------|
| 1-10 | 3 | 3 | 3 | 12 cores | 20 GB | 100 GB |
| 11-50 | 3 | 3 | 3 | 15 cores | 30 GB | 200 GB |
| 51-100 | 3 | 3 | 5 | 20 cores | 40 GB | 500 GB |
| 100+ | 5 | 5 | 5 | 30+ cores | 60+ GB | 1+ TB |

### Factors affecting resource usage

Your actual resource needs may vary based on:

- number of monitored database instances
- metrics resolution and retention period
- Query Analytics (QAN) volume
- number of concurrent users
- custom dashboards and queries

### Understand how resources are distributed

PMM HA spreads resource consumption across multiple components to ensure high availability. Understanding this breakdown helps you identify which components to scale as your monitoring needs grow, and where bottlenecks might occur.

| Component | CPU | Memory | Storage | Notes |
|-----------|-----|--------|---------|-------|
| **Per PMM Server pod** | 2 cores | 4 GB | Minimal | Stores configuration only |
| **HAProxy (3 replicas)** | 1 core | 2 GB | - | Routing and failover overhead |
| **ClickHouse cluster** | 3-6 cores | 8-12 GB | 50+ GB | Stores QAN data and scales with query volume |
| **VictoriaMetrics cluster** | 2-4 cores | 4-8 GB | 50+ GB | Stores metrics and scales with retention period |
| **PostgreSQL cluster** | 1-2 cores | 2-4 GB | 10 GB | Grafana metadata storage |
| **Kubernetes operators** | 1-2 cores | 2-4 GB | - | Operator management overhead |

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
