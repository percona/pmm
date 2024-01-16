# Uninstall PMM using Helm

To uninstall `pmm` deployment:

```sh
helm uninstall pmm
```

This command takes a release name and uninstalls the release.

It removes all resources associated with the last release of the chart as well as the release history.

Helm will not delete PVC, PV, and any snapshots. Those need to be deleted manually.

Also, delete PMM `Secret` if no longer required:

```sh
kubectl delete secret pmm-secret
```