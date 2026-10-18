# Understand PMM on a Kubernetes single instance

A Kubernetes single instance runs PMM Server as one pod. If it fails or the node it runs on goes down, Kubernetes restarts it on a healthy node automatically. 

Monitoring is interrupted for 2–5 minutes during recovery. PMM Clients cache metrics locally during that time and resend them once PMM comes back.

If you don't have Kubernetes, run PMM Server with `--restart always` and [Docker will restart it automatically after crashes or reboots](install-pmm-server/deployment-options/docker/index.md).

### Key benefits

Running PMM Server on Kubernetes gives you automatic recovery without the complexity of an HA Cluster:

- Kubernetes reschedules failed pods to healthy nodes without manual intervention
- All monitoring data, configurations, and dashboards survive pod restarts
- Liveness and readiness probes ensure only healthy instances receive traffic
- Easier to manage than a distributed HA Cluster

### How it works

Kubernetes watches your PMM deployment and fixes problems automatically. If a pod crashes or a node fails, Kubernetes restarts it on a healthy node within a few minutes.

Your persistent volume keeps all your data safe and it stays attached when the pod moves. Your PMM clients cache metrics locally during the restart and resend them once PMM comes back up.

### Limitations

A Kubernetes single instance has no redundancy and no load distribution:

- 2–5 minutes of monitoring downtime during pod rescheduling
- Pod rescheduling takes longer than a simple container restart
- Single PMM instance handles all traffic with no load distribution

This option is suitable for production environments that can tolerate a brief monitoring gap during recovery.

## Choose your deployment type

If a 2–5 minute recovery window is acceptable, a Kubernetes single instance is the simpler choice. For a full comparison with PMM HA Cluster, see [Choose your HA deployment](HA.md).

## Get started

- [Install Kubernetes single instance](install-kubernetes-single-instance.md)
- [Install HA Cluster](install-HA-clustered.md)
