---
name: kubernetes-database-incidents
description: Databases run by Percona Operators on Kubernetes (PXC, PSMDB, PostgreSQL) — pods restarting, Pending, OOMKilled or Evicted, PVC full, Custom Resource not ready, or PMM gaps from the pmm-client sidecar. Uses kubectl read-only tools.
---

# Databases on Kubernetes (Percona Operators)

## When to use
An operator-managed database pod restarts, crash-loops, stays Pending, is OOMKilled or Evicted, runs out of storage, or its Custom Resource (CR) is not ready; or PMM shows the service down or with gaps. Needs the Kubernetes toolsets (kubernetes/core, kubernetes/logs) or `bash` with read-only kubectl. Without them, stay on PMM evidence and give the user the kubectl commands below with names filled in. Database-internal causes (slow queries, replication, locks) belong to the engine skills.

## Steps
1. **PMM first.** `pmm_list_services` → service_name, node_name. Operator pods usually register with the pod name (e.g. `cluster1-pxc-0`); confirm before you rely on it. Instant query for availability (`mysql_up`, `pg_up`, `mongodb_up`) and check restart alerts (`pmm_mysql_restarted`, `pmm_postgresql_restarted`, `pmm_mongodb_restarted`). Read the database log for that node in otel.logs.
2. **Custom Resource.** `kubectl get pxc,psmdb,perconapgcluster -A` → state and conditions (`ready`, `initializing`, `error`, `paused`). `kubectl describe <kind>/<name> -n <ns>` for the reconcile message.
3. **Pods.** `kubectl get pods -n <ns> -o wide` (filter by the cluster name): status, restarts, node, age.
4. **Each pod you name.** `kubectl describe pod <pod> -n <ns>` → events and the container's Last State (reason, exit code). Logs: `kubectl_logs` / `kubectl_previous_logs` (or `kubectl logs <pod> -c <container> --previous`) for the database container (e.g. pxc, mongod, database) and the `pmm-client` sidecar.
5. **Events.** `kubectl get events -n <ns> --sort-by=.lastTimestamp` → Evicted, FailedScheduling, FailedMount, OOMKilling, BackOff, Unhealthy.
6. **Storage.** `kubectl get pvc -n <ns>` (Bound, capacity, storage class); the engine's disk alerts (`pmm_mongodb_dbpath_disk_space`) and node disk metrics if node_exporter runs.
7. **Operator.** `kubectl logs deploy/<operator-deployment> -n <operator-ns> --since=1h` for reconcile errors.
8. **Timeline.** Line up PMM metrics (connections, QPS, memory) and otel.logs with the pod events in the same minutes.

## Interpretation
| Signal | Meaning |
|---|---|
| Last State OOMKilled, exit 137 | Memory limit below buffer pool / WiredTiger cache / shared_buffers plus connection memory |
| Evicted (kubelet event) | Node memory or disk pressure — investigate the node, not the database |
| Pending + FailedScheduling | Not enough CPU or memory, affinity or anti-affinity, or a PVC bound to another zone — say which |
| FailedMount, PVC Pending | Storage class or provisioner problem |
| CrashLoopBackOff, exit 1 | Database failed to start: read the previous log (crash recovery, config, datadir permissions) |
| SuccessfulCreate from the StatefulSet controller, new pod, Restart Count 0 | Rollout or reschedule (operator upgrade, CR change) — not damage |
| PMM gaps while the pod is healthy | pmm-client sidecar: check its log and agent status (see exporter-scrape-failures) |
| CR state error | Reconcile failure; the operator log names it |

A pod is affected only with its own evidence (event, last-state reason, log line). For many pods, up to 3 per StatefulSet are enough; report against the StatefulSet.

## Remediation
Operator deployments: change the Custom Resource, never files inside the pod (the operator reverts them). Check field names for the installed operator version with `kubectl explain <kind>.spec`.
- **Memory:** raise the component's `resources.limits.memory` (and requests) in the CR, or lower the cache size in the CR's configuration section, then `kubectl apply -f cr.yaml`.
- **Storage:** expand the volume request in the CR when the storage class has `allowVolumeExpansion: true`; otherwise shorten binlog/WAL/oplog retention through the CR configuration.
- **Scheduling:** adjust affinity, tolerations or resource requests in the CR, or add node capacity.
- **Sidecar:** fix the `pmm` section of the CR (server host, credentials secret) and let the operator restart the pod.
Write every command with the real namespace, cluster and pod names.

## Related skills
exporter-scrape-failures, node-capacity-saturation, mysql-availability, postgresql-incidents, mongodb-incidents.
