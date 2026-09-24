# Troubleshoot PMM HA Cluster issues

## No quorum: cluster is unreachable after losing multiple replicas

**Symptom:** The PMM HA cluster stops responding entirely. Requests to the HAProxy endpoint return `503 Service Unavailable`, and this does not recover on its own.

**Cause:** PMM HA elects a leader using the Raft consensus protocol, which requires a majority (**quorum**) of the three PMM server replicas to be reachable to hold a valid election—at least 2 of 3. If two replicas are lost at the same time (for example, two nodes fail together, or the `StatefulSet` is scaled down too far), the single remaining replica cannot elect itself leader. HAProxy only routes traffic to a replica that passes its leader health check, so with no leader, every request fails with `503`—even though the surviving replica is otherwise healthy and running.

This is different from a normal rolling restart (during an upgrade, for example), where the two surviving replicas out of three still form a quorum and elect a new leader within seconds. Quorum loss means the outage does **not** self-resolve, however long you wait.

**Confirm this is what's happening.** On the surviving replica, look for a repeating election loop in the logs, with no successful election:

```sh
kubectl exec -n <namespace> <surviving-pod> -c pmm-ha -- \
  grep -i "raft" /srv/logs/pmm-managed.log | tail -20
```

A cluster stuck without quorum repeats lines like these indefinitely, never reaching a resolution:

```text
[WARN]  raft: Election timeout reached, restarting election
[ERROR] raft: failed to make requestVote RPC: target="{Voter pmm-ha-1 ...}" error="dial tcp: lookup pmm-ha-1... no such host"
```

**Fix it:** bring at least one more of the original replicas back online so two of the three can reach each other again—for example, fix whatever took the node down, uncordon it, or resolve why the pod can't be scheduled:

```sh
kubectl get pods -n <namespace> -l app.kubernetes.io/component=pmm-server
kubectl describe pod <pod-name> -n <namespace>   # check why it isn't Running
```

No manual Raft intervention (like forcing a new cluster) is needed in this case: as soon as a second replica becomes network-reachable, the two remaining members elect a leader immediately—confirmed in testing by the log line `msg="I am the leader!" component=ha` appearing within seconds of the second pod coming back, before it even finished its Kubernetes readiness checks. In a live test, restoring service after a full quorum loss took about a minute, almost all of it Kubernetes rescheduling and starting the pod—not waiting on the election itself.

!!! danger "If none of the original replicas can come back"
    If a replica's underlying storage or node is permanently unrecoverable, getting the *same* replica back isn't an option. Manually reconstituting Raft membership with new replicas is an advanced recovery scenario this topic doesn't cover—treat it as a case for Percona Support rather than improvising, since forcing Raft membership changes incorrectly on a cluster with mismatched logs can lead to a split-brain and data loss.

## See also

- [Understand PMM High Availability Cluster](../install-pmm/HA-clustered.md)
- [Upgrade PMM HA Cluster using Helm](../pmm-upgrade/upgrade_helm_ha.md)
