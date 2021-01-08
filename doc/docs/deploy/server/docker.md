# Running PMM Server via Docker

Docker images of PMM Server are stored at the [percona/pmm-server](https://hub.docker.com/r/percona/pmm-server/tags/) public repository. The host must be able to run Docker 1.12.6 or later, and have network access.

**NOTE**: You can also download the PMM Server image from the [PMM download page](https://www.percona.com/downloads/pmm/). Choose the appropriate PMM version and the *Server - Docker Image* item in two pop-up menus to get the download link.

PMM needs roughly 1GB of storage for each monitored database node with data retention set to one week. Minimum memory is 2 GB for one monitored database node, but it is not linear when you add more nodes.  For example, data from 20 nodes should be easily handled with 16 GB.

Make sure that the firewall and routing rules of the host do not constrain the Docker container. For more information, see [How to troubleshoot communication issues between PMM Client and PMM Server?](../../faq.md#troubleshoot-connection).

For more information about using Docker, see the [Docker Docs](https://docs.docker.com).

!!! important
    By default, [retention](../../glossary.terminology.md#data-retention) is set to 30 days for
   Metrics Monitor and to 8 days for PMM Query Analytics.  Also consider [disabling table statistics](../../faq.md#performance-issues), which can greatly decrease Prometheus database size.


* [Setting Up a Docker Container for PMM Server](docker.setting-up.md)
* [Updating PMM Server Using Docker](docker.upgrading.md)
* [Backing Up PMM Data from the Docker Container](docker.backing-up.md)
* [Restoring the Backed Up Information to the PMM Data Container](docker.restoring.md)
