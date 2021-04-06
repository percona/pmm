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
