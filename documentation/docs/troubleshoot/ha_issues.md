# Troubleshoot PMM HA Cluster issues

Use this page when a PMM High Availability (HA) Cluster is unreachable, has no leader, or has stopped collecting data. For problems that aren't specific to HA, such as dashboards, agents, or queries on an otherwise healthy server, see [Troubleshoot PMM](index.md).

!!! note "Default values used in examples"
    The examples assume the default namespace `pmm` and release name `pmm-ha`, which gives three PMM Server pods: `pmm-ha-0`, `pmm-ha-1`, and `pmm-ha-2`. If you used different values, adjust the commands accordingly.

## Before you begin

Set these variables so you can copy the commands as-is. The `curl` examples use `-k` to accept self-signed certificates; remove it if your PMM Server has a trusted certificate.

```sh
export PMM_URL="https://<your-pmm-address>"
export PASSWORD="<admin-password>"
```

## Check cluster health

Start with these three commands. Together they show whether the pods are running, whether HA is enabled, and which node is the leader:

```sh
# 1. Are all PMM Server pods running and ready?
kubectl get pods -n pmm

# 2. Is HA enabled?
curl -sk -u admin:$PASSWORD "$PMM_URL/v1/ha/status"

# 3. Which node is the leader, and are all nodes healthy?
curl -sk -u admin:$PASSWORD "$PMM_URL/v1/ha/nodes"
```

How to read the results:

- **Pods:** All three `pmm-ha-*` pods should show `Running` with all containers ready (for example, `1/1`). Pods in `Pending`, `CrashLoopBackOff`, or with restarts that keep climbing need attention first.
- **`/v1/ha/status`:** Returns `{"status":"Enabled"}` when HA is active. Any other value means the server is not running in HA mode.
- **`/v1/ha/nodes`:** Lists each node with its role (`NODE_ROLE_LEADER` or `NODE_ROLE_FOLLOWER`) and status. A healthy cluster has exactly one leader, and every other node is a follower. To spot a missing node, compare the number of entries in the list with `expected_nodes`. That field is the replica count PMM was installed with, not a live count, so it stays the same even when a node is down.
```

## Find the logs for each component

PMM Server and the components the operators manage store their logs in different places.

| Component | Pods | How to read the logs |
|-----------|------|----------------------|
| **PMM Server** | `pmm-ha-0`, `-1`, `-2` | Files under `/srv/logs/` inside the pod (see below) |
| **HAProxy** | `pmm-ha-haproxy-*` | `kubectl logs <pod> -n pmm` |
| **ClickHouse** | `pmm-ha-pmmdb-0-0-0`, `-0-1-0`, `-0-2-0` | `kubectl logs <pod> -n pmm -c clickhouse` |
| **ClickHouse Keeper** | `pmm-ha-keeper-keeper-0-*-0` | `kubectl logs <pod> -n pmm` |
| **VictoriaMetrics** | `vmstorage-*`, `vmselect-*`, `vminsert-*` | `kubectl logs <pod> -n pmm` |
| **PostgreSQL** | `pmm-ha-pg-db-instance1-*-0` | `kubectl logs <pod> -n pmm` |
| **Operators** | `pmm-operators-*` | `kubectl logs <pod> -n pmm` |

Pod names that contain a random suffix change on every deployment. Find them by label instead of copying them:

```sh
kubectl get pods -n pmm -l app.kubernetes.io/component=pmm-server
kubectl get pods -n pmm -l clickhouse.altinity.com/chi=pmm-ha
kubectl get pods -n pmm -l app.kubernetes.io/name=haproxy
```

Always collect namespace events as well. Most installation problems are only visible there:

```sh
kubectl get events -n pmm --sort-by=.metadata.creationTimestamp
```

## PMM Server logs show only startup messages

`kubectl logs` on a PMM Server pod returns `supervisord` output such as `INFO success: grafana entered RUNNING state`, and none of the messages you are looking for.

**Cause:** PMM Server processes write to log files on the `/srv` volume instead of to the container output.

**Fix:** Read the log files inside the pod:

```sh
kubectl exec -n pmm pmm-ha-0 -c pmm-ha -- tail -100 /srv/logs/pmm-managed.log
```

`/srv/logs/` also contains `grafana.log`, `nginx.log`, `pmm-agent.log`, `qan-api2.log`, `vmalert.log`, `vmproxy.log`, and `supervisord.log`.

!!! note ""
    `/srv` is a persistent volume, so these logs survive a pod restart and contain messages from earlier runs. Check the timestamps before concluding that a message relates to the problem you are investigating.

## Cannot log in with the admin password

Every login attempt fails with 401 and the message `Invalid username or password`, even though you are using the password from your installation.

**Cause:** Two different problems produce the same error:

- Grafana's admin password is out of sync with the password stored in the `pmm-secret` secret.
- Grafana has locked the admin user after five failed login attempts.

**Fix:** First clear the failed login attempts, then try again:

```sh
kubectl exec -n pmm statefulset/pmm-ha -c pmm-ha -- bash -c '
  H="${GF_DATABASE_HOST%%:*}"; P="${GF_DATABASE_HOST##*:}"
  PGPASSWORD="$GF_DATABASE_PASSWORD" psql -h "$H" -p "$P" -U "$GF_DATABASE_USER" \
    -d "$GF_DATABASE_NAME" -c "DELETE FROM login_attempt;"'
