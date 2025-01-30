# Prerequisites for PMM Client

Before installing PMM Client, ensure you meet the following requirements:
{.power-number}

1. [Install PMM Server](../install-pmm-server/index.md) and note the server's IP address - it must be accessible from the Client node.
2. Check that you have superuser (`root`) access on the client host.
3. Check that you have superuser access to all database servers you plan to monitor.
4. Verify you have these Linux packages installed:
    * `curl`

    * `gnupg`

    * `sudo`

    * `wget`
5. If you use it, [install Docker](https://docs.docker.com/get-started/get-docker/).
6. Check [hardware and system requirements for PMM Client](../plan-pmm-installation/hardware_and_system.md)