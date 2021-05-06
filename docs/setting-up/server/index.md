# Setting up PMM Server

## System requirements

**Disk**

Approximately 1 GB of storage per monitored database node with data retention set to one week. By default, [retention](../../how-to/configure.md#data-retention) is 30 days.

> [Disable table statistics](../../how-to/optimize.md) to decrease the VictoriaMetrics database size.

**Memory**

A minimum of 2 GB per monitored database node. The increase in memory usage is not proportional to the number of nodes. For example, data from 20 nodes should be easily handled with 16 GB.

**Architecture**

Your CPU must support the SSE4.2 instruction set, a requirement of ClickHouse, a third-party column-oriented database used by Query Analytics. If your CPU is lacking this instruction set you won't be able to use Query Analytics.

---

Choose how you want to run PMM Server:

- [with Docker](docker.md)
- [as a virtual appliance](virtual-appliance.md)
- [on an Amazon AWS EC2 instance](aws.md)

When PMM Server is running, set up [PMM Client](../client/index.md) for each node or service.

## Single line installer

> <b style="color:goldenrod">Caution</b> This is a [technical preview](../../details/glossary.md#technical-preview) and is subject to change.

```sh
curl -fsSL https://raw.githubusercontent.com/percona/pmm/PMM-2.0/get-pmm.sh -o get-pmm2.sh ; chmod +x get-pmm2.sh ; ./get-pmm2.sh
```

> <b style="color:red">Warning</b> We highly recomend you review `get-pmm2.sh` prior to running on your system, to ensure the content is as expected.

This command will:

- if Docker is not already installed, install it
- if there is a PMM Server docker container running, stop it and back it up
- pull and run the latest PMM Server docker image
