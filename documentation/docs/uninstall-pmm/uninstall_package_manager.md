# Uninstall PMM client using package manager

To uninstall PMM client with package manager, do the following steps:

=== "Debian-based distributions"

    To uninstall PMM client with Debian-based distributions:
    {.power-number}

    1. Uninstall the PMM Client package.

        ```sh
        apt remove -y pmm-client
        ```

    2. Remove the Percona repository

        ```sh
        dpkg -r percona-release
        ```

=== "Red Hat-based distributions"

    To uninstall PMM client with Red Hat based distributions:
    {.power-number}

    1. Uninstall the PMM Client package.

        ```sh
        yum remove -y pmm-client
        ```

    2. Remove the Percona repository

        ```sh
        yum remove -y percona-release
        ```