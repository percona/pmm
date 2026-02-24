# Exporter issues

## Exporter consuming excessive CPU or memory

If a PMM exporter (such as `mysqld_exporter`, `postgres_exporter`, or `mongodb_exporter`) is using excessive CPU or memory, you can collect profiling data to diagnose the issue.

PMM exporters expose `/debug/pprof/` endpoints for performance profiling. Use these to generate diagnostic data for analysis or to share with Percona Support.

### Collect profiling data

The `pprof` endpoints use basic authentication:

- **username:** `pmm`
- **password:** The exporter's `agent_id` (default), or your custom password if configured

To collect a profile, you need the exporter's password and listening port:
{.power-number}

1. Find the exporter's listening port and password (`agent_id` or your custom password, if set up):
```bash
    pmm-admin list
```

2. From the PMM Client host, collect the profile:
```bash
    # Heap (memory) profile
    curl -su pmm:<password> http://127.0.0.1:<port>/debug/pprof/heap > heap.pprof

    # CPU profile (60 seconds)
    curl -su pmm:<password> "http://127.0.0.1:<port>/debug/pprof/profile?seconds=60" > cpu.pprof
```

    Replace `<password>` with the `agent_id` or your custom password, and `<port>` with the listening port.

### Analyze the profile

=== "Online visualization"
    Upload the `.pprof` file to [pprof.me](https://pprof.me) to explore it interactively using flame graphs and call trees.

=== "Using Go locally"
    If Go is installed, run the following command to open an interactive web interface for exploring the profile:
    
    ```bash
        go tool pprof -http=:8080 heap.pprof
    ```

### Available profiles

| Endpoint | Description |
|----------|-------------|
| `/debug/pprof/heap` | Memory allocation profile |
| `/debug/pprof/profile?seconds=60` | CPU profile (60 seconds) |
| `/debug/pprof/goroutine` | Goroutine stack traces |
| `/debug/pprof/` | Index of all available profiles |