# Hardware and system requirements
To ensure optimal performance for your monitoring environment, check the appropriate hardware specifications before installing PMM.

## PMM Server requirements
Resource requirements scale with the number of nodes and services monitored. Here are our recommendations for different deployment scales:

=== "Typical deployment (1-30 nodes)"

    This is the most common deployment scenario, suitable for small to medium-sized environments:

    - **CPU**: 4 cores
    - **Memory**: 8 GB  
    - **Storage**: 100 GB

=== "Medium deployment (31-200 nodes)"

    Recommended for environments monitoring MySQL, PostgreSQL, or MongoDB at scale:

    - **CPU**: 8-16 cores
    - **Memory**: 16-32 GB
    - **Storage**: 200 GB
    - **CPU usage**: Expect 20-70% utilization

=== "Large deployment (200+ nodes)"

    Designed for extensive monitoring environments with high node counts:

    - **CPU**: 16+ cores
    - **Memory**: 32+ GB
    - **Storage**: 500+ GB

## Storage planning
Adjust storage calculations based on your data retention period and the number of metrics collected. To estimate storage requirements:

- **Base formula**: `nodes × retention_period_in_weeks × 1 GB`
- **Quick estimate**: for the default 30-day retention period, use the formula `number_of_nodes x 4 GB`
- **High-precision monitoring**: increase estimates by 20-50% when using 1-second collection intervals

### Server architecture requirements

- **CPU**: must support the [`SSE4.2`](https://wikipedia.org/wiki/SSE4#SSE4.2), which is required for Query Analytics (QAN).
- **ARM64**: ensure your system uses a supported ARM64 architecture (e.g., ARMv8).
- **ARM limitations**: PMM Server is not currently available as a native ARM64 build. For ARM-based systems, use Docker or Podman to run x86_64 images via emulation.

### Storage optimization

To reduce storage usage, consider [disabling table statistics](../../install-pmm/install-pmm-client/connect-database/mysql/disable_table_stats.md), which can significantly decrease the size of the VictoriaMetrics database.

## PMM Client requirements

### Storage

The PMM Client package requires 100 MB of storage for installation. Under normal operation with a stable connection to the PMM Server, no additional storage is needed. However, during network instability or low throughput periods, the Client temporarily caches collected data until it can be transmitted. 

The VM Agent reserves 1 GB of disk space for caching during network outages, while Query Analytics (QAN) uses RAM instead of disk storage for its cache.

### Operating system compatibility

PMM Client is compatible with modern 64-bit Linux distributions on both x86_64 and ARM64 architectures. Supported platforms include current versions of Debian, Ubuntu, CentOS, and Red Hat Enterprise Linux. 

For specific version support details, see [Percona software support life cycle](https://www.percona.com/services/policies/percona-software-support-lifecycle#pt).

### ARM-specific considerations

- **Docker**: If using Docker for PMM Client on ARM systems, ensure you're using the ARM64-compatible Docker images.
- **Performance**: Performance may vary across different ARM implementations. Conduct thorough testing to ensure optimal performance in your environment.
- **Compatibility**: Ensure you're using ARM-compatible versions of any additional software or databases you're monitoring with PMM.
- **Resource usage**: Monitor resource usage closely on ARM systems, as it may differ from x86_64 systems. Adjust your configuration as needed for optimal performance.