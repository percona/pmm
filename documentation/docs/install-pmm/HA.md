# Choose your HA deployment

When your database monitoring goes down, you lose visibility into critical performance issues just when you need it most. HA ensures your PMM monitoring stays online even when servers fail, networks disconnect, or hardware breaks.

Implement HA to build a resilient PMM deployment that keeps monitoring your databases no matter what happens to individual components.

## Understand what PMM HA can and can't do

Before you invest time in setting up HA for PMM, evaluate whether its benefits justify the added complexity for your specific use case.

Critical systems requiring sub-second failover gain the most value from PMM HA, while environments that can tolerate brief monitoring gaps (seconds to minutes) may find simpler solutions more appropriate. Consider your RTO requirements and incident response processes when deciding whether HA justifies the operational investment.

### What PMM HA provides

- Continuous monitoring visibility during server failures, preventing blind spots when you need observability most
- Automatic failover that restarts services or switches to backup systems without manual intervention
- Metric data preserved during brief outages, as PMM clients cache and resend metrics once connectivity resumes
- Reduced operational risk by maintaining monitoring coverage during critical incidents

### What PMM HA cannot solve

- Even with perfect HA, you'll still only detect issues after PMM's minimum one-minute alerting interval
- Complete network partitions that isolate entire segments of your infrastructure from monitoring
- Increased operational overhead since HA introduces additional complexity in deployment, maintenance, and troubleshooting

## Feature comparison

| Feature | [Single-Instance HA](../install-pmm/HA-kubernetes-single-instance.md) | [HA Cluster](../install-pmm/HA-clustered.md) |
|---------|-------------------|---------------------|
| **Kubernetes required** | Yes | Yes (Kubernetes or OpenShift) |
| **PMM instances** | 1 | 3 |
| **Failover time** | 2-5 min | < 30 sec |
| **Zero downtime** | No | Yes |
| **Setup complexity** | Low | High |
| **Resource overhead** | 1.2x | 3-5x |
| **Metric data during outage** | Cached on clients, resent on reconnect | Always available across multiple servers |

!!! tip "No Kubernetes? Use Docker with auto-restart"
    If you don't have Kubernetes, run PMM Server with `--restart always` and Docker will automatically restart it after crashes or reboots. See [Install PMM Server with Docker](install-pmm-server/deployment-options/docker/index.md).

## HA deployment options

Choose the deployment option that matches your infrastructure and requirements:

=== "Single-Instance HA"

    Enterprise-grade high availability through Kubernetes orchestration. Provides automatic pod rescheduling and persistent data across failures.

    **Key features**
    
    - Kubernetes automatically reschedules failed pods to healthy nodes
    - Persistent volumes preserve all data and configurations
    - Health probes ensure only healthy instances receive traffic
    - Production-tested and stable

    **Limitations**
    
    - 2-5 minutes monitoring interruption during pod rescheduling
    - Single PMM instance (no load distribution)

    **When to use this option**
    
    - You have Kubernetes infrastructure
    - You need production-ready HA
    - You can tolerate 2-5 minutes of downtime
    - You want automatic recovery without complexity

    [Install Single-Instance HA](../install-pmm/install-HA-kubernetes-single-instance.md){.md-button} 

=== "HA Cluster"

    Zero-downtime high availability with multiple active PMM instances, distributed databases, and automatic load balancing. Supported on Amazon EKS and OpenShift 4.21+.

    **Key features**
    
    - Zero-downtime failover (< 30 seconds)
    - 3 PMM server replicas with leader election
    - HAProxy load balancing
    - Distributed ClickHouse, VictoriaMetrics, and PostgreSQL clusters

    **What makes it different**
    
    - **No monitoring blind spots**: unlike single-instance (2-5 min gaps), clustered maintains continuous visibility
    - **Multiple active instances**: load distribution and instant failover to followers
    - **Horizontal scalability**: add replicas as monitoring load grows
    - **True zero-downtime**: Traffic redirects to healthy instances in < 30 seconds

    **Known limitations**
    
    - complex setup requiring 3 Kubernetes operators
    - 3x resource overhead minimum
    - PostgreSQL monitoring may show incorrect FAILED status

    **When to use this option**
    
    - You need continuous monitoring visibility (no 2-5 min gaps)
    - You have strict SLA requirements for sub-30-second failover
    - You need multiple active PMM instances for load distribution
    - You have expert Kubernetes or OpenShift skills