```

If the login still fails, reset Grafana's admin password to the value in the secret:

```sh
kubectl exec -n pmm statefulset/pmm-ha -c pmm-ha -- bash -c 'printf %s "$PMM_ADMIN_PASSWORD" | grafana-cli \
  --homepath /usr/share/grafana --config /etc/grafana/grafana.ini \
  --configOverrides "cfg:database.type=$GF_DATABASE_TYPE cfg:database.host=$GF_DATABASE_HOST cfg:database.name=$GF_DATABASE_NAME cfg:database.user=$GF_DATABASE_USER cfg:database.password=$GF_DATABASE_PASSWORD cfg:database.ssl_mode=$GF_DATABASE_SSL_MODE" \
  admin reset-admin-password --password-from-stdin'
```

!!! warning ""
    Pass the database settings with `--configOverrides` as shown. Without them, `grafana-cli` writes to a local database that the running Grafana does not use, reports `Admin password changed successfully`, and changes nothing.

## Pods stay in Pending state

One or more pods never start and remain `Pending`.

**Cause:** The cluster does not have enough CPU or memory, or a volume cannot be provisioned.

**Fix:** Check why the scheduler rejected the pod:

```sh
kubectl describe pod <pod-name> -n pmm
kubectl get pv,pvc -n pmm
```

Add a node, or reduce the resource requests in your values file. If you reduce the requests, delete the Pending pod manually. A `StatefulSet` does not recreate a pending pod when the template changes, so it keeps its original request.

## PMM returns errors for a few seconds after a pod restarts

Requests fail with 5xx errors for a short time after a PMM Server pod is restarted or deleted, then recover on their own.

**Cause:** HAProxy sends all traffic to the leader only. It checks `/v1/server/leaderHealthCheck`, which returns 200 on the leader and 400 on followers. While a new leader is being elected, no pod passes the check.

**Fix:** This is expected. Wait for the election to finish and retry. If a page was open when this happened, reload it. The browser can keep showing the request that failed during the election.

## PMM is permanently unreachable

Requests keep failing and do not recover.

**Fix:** Check that the HAProxy service has endpoints:

```sh
kubectl get endpoints pmm-ha-haproxy -n pmm
```

If the endpoint list is empty, no pod is passing the leader health check. Ask each pod directly which one is the leader:

```sh
for p in pmm-ha-0 pmm-ha-1 pmm-ha-2; do
  printf '%s -> ' "$p"
  kubectl exec -n pmm "$p" -c pmm-ha -- \
    curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/server/leaderHealthCheck
done
```

Exactly one pod must answer 200. If none does, see the sections below.

To reach PMM while you investigate, bypass the load balancer:

```sh
kubectl port-forward -n pmm svc/pmm-ha-haproxy 18443:443
```

PMM is then available at `https://127.0.0.1:18443`.

## No leader is elected, or there is more than one

PMM does not serve requests, or the cluster behaves inconsistently.

**Fix:** Check the HA metrics. Query them through the Grafana data source proxy, using the UID of the Metrics data source:

```sh
curl -sk -u admin:$PASSWORD \
  "$PMM_URL/graph/api/datasources/proxy/uid/<datasource-uid>/api/v1/query?query=pmm_ha_leader_status"
```

Interpret the result:

| Query | Healthy value | Meaning if different |
|-------|---------------|----------------------|
| `sum(pmm_ha_leader_status)` | `1` | `0` means no leader; above `1` means split brain |
| `count(pmm_ha_up{role="voter"} == 1)` | Number of replicas | A lower value means the cluster has lost quorum |
| `pmm_ha_raft_term` | Stable | A rapidly increasing value means the leader keeps changing |

