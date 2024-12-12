# Hardware and system requirements

## Server requirements

* **Disk**

    Approximately 1 GB of storage per monitored database node with data retention set to one week. By default, [retention](../../configure-pmm/advanced_settings.md#data-retention) is 30 days.

    !!! hint alert alert-success "Tip"
        [Disable table statistics](../../optimize/disable_table_stats.md) to decrease the VictoriaMetrics database size.

* **Memory**

    A minimum of 2 GB per monitored database node. The increase in memory usage is not proportional to the number of nodes. For example, data from 20 nodes should be easily handled with 16 GB.

* **Architecture**

    - Your CPU must support the [`SSE4.2`](https://wikipedia.org/wiki/SSE4#SSE4.2) instruction set, a requirement of ClickHouse, a third-party column-oriented database used by Query Analytics. If your CPU is lacking this instruction set you won't be able to use Query Analytics.
    - For ARM64 systems, ensure you're using a supported ARM64 architecture (e.g., ARMv8). Note that some features may have different performance characteristics on ARM compared to x86_64 systems.

## Client requirements

* **Disk**

    A minimum of 100 MB of storage is required for installing the PMM Client package. With a good connection to PMM Server, additional storage is not required. However, the client needs to store any collected data that it cannot dispatch immediately, so additional storage may be required if the connection is unstable or the throughput is low. VMagent uses 1 GB of disk space for cache during a network outage. QAN, on the other hand, uses RAM to store cache.

* **Operating system**

    PMM Client runs on any modern 64-bit Linux distribution, including ARM-based systems. It is tested on supported versions of Debian, Ubuntu, CentOS, and Red Hat Enterprise Linux, on both x86_64 and ARM64 architectures. See [Percona software support life cycle](https://www.percona.com/services/policies/percona-software-support-lifecycle#pt).


### ARM-specific considerations

- **Docker**: If using Docker for PMM Client on ARM systems, ensure you're using the ARM64-compatible Docker images.
- **Performance**: Performance may vary across different ARM implementations. Conduct thorough testing to ensure optimal performance in your environment.
- **Compatibility**: Ensure you're using ARM-compatible versions of any additional software or databases you're monitoring with PMM.
- **PMM Server**: PMM Server is not currently available as a native ARM64 build. For ARM-based systems, consider using the Docker or Podman installation methods, which can run x86_64 images via emulation on ARM platforms.
- **Resource usage**: Monitor resource usage closely on ARM systems, as it may differ from x86_64 systems. Adjust your configuration as needed for optimal performance.
