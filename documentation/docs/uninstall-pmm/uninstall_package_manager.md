# Uninstall PMM Client using package manager

This removes PMM Client installed via system package managers (APT, YUM, etc.).

## Prerequisites

- [Unregister PMM Client](unregister_client.md) from PMM Server
- Root or sudo access to the system

## Uninstall steps

To uninstall PMM client with package manager:

=== "Debian-based distributions"

    To uninstall PMM client with Debian-based distributions:
    {.power-number}

    1. Uninstall the PMM Client package.

        ```sh
        sudo apt remove -y pmm-client
        ```

    2. Remove the Percona repository

        ```sh
        sudo dpkg -r percona-release
        ```

=== "Red Hat-based distributions"

    To uninstall PMM client with Red Hat based distributions:
    {.power-number}

    1. Uninstall the PMM Client package.

        ```sh
        sudo yum remove -y pmm-client
        ```

    2. Remove the Percona repository

        ```sh
        sudo yum remove -y percona-release
        ```