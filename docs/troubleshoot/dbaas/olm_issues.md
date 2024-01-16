
# Troubleshooting OLM installation

By default OLM installs its operators and catalogs into `olm` namespace. Here's the example of successful installation of OLM

```
kubectl get pods -n olm
NAME                                                              READY   STATUS      RESTARTS   AGE
catalog-operator-67bcbb4f5d-jmdhv                                 1/1     Running     0          16h
olm-operator-d9f76fdc9-7vdx2                                      1/1     Running     0          16h
operatorhubio-catalog-4gzfk                                       1/1     Running     0          74s
packageserver-877948445-swdq5                                     1/1     Running     0          16h
packageserver-877948445-sz44r                                     1/1     Running     0          16h
percona-dbaas-catalog-777hb                                       1/1     Running     0          16h
```

Note: the output may be longer but it shows what needs to be installed automatically

1. OLM Operator is responsible for deploying applications defined by CSV resources after the required resources specified in the CSV are present in the cluster.
2. The Catalog Operator is responsible for resolving and installing CSVs and the required resources they specify. It is also responsible for watching CatalogSources for updates to packages in channels and upgrading them (optionally automatically) to the latest available versions.
3. Packageserver is responsible for providing metadata to the operators like ClusterServiceVersion that is used for installing/upgrading operators.
4. percona-dbaas-catalog is Percona's managed catalog that defines which operators' versions are available. This component has information about tested and supported versions of operators.

You can use `kubectl describe pod -n olm podName` to understand what went wrong during the installation.



