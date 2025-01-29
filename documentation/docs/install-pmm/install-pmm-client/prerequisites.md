# Prerequisites for PMM Client

Before installing PMM Client, ensure you meet the following requirements:
{.power-number}

1. [Install PMM Server](../install-pmm-server/index.md) and note the server's IP address - it must be accessible from the client node.
2. Check that you have superuser (root) access on the client host.
3. Check that you have superuser access to all database servers you plan to monitor.
4. Verify you have these Linux packages installed:
    * `curl`

    * `gnupg`

    * `sudo`

    * `wget`
5. Check that your system meets the hardware requirements:

    - minimum 100 MB for PMM Client package for storage
    - additional storage for unstable network conditions
    - 1 GB disk space for VictoriaMetrics Agent caching during outages
6. Check that your operating system is compatible:

Any modern 64-bit Linux distribution
Supported on both x86_64 and ARM64 architectures
Tested on:

Debian
Ubuntu
CentOS
Red Hat Enterprise Linux




Verify network connectivity:

Stable connection to PMM Server
Firewall rules allow necessary connections
Buffer storage available for unstable connections


If using Docker:

Install Docker
For ARM systems:

Use ARM64-compatible Docker images
Ensure monitored software is ARM-compatible
Monitor resource usage (may differ from x86_64)
Adjust configurations based on ARM performance
5. If you use it, install [Docker](https://docs.docker.com/get-docker/).
6. Check that your system meets the hardware requirements:



Storage: Minimum 100 MB for PMM Client package
Additional storage for unstable network conditions
1 GB disk space for VictoriaMetrics Agent caching during outages


Confirm your operating system is compatible:

Any modern 64-bit Linux distribution
Supported on both x86_64 and ARM64 architectures
Tested on:

Debian
Ubuntu
CentOS
Red Hat Enterprise Linux




Verify network connectivity:

Stable connection to PMM Server
Firewall rules allow necessary connections
Buffer storage available for unstable connections


If using Docker:

Install Docker
For ARM systems:

Use ARM64-compatible Docker images
Ensure monitored software is ARM-compatible
Monitor resource usage (may differ from x86_64)
Adjust configurations based on ARM performance