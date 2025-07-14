# Uninstall PMM using Helm

Remove PMM Server deployed via Helm in a Kubernetes cluster.

!!! warning "Data loss warning"
    This permanently removes PMM Server and all monitoring data. Ensure you have backed up any important data before uninstalling.

## Prerequisites

- [Unregister PMM Client](unregister_client.md) from PMM Server
- Helm and kubectl access to the cluster
- Permissions to manage resources in the namespace where PMM is deployed

## Uninstall steps

1. Uninstall the `pmm` Helm release and remove all resources associated with the PMM release and the release history:

    ```sh
    helm uninstall pmm
    ```

2. Manually remove remaining resources as Helm does not delete PVC, PV, and any snapshots: 

```
# Delete persistent volume claims
kubectl get pvc | grep pmm
kubectl delete pvc <pvc-name>

# Delete secrets (if no longer needed)
kubectl delete secret pmm-secret

# Delete any remaining config maps
kubectl get configmap | grep pmm
kubectl delete configmap <configmap-name>
```