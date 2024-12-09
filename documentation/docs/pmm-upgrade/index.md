# About PMM Server upgrade

!!! caution alert alert-warning "Upgrade PMM Server before Clients"
    - When upgrading PMM, always upgrade the PMM Server before upgrading any PMM Clients.
    - Make sure that the PMM Server version is higher than or equal to the PMM Client version. Mismatched versions can lead to configuration issues and failures in Client-Server communication, as the PMM Server may not recognize all parameters in the client configuration.

Find the detailed information on how to upgrade PMM in the following sections:

* [Upgrade PMM Server from the UI](ui_upgrade.md)

* [Upgrade PMM Client](upgrade_agent.md)

* [Upgrade PMM Server using Docker](upgrade_docker.md)

* [Migrate from PMM 2](migrating_from_pmm_2.md)
