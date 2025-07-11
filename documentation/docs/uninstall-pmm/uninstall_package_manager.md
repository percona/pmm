# Uninstall PMM client using package manager

To uninstall PMM client with package manager, do the following steps:

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