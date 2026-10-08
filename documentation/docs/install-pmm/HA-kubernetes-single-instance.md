# Understand Single-Instance HA

Kubernetes provides enterprise-grade high availability through automated container orchestration, self-healing capabilities, and intelligent workload distribution. This production-ready option combines simplicity with Kubernetes' built-in resilience for automatic recovery from failures.

## What it is

Single-Instance HA leverages Kubernetes' native pod management and self-healing capabilities to ensure PMM stays available even when infrastructure fails.

Combined with persistent volumes and PMM Client caching, this approach preserves metric data during brief outages and maintains monitoring continuity with minimal operational overhead.

### Key benefits

- **Automatic recovery**: Kubernetes reschedules failed pods to healthy nodes without manual intervention
- **Persistent data**: All monitoring data, configurations, and dashboards survive pod restarts
- **Health monitoring**: Liveness and readiness probes ensure only healthy instances receive traffic
- **Metric data preserved**: PMM clients cache metrics locally during brief outages and resend them once connectivity resumes
- **Production-tested**: Stable and battle-tested in production environments for years
- **Simple operations**: Single PMM instance is easier to manage than distributed clusters

### How it works

Kubernetes watches your PMM deployment and fixes problems automatically.

If a pod crashes or a node fails, Kubernetes restarts it on a healthy node within a few minutes.

Your persistent volume keeps all your data safe and it stays attached when the pod moves. Your PMM clients cache metrics locally during the restart and resend them once PMM comes back up.

### Limitations

- **Brief monitoring gaps**: 2-5 minutes of downtime during pod rescheduling
- **Single PMM instance**: No load distribution across multiple servers
- **No zero-downtime**: Cannot maintain continuous monitoring during failures
- **Node-level delays**: Pod rescheduling takes longer than container restarts

This solution works well for production environments that can tolerate brief monitoring interruptions during automatic failover.

## Choose your deployment type

| Consideration | Single-instance | HA Cluster |
|--------------|-----------------|------------|
| **Failover time** | 2-5 minutes | Immediate |
| **Setup complexity** | ● Low | ●●● Medium |
| **Resource overhead** | 1x baseline | 3-5x baseline |
| **PMM instances** | 1 pod | 3 pods with leader election |
| **Automatic failover routing** | No | Yes (HAProxy routes to active leader) |
| **Databases** | Built-in | External clusters |

### When to consider PMM HA Cluster

Consider upgrading to [Kubernetes / OpenShift HA Cluster](HA-clustered.md) when you:

- require **zero-downtime monitoring** (< 30 second failover)
- have **expert Kubernetes or OpenShift skills** to manage complex deployments
- need **multiple active PMM instances** for load distribution

### When to use Docker instead

If you don't have Kubernetes, run PMM Server with `--restart always` and Docker handles automatic restarts after crashes or reboots. See [Install PMM Server with Docker](install-pmm-server/deployment-options/docker/index.md).

## Ready to deploy?

[Install PMM Server on Kubernetes →](install-HA-kubernetes-single-instance.md){.md-button}