The `node_id` label on these metrics is the pod name.

!!! note ""
    Immediately after a failover these values are briefly inconsistent because the metrics are only updated at the next scrape. Wait and query again before concluding that the cluster is broken.

## No quorum: cluster is unreachable after losing multiple replicas

PMM HA uses the Raft consensus protocol, which requires a majority (quorum) of the three PMM Server replicas to be reachable to hold a valid election: at least 2 of 3. If two replicas are lost at the same time (for example, two nodes fail together, or the `StatefulSet` is scaled down too far), the single remaining replica cannot elect itself leader. HAProxy only routes traffic to a replica that passes its leader health check, so with no leader, every request fails with `503`, even though the surviving replica is otherwise healthy.

This is different from a normal rolling restart (during an upgrade, for example), where the two surviving replicas out of three still form a quorum and elect a new leader within seconds. Quorum loss does not self-resolve.

**Confirm this is what's happening.** On the surviving replica, look for a repeating election loop in the logs with no successful election:

```sh
kubectl exec -n <namespace> <surviving-pod> -c pmm-ha -- \
  grep -i "raft" /srv/logs/pmm-managed.log | tail -20
```

A cluster stuck without quorum repeats lines like these indefinitely:

```text
[WARN]  raft: Election timeout reached, restarting election
[ERROR] raft: failed to make requestVote RPC: target="{Voter pmm-ha-1 ...}" error="dial tcp: lookup pmm-ha-1... no such host"
```

**Fix:** Bring at least one more of the original replicas back online so two of the three can reach each other again:

```sh
kubectl get pods -n <namespace> -l app.kubernetes.io/component=pmm-server
kubectl describe pod <pod-name> -n <namespace>   # check why it isn't Running
```

No manual Raft intervention is needed: as soon as a second replica becomes network-reachable, the two members elect a leader immediately. In testing, restoring service after a full quorum loss took about a minute, almost all of it Kubernetes rescheduling and starting the pod, not waiting on the election itself.

!!! danger "If none of the original replicas can come back"
    If a replica's underlying storage or node is permanently unrecoverable, manually reconstituting Raft membership with new replicas is an advanced recovery scenario. Treat it as a case for Percona Support rather than improvising: forcing Raft membership changes incorrectly on a cluster with mismatched logs can lead to split-brain and data loss.

## Check status in the UI before troubleshooting data issues

If dashboards or Query Analytics are empty, check the state of your cluster in the PMM UI before you start reading logs.

**PMM HA Health Overview dashboard:** Go to **Dashboards > PMM Health > PMM HA Health Overview**. The dashboard shows the health of each component (PMM, PostgreSQL, ClickHouse, VictoriaMetrics, and HAProxy), together with pod count, pod restarts, and storage usage for each data store.

If a component is reported as unhealthy here, troubleshoot that component first. An empty dashboard or empty Query Analytics is usually a symptom of a component that is not running.

**Inventory page:** Go to **Configuration > Inventory**. On the **Nodes** tab, check that all `pmm-ha` nodes are present and their status is healthy. On the **Services** tab, check that the services you added are listed and that their agents are running.

If a service is missing from Inventory, the problem is with adding the service, not with storing or displaying its data.

## Dashboards show no data

Dashboards are empty, but PMM itself is working.

First check the PMM HA Health Overview dashboard and the Inventory page as described above. Continue only if all components are healthy and your services are listed.

**Cause:** In HA mode, metrics are stored in an external VictoriaMetrics cluster. The single-node query path that PMM uses in a standard installation is not available. Requesting it returns:

```
missing route for "/prometheus/api/v1/query"
```

**Fix:** Query metrics through the Grafana data source proxy instead, which is what Explore uses:

```
/graph/api/datasources/proxy/uid/<datasource-uid>/api/v1/query
```

If Explore returns data but a dashboard does not, the problem is with the dashboard, not with HA.

Also check that the VictoriaMetrics pods are running:

```sh
kubectl get pods -n pmm | grep -E 'vmstorage|vmselect|vminsert|vmagent'
```

!!! note ""
    Metrics for a newly added service appear after a few scrape intervals. An empty result immediately after adding a service does not indicate a problem.

## Query Analytics shows no data

QAN is empty while dashboards show data.

First check the PMM HA Health Overview dashboard and the Inventory page as described above. Continue only if ClickHouse is healthy and your services are listed.

