# About PMM Server upgrade

!!! caution alert alert-warning "Upgrade PMM Server before Clients"
    - When upgrading PMM, always upgrade the PMM Server before upgrading any PMM Clients.
    - Make sure that the PMM Server version is higher than or equal to the PMM Client version. Mismatched versions can lead to configuration issues and failures in Client-Server communication, as the PMM Server may not recognize all parameters in the client configuration.

## Available upgrade methods

Choose your preferred upgrade method based on your setup:

* [Upgrade PMM Client](upgrade_client.md)
* [Upgrade PMM Server using Docker](upgrade_docker.md)
* [Upgrade PMM Server using Helm](upgrade_helm.md)
* [Upgrade PMM HA Cluster using Helm](upgrade_helm_ha.md) (upgrading HA clusters not supported before PMM 3.10.0)
* [Migrate from PMM 2](migrating_from_pmm_2.md) (direct migration deprecated)

## Upgrade from PMM 3.9.1 or earlier

If your PMM Server runs PMM 3.9.1 or earlier, upgrading it also moves your data from PostgreSQL 14 to PostgreSQL 18, which takes a bit more time.

This is because PMM 3.10.0 and later use PostgreSQL 18. Following the end of PostgreSQL support for version 14, PMM 3.11.0 deprecates it and will remove it completely in PMM 3.13.0 (expected January 2027).

Once PMM 3.13.0 removes PostgreSQL 14, PMM can no longer migrate that data, so it can't upgrade a PMM Server that runs PMM 3.9.1 or earlier directly. If your PMM Server runs PMM 3.9.1 or earlier:

- before PMM 3.13.0: upgrade to PMM 3.11.0 or 3.12.x as usual. Your data moves to PostgreSQL 18 automatically.
- after PMM 3.13.0: [upgrade in two steps](prepare_postgresql_14_removal.md#if-you-upgrade-after-pmm-3130-is-out). First upgrade to a version from PMM 3.10.0 through 3.12.x, then to PMM 3.13.0 or later.
