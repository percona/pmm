# Choose your HA deployment

When your database monitoring goes down, you lose visibility into critical performance issues just when you need it most. HA ensures your PMM monitoring stays online even when servers fail, networks disconnect, or hardware breaks.

Implement HA to build a resilient PMM deployment that keeps monitoring your databases no matter what happens to individual components.

## Understand what PMM HA can and can't do

Before you invest time in setting up HA for PMM, evaluate whether its benefits justify the added complexity for your specific use case.

PMM HA is worth the added complexity when your monitoring must recover in seconds, not minutes. If a brief monitoring gap doesn't affect your ability to respond to incidents, a simpler deployment is enough.

### What PMM HA provides

- Continuous monitoring visibility during server failures, preventing blind spots when you need observability most
- Automatic failover that restarts services or switches to backup systems without manual intervention
- Metric data preserved during brief outages, as PMM clients cache and resend metrics once connectivity resumes
- Reduced operational risk by maintaining monitoring coverage during critical incidents

### What PMM HA cannot solve

- **Alerting delay**: PMM still detects issues only after its minimum one-minute alerting interval
- **Network partitions**: HA can't monitor infrastructure segments that are completely cut off from PMM
- **Operational overhead**: HA adds complexity to deployment, maintenance, and troubleshooting

## Feature comparison

| Feature | [Kubernetes single instance](../install-pmm/kubernetes-single-instance.md) | [HA Cluster](../install-pmm/HA-clustered.md) |
|---------|-------------------|---------------------|
| **Kubernetes required** | Yes | Yes (Kubernetes or OpenShift) |
| **PMM instances** | 1 | 3 |
| **Failover time** | 2–5 min | 5–10 sec interruption on leader failover |
| **Setup complexity** | Low | High |
| **Resource overhead** | 1.2x | 3–5x |
| **Metric data during outage** | Cached on clients, resent on reconnect | Cached on clients. Brief gap during leader failover (5–10 sec) |

!!! tip "No Kubernetes? Use Docker with auto-restart"
    If you don't have Kubernetes, run PMM Server with `--restart always` and Docker will automatically restart it after crashes or reboots. See [Install PMM Server with Docker](install-pmm-server/deployment-options/docker/index.md).

## HA deployment options

Choose the deployment option that matches your infrastructure and requirements:

=== "Kubernetes single instance"

    Kubernetes restarts PMM Server automatically if it fails, and persistent volumes keep your data across restarts.

    **Key features**
    
    - Kubernetes automatically reschedules failed pods to healthy nodes
    - Persistent volumes preserve all data and configurations
    - Health probes ensure only healthy instances receive traffic
    - Supported for production use

    **Limitations**
    
    - 2–5 minutes monitoring interruption during pod rescheduling
    - Single PMM instance

    **When to use this option**
    
    Choose this option if you:

    - already run Kubernetes infrastructure
    - want PMM Server to recover from failures automatically
    - can tolerate 2–5 minutes of downtime during recovery

    [Install a Kubernetes single instance](../install-pmm/install-kubernetes-single-instance.md){.md-button} 

=== "HA Cluster"

    High availability with fast automatic failover and distributed databases. Supported on [Amazon EKS and OpenShift](HA-clustered.md#check-if-your-platform-is-supported).

    **Key features**
    
    - Failover in 5–10 seconds when the leader pod restarts
    - Leader election across 3 PMM server replicas, with only the active leader handling traffic
    - HAProxy routes all traffic to the active leader automatically
    - Distributed ClickHouse, VictoriaMetrics, and PostgreSQL clusters

    **What makes it different**
    
    - **Shorter outages**: unlike a Kubernetes single instance (2–5 min), failover takes about 5–10 seconds
    - **Automatic failover routing**: HAProxy switches to the new leader without manual intervention
    - **Higher resilience**: additional replicas provide more standby nodes, reducing the risk of losing quorum

    **Known limitations**
    
    - Complex setup requiring 3 Kubernetes operators
    - 3x resource overhead minimum
    - PostgreSQL monitoring may show incorrect FAILED status

    **When to use this option**

    Choose this option if you:
    
    - need near-continuous monitoring (seconds of interruption, not minutes)
    - have strict SLAs that require failover in under 30 seconds
    - need automatic leader failover across multiple PMM Server replicas
    - have advanced Kubernetes or OpenShift experience
