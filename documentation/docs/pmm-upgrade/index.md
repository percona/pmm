# About PMM Server upgrade

!!! caution alert alert-warning "Upgrade PMM Server before Clients"
    - When upgrading PMM, always upgrade the PMM Server before upgrading any PMM Clients.
    - Make sure that the PMM Server version is higher than or equal to the PMM Client version. Mismatched versions can lead to configuration issues and failures in Client-Server communication, as the PMM Server may not recognize all parameters in the client configuration.
    - For the UI upgrade option, Watchtower must be installed with PMM Server

## Available upgrade methods

Choose your preferred upgrade method based on your setup:

* [Upgrade PMM Server from the UI](ui_upgrade.md)

* [Upgrade PMM Client](upgrade_client.md)

* [Upgrade PMM Server using Docker](upgrade_docker.md)

* [Migrate from PMM 2](migrating_from_pmm_2.md)
