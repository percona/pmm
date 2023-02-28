# pmm-server-upgrade tests

## Requirements
- `percona/pmm-server-upgrade:latest` image  
  ```
  make -C admin build-docker PMM_RELEASE_VERSION=latest
  ```
- `pmm` cli tool
  ```
  make -C admin install
  ```

## Self-update
### Requirements
- `percona/pmm-server-upgrade:first` image  
  ```
    make -C admin build-docker PMM_RELEASE_VERSION=first
  ```


## Upgrade
### Configuration

`UPGRADER_UPGRADE_OLD_IMAGE` - image used for running the current version of PMM Server

### Running locally

Build your own image of PMM Server:
```
make env-up
make env
make run-managed
exit
docker stop pmm-managed-server
docker commit pmm-managed-server ps
```

Build `pmm-server-upgrade` Docker image:
```
make -C admin build-docker
```

Run tests
```
UPGRADER_UPGRADE_OLD_IMAGE=ps npx playwright test --reporter=list tests/pmm-server-upgrade/upgrade/
```
