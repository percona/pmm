# About PMM Server upgrade

!!! caution alert alert-warning "Upgrade PMM Server before Clients"
    - When upgrading PMM, always upgrade the PMM Server before upgrading any PMM Clients.
    - Make sure that the PMM Server version is higher than or equal to the PMM Client version. Mismatched versions can lead to configuration issues and failures in Client-Server communication, as the PMM Server may not recognize all parameters in the client configuration.
## Available upgrade methods

Choose your preferred upgrade method based on your setup:

* [Upgrade PMM Client](upgrade_client.md)
* [Upgrade PMM Server using Docker](upgrade_docker.md)
* [Upgrade PMM Server using Helm](upgrade_helm.md)
* [Migrate from PMM 2](migrating_from_pmm_2.md) (direct migration deprecated)

## Internal PostgreSQL database upgrade

Starting with PMM 3.10.0, the internal PostgreSQL database of PMM Server uses PostgreSQL 18. If you upgrade from an earlier version, PMM moves your data from PostgreSQL 14 to PostgreSQL 18 automatically the first time PMM Server starts. 

This doesn't apply to a [PMM HA Cluster](../install-pmm/HA-clustered.md) or to a PMM Server that uses an [external PostgreSQL database](../reference/third-party/postgresql.md).

To upgrade from a version earlier than PMM 3.10.0:
{.power-number}

1. Make sure the PMM Server data volume has enough free space for a second copy of the database.
2. Upgrade PMM Server with your usual [upgrade method](#available-upgrade-methods). The first start takes longer than usual while PMM copies your data.
3. When you're sure that PMM works as expected and you don't need to roll back, delete `/srv/postgres14.old` to free up space. PMM keeps your old data there after the upgrade.

If the data copy fails, PMM Server doesn't start. Your existing data stays unchanged, and PMM tries again each time the container restarts. 

To find out what went wrong, check the container logs, for example with `docker logs pmm-server`.
