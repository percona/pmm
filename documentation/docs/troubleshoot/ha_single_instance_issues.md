# Troubleshoot Single-Instance HA issues

Use this page when your PMM pod is not running, not accessible, or not persisting data. For problems that aren't specific to this deployment, such as dashboards, agents, or queries on an otherwise healthy server, see [Troubleshoot PMM](index.md).

## Pod stuck in Pending state

**Problem**: PMM pod remains in `Pending` state.

**Solution**: Check for resource or storage issues:

```sh
# Check why pod is pending
kubectl describe pod -n monitoring -l app=pmm

# Check PVC status
kubectl get pvc -n monitoring

# Check available node resources
kubectl top nodes
```

Common causes:

- Insufficient CPU or memory on nodes
- PVC cannot be bound (no available PV or storage class issue)
- Node selector or affinity rules prevent scheduling

## Pod constantly restarting

**Problem**: Pod enters `CrashLoopBackOff` state.

**Solution**: Check logs for errors:

```sh
# View recent logs
kubectl logs -n monitoring -l app=pmm --tail=100

# View previous container logs
kubectl logs -n monitoring -l app=pmm --previous
```

Common causes:

- Corrupted data volume (check PV)
- Insufficient memory (increase limits)
- Configuration errors (check environment variables)

## Cannot access PMM UI

**Problem**: LoadBalancer external IP is pending or connection is refused.

**Solution**: Verify service configuration:

```sh
# Check service status
kubectl get svc -n monitoring monitoring-service

# Check service endpoints
kubectl get endpoints -n monitoring monitoring-service

# Test connectivity from within the cluster
kubectl run -it --rm debug \
  --image=curlimages/curl \
  --restart=Never \
  -- curl -k https://monitoring-service.monitoring.svc.cluster.local
```

If the LoadBalancer IP stays pending, verify that your cloud provider supports LoadBalancer services, check quota limits, or switch to NodePort or Ingress.

## High memory usage

**Problem**: PMM is consuming excessive memory.

**Solution**: Reduce data retention or increase memory limits:

```sh
# Check current memory usage
kubectl top pod -n monitoring -l app=pmm

# Reduce data retention
kubectl exec -n monitoring -l app=pmm -- \
  pmm-admin config --data-retention=7d
```

To increase memory limits, update `values.yaml`:

```yaml
resources:
  limits:
    memory: "16Gi"
```

Then apply the change:

```sh
helm upgrade pmm percona/pmm \
  --namespace monitoring \
  --values values.yaml
```

## Data not persisting after pod restart

**Problem**: Data is lost after a pod restart.

**Solution**: Verify persistent volume configuration:

```sh
# Check PVC is bound
kubectl get pvc -n monitoring

# Check PV exists
kubectl get pv

# Verify pod is using the PVC
kubectl describe pod -n monitoring -l app=pmm | grep -A 5 Volumes
```

Also confirm that persistence is enabled in `values.yaml`:

```yaml
persistence:
  enabled: true
```

## Slow pod rescheduling

**Problem**: Pod takes longer than 5 minutes to reschedule after a node failure.

**Solution**: Check node health and volume attachment:

```sh
# Check node status
kubectl get nodes

# Check pod events
kubectl describe pod -n monitoring -l app=pmm
```

For cloud providers, volume detachment from a failed node can add time before the pod starts on a new node. Using a regional persistent disk (rather than a zonal one) reduces this delay.

## See also

- [Understand Single-Instance HA](../install-pmm/HA-kubernetes-single-instance.md)
- [Install PMM Single-Instance HA](../install-pmm/install-HA-kubernetes-single-instance.md)
