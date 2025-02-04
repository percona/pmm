# Hardware and system requirements

## PMM Server requirements

PMM Server's resource requirements depend on your monitoring environment. Here are our recommendations for different deployment scales:

### Typical deployment (up to 30 nodes)
This is the most common deployment scenario, suitable for small to medium-sized environments:

- **CPU**: 4 cores
- **Memory**: 8 GB  
- **Storage**: 100 GB

### Medium deployment (up to 200 nodes) 
Recommended for environments monitoring MySQL, PostgreSQL, or MongoDB at scale:

- **CPU**: 8-16 cores
- **Memory**: 16-32 GB
- **Storage**: 200 GB
- **CPU usage**: Expect 20-70% utilization

### Large deployment (500+ nodes)
Designed for extensive monitoring environments with high node counts:

- **CPU**: 16+ cores
- **Memory**: 32+ GB
- **Storage**: 500+ GB

## Storage calculation
Adjust storage calculations based on your data retention period and the number of metrics collected. To estimate storage requirements:

- allow approximately 1 GB of storage per monitored node per week.
- for the default 30-day retention period, use the formula: `number_of_nodes * 4 GB`.

### Server architecture requirements

- **CPU**: Must support the [`SSE4.2`](https://wikipedia.org/wiki/SSE4#SSE4.2), which is required for Query Analytics (QAN).
- **ARM64**: Ensure your system uses a supported ARM64 architecture (e.g., ARMv8).
- **ARM limitations**: PMM Server is not currently available as a native ARM64 build. For ARM-based systems, use Docker or Podman to run x86_64 images via emulation.

!!! hint alert alert-success "Tip"
    To reduce storage usage, consider [disabling table statistics](../../optimize/disable_table_stats.md), which can significantly decrease the size of the VictoriaMetrics database.

## Client requirements

### Storage

The PMM Client package requires 100 MB of storage for installation. Under normal operation with a stable connection to PMM Server, no additional storage is needed. During network instability or low throughput periods, the Client temporarily stores collected data that cannot be immediately dispatched. The VM Agent reserves 1 GB of disk space for caching during network outages, while Query Analytics (QAN) utilizes RAM instead of disk storage for its cache.

### Operating system

PMM Client is compatible with modern 64-bit Linux distributions on both x86_64 and ARM64 architectures. Supported platforms include current versions of Debian, Ubuntu, CentOS, and Red Hat Enterprise Linux. For specific version support details, see [Percona software support life cycle](https://www.percona.com/services/policies/percona-software-support-lifecycle#pt).

### ARM-specific considerations

- **Docker**: If using Docker for PMM Client on ARM systems, ensure you're using the ARM64-compatible Docker images.
- **Performance**: Performance may vary across different ARM implementations. Conduct thorough testing to ensure optimal performance in your environment.
- **Compatibility**: Ensure you're using ARM-compatible versions of any additional software or databases you're monitoring with PMM.
- **Resource usage**: Monitor resource usage closely on ARM systems, as it may differ from x86_64 systems. Adjust your configuration as needed for optimal performance.