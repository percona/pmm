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

## Internal PostgreSQL database

Starting with PMM 3.10.0, PMM Server's built-in PostgreSQL database (used by pmm-managed and Grafana) is PostgreSQL 18. If your PMM Server data volume still has PostgreSQL 14 data, PMM migrates it automatically the first time the container starts on the new version — no manual steps are required. The PostgreSQL 14 data stays untouched until the migration finishes, and PMM retries automatically on every restart until it succeeds.

This doesn't apply if you use PMM HA or an external PostgreSQL database — PostgreSQL is managed separately in both cases.