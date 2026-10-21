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

| Consideration | [Kubernetes single instance](HA-kubernetes-single-instance.md) | [HA Cluster](HA-clustered.md) |
|---|---|---|
| **Overview** | Kubernetes restarts PMM Server automatically if it fails. Persistent volumes keep your data across restarts. | Fast automatic failover with distributed databases. Supported on [Amazon EKS and OpenShift](HA-clustered.md#check-if-your-platform-is-supported). |
| **PMM instances** | 1 | 3 |
| **Failover time** | 2–5 min | 5–10 sec interruption on leader failover |
| **Setup complexity** | Low | High |
| **Resource overhead** | 1.2x | 3–5x |
| **Metric data during outage** | Cached on clients, resent on reconnect | Cached on clients. Brief gap during leader failover (5–10 sec) |
| **Limitations** | Monitoring stops for 2–5 minutes if the pod restarts or moves to another node | Requires advanced Kubernetes or OpenShift experience and significantly more resources |
| **Use when** | You already run Kubernetes, want automatic recovery, and can tolerate 2–5 min of downtime. | You need near-continuous monitoring, failover under 30 seconds, and have advanced Kubernetes or OpenShift experience. |

## Get started

- [Install a Kubernetes single instance](../install-pmm/install-kubernetes-single-instance.md)
- [Install PMM HA Cluster](install-HA-clustered.md)

