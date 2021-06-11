# Set up PMM Server

To set up PMM Server:

1. Configure your [network](network.md).

2. Decide how you want to run PMM Server. Choose from:

    - [Docker](docker.md)
    - [Virtual appliance](virtual-appliance.md)
    - [Amazon AWS EC2 instance](aws.md)
    - [One-line installer](#one-line-installer)

## System requirements

**Disk**

Approximately 1 GB of storage per monitored database node with data retention set to one week. By default, [retention](../../how-to/configure.md#data-retention) is 30 days.

!!! tip alert alert-success "Tip"
    [Disable table statistics](../../how-to/optimize.md) to decrease the VictoriaMetrics database size.

**Memory**

A minimum of 2 GB per monitored database node. The increase in memory usage is not proportional to the number of nodes. For example, data from 20 nodes should be easily handled with 16 GB.

**Architecture**

Your CPU must support the SSE4.2 instruction set, a requirement of ClickHouse, a third-party column-oriented database used by Query Analytics. If your CPU is lacking this instruction set you won't be able to use Query Analytics.



## One-line installer

!!! caution alert alert-warning "Caution"
    This is a [technical preview] and is subject to change.

```sh
curl -fsSL https://raw.githubusercontent.com/percona/pmm/PMM-2.0/get-pmm.sh -o get-pmm2.sh ; chmod +x get-pmm2.sh ; ./get-pmm2.sh
```

!!! caution alert alert-warning "Caution"
    Download and check `get-pmm2.sh` before running it to make sure you know what it does.

This command will:

- install Docker if not already installed;
- if there is a PMM Server docker container running, stop it and back it up;
- pull and run the latest PMM Server docker image.

[technical preview]: ../../details/glossary.md#technical-preview