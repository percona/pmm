# Upgrade PMM server uisng Helm

Percona will release a new chart updating its containers if a new version of the main container is available, there are any significant changes, or critical vulnerabilities exist.

By default UI update feature is disabled and should not be enabled. Do not modify that parameter or add it while modifying the custom `values.yaml` file:

```yaml
pmmEnv:
  DISABLE_UPDATES: "1"
```

Before updating the helm chart,  it is recommended to pre-pull the image on the node where PMM is running, as the PMM images could be large and could take time to download.

Update PMM as follows:

```sh
helm repo update percona
helm upgrade pmm -f values.yaml percona/pmm
```

This will check updates in the repo and upgrade deployment if the updates are available.