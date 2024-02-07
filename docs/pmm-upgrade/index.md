# About PMM upgrade

!!! caution alert alert-warning "Important"
    Upgrade the PMM Server before you upgrade the PMM Client.
    Ensure that the PMM Server version is higher than or equal to the PMM Client version. Otherwise, there might be configuration issues, thus leading to failure in the client-server communication as PMM Server might not be able to identify all the parameters in the configuration.

    For example, for a PMM Server version 2.25.0, the PMM Client version should be 2.25.0 or 2.24.0. If the PMM Client version is 2.26.0, PMM might not work as expected.

Find the detailed information on how to upgrade PMM in the following sections:

* [Upgrade PMM server using the UI](ui_upgrade.md)

* [Upgrade PMM agent](upgrade_agent.md)

* [Upgrade PMM server using Docker](upgrade_docker.md)

* [Upgrade from PMM 1](upgrade_from_pmm_1.md)