# About uninstalling PMM 

To completely remove PMM from your system:
{.power-number}

1. [Unregister PMM Client from PMM Server](unregister_client.md) to disconnect from PMM Server and clean up monitoring
2. Uninstall PMM Client to remove the software using your installation method:

    - [Uninstall PMM client with Docker container](uninstall_docker.md)
    - [Uninstall PMM using Helm](uninstall_helm.md)
    - [Uninstall PMM client with package manager](uninstall_package_manager.md)

!!! warning
    Always unregister before uninstalling to avoid orphaned monitoring data on your PMM Server.