**Cause:** QAN data is stored in the external ClickHouse cluster, not on the PMM Server pod.

**Fix:** Check that the ClickHouse pods and their Keeper pods are running, and read their logs:

```sh
kubectl get pods -n pmm -l clickhouse.altinity.com/chi=pmm-ha
kubectl logs -n pmm pmm-ha-pmmdb-0-0-0 -c clickhouse --tail=100
```

Then check `qan-api2.log` on the PMM Server pod:

```sh
kubectl exec -n pmm pmm-ha-0 -c pmm-ha -- tail -100 /srv/logs/qan-api2.log
```

!!! note ""
    The **Query Analytics for PMM Server** option under **Configuration > Settings > Advanced Settings** cannot be enabled in HA mode. It returns `QAN for internal PostgreSQL is already configured via an environment variable`. This is expected.

## A backup or restore fails

`pmm-backup.sh` exits with an error, or a scheduled backup Job fails.

**Cause:** One of the components couldn't be backed up or restored, the storage isn't reachable, or an earlier run that didn't finish is still holding a lock.

**Fix:** Check the log of the most recent run:

```sh
kubectl exec -n pmm deploy/pmm-ha-backup-tools -- sh -c 'ls -t /backups/logs | head'
kubectl exec -n pmm deploy/pmm-ha-backup-tools -- tail -50 /backups/logs/<log-file>
```

Then check whether a lock is still held:

```sh
kubectl get leases -n pmm | grep pmm-backup
```

A lock from a run that was stopped expires after 15 minutes. A new run started before then is refused on purpose, so it can't interfere with the stopped one. Wait for the lock to expire, then run the backup or restore again.

## A restore was interrupted and PMM stays down

After a restore was stopped part-way, PMM is unreachable and the PMM Server pods don't come back.

**Cause:** A restore scales PMM Server and VictoriaMetrics down to zero before replacing their data. If it's stopped before it scales them back up, for example because its node was replaced, they stay at zero. PMM Server is scaled back up by the next restore, but VictoriaMetrics is not, so the next restore refuses to start.

**Fix:**
{.power-number}

1. Scale VictoriaMetrics back to the replica counts from your values file. The defaults are 3 `vmstorage` and 2 `vminsert` replicas:

    ```sh
    kubectl patch vmcluster pmm-ha-vmcluster -n pmm --type=merge \
      -p '{"spec":{"vmstorage":{"replicaCount":3},"vminsert":{"replicaCount":2}}}'
    ```

2. Wait for any leftover restore locks to expire:

    ```sh
    kubectl get leases -n pmm | grep pmm-backup
    ```

3. Run the restore again. To avoid another interruption, run it as a Job. See [Restore in place](../install-pmm/backup-restore-HA-clustered.md#restore-in-place).

## A backup or restore Job stays in Pending state

A Job created from the `pmm-ha-backup` CronJob never starts.

**Cause:** With S3 storage, the Job must run on the same node as the `pmm-ha-backup-tools` pod, because they share a `ReadWriteOnce` volume. If that node has no free CPU or memory, the Job can't be scheduled. The scheduler reports a pod affinity error rather than a resource shortage.

**Fix:** Free resources on that node. Before a restore, scaling PMM Server to zero is usually enough, since the restore does this anyway:

```sh
kubectl scale statefulset pmm-ha -n pmm --replicas=0
```

To avoid this permanently, set `centralBackupStorage.accessMode: ReadWriteMany` in your values file, so the Job can run on any node.

## PMM Client pods fail after a restore into another namespace

After you restore a backup into another namespace, the `pmm-ha-client` or PostgreSQL `pmm-client` containers restart repeatedly with authentication errors.

**Cause:** The restore replaced the Grafana database, so PMM now uses the source installation's admin password. The new namespace's `pmm-secret` still holds its own password, so the clients can't authenticate.

**Fix:** Set `PMM_ADMIN_PASSWORD` in the new namespace's `pmm-secret` to the source installation's admin password, then restart the failing pods. See [Restore into another namespace](../install-pmm/backup-restore-HA-clustered.md#restore-into-another-namespace).

## See also

- [Understand PMM High Availability Cluster](../install-pmm/HA-clustered.md)
- [Install PMM HA Cluster](../install-pmm/install-HA-clustered.md)
- [Back up and restore PMM HA Cluster](../install-pmm/backup-restore-HA-clustered.md)
- [Upgrade PMM HA Cluster using Helm](../pmm-upgrade/upgrade_helm_ha.md)
