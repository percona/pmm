# Set up PMM Server

1. Check system requirements.

    **Disk**

    Approximately 1 GB of storage per monitored database node with data retention set to one week. By default, [retention](../../how-to/configure.md#data-retention) is 30 days.

    !!! hint alert alert-success "Tip"
        [Disable table statistics](../../how-to/optimize.md) to decrease the VictoriaMetrics database size.

    **Memory**

    A minimum of 2 GB per monitored database node. The increase in memory usage is not proportional to the number of nodes. For example, data from 20 nodes should be easily handled with 16 GB.

    **Architecture**

    Your CPU must support the [`SSE4.2`](https://wikipedia.org/wiki/SSE4#SSE4.2) instruction set, a requirement of ClickHouse, a third-party column-oriented database used by Query Analytics. If your CPU is lacking this instruction set you won't be able to use Query Analytics.

1. Configure your [network](network.md).

1. Decide how you want to run PMM Server. Choose from:

    - [Docker];
    - [Virtual appliance];
    - [Amazon AWS];
    - Use the [easy install] script.

[Docker]: docker.md
[virtual appliance]: virtual-appliance.md
[Amazon AWS]: aws.md
[easy install]: easy-install.md
