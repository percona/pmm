# MongoDB Log-based Query Analytics (mongolog)

PMM supports collecting MongoDB query metrics from slow query logs instead of the profiler by using the `mongolog` query source. Mongolog parses MongoDB's slow query logs from disk in real time.

This solves connection pool exhaustion issues that occur with the traditional profiler approach, particularly in high-traffic environments with hundreds of databases.

When using the standard profiler method, PMM-Agent queries compete with application traffic for database connections. In environments with 600+ databases this leads to severe errors like:

*couldn't create system.profile iterator, reason: timed out while checking out a connection from connection pool: context deadline exceeded; maxPoolSize: 100, connections in use by cursors: 0, connections in use by transactions: 0, connections in use by other operations: 100*

This occurs because:

- PMM-Agent tries to query `system.profile` collections across all databases
- each query consumes a connection from the pool
- with hundreds of databases, all 100 connections get exhausted
- new monitoring queries timeout waiting for connections, which leads to missing query analytics (QAN) data and blind spots in monitoring

## How mongolog works

The `mongolog` query source eliminates this problem by reading MongoDB's slow query logs directly from disk, completely bypassing database connections entirely. This file-based approach:

- does not query `system.profile` collections
- uses zero database connections for metrics collection
- scales to any number of databases without performance degradation
- provides identical query analytics data in PMM

## When to use mongolog

Use mongolog when you:

- experience connection pool exhaustion errors (primary use case)
- manage 100+ databases on a single MongoDB instance
- see pmm-agent timeout errors in your logs
- need to avoid profiler performance overhead
- work in restricted environments where `system.profile` is unavailable (mongos, managed services)
- require consistent metrics collection without gaps
- have production workloads where the profiler is too heavy

## Prerequisites

- MongoDB 5.0+ (tested with 5.0.20-17)
- Write access to MongoDB log directory
- Log file readable by PMM Agent user

## Configure MongoDB
Configure MongoDB to log slow operations to a file using either a config file or command-line flags. 

This requires enabling slow operation logging (not full profiling). Only slow operation logging is required, full profiling is not needed.

=== "Config file (recommended)"
    This configuration logs slow operations to a file, appends instead of overwriting, and sets the threshold to 100ms. Ensure the log file is readable by the user running the PMM Agent.

    Configure MongoDB using `mongod.conf`:

    ```yaml
    systemLog:
      destination: file
      path: /var/log/mongodb/mongod.log
      logAppend: true

    operationProfiling:
      mode: slowOp
      slowOpThresholdMs: 100
    ```

    **Important**:

    - `mode: slowOp` logs operations to file only (does NOT populate system.profile)
    - Set `slowOpThresholdMs` based on your performance requirements (100ms is a good starting point)

=== "Command-line flags"
    Alternatively, start `mongod` with the flags below.
    These flags can be adapted to your deployment automation (Docker, systemd, etc):

    ```bash
    mongod \
      --dbpath /var/lib/mongo \
      --logpath /var/log/mongodb/mongod.log \
      --logappend \
      --profile 1 \
      --slowms 100
    ```

    **Flag descriptions**

    | Flag | Purpose |
    |----------------|--------------------------------------------------------|
    | `--logpath` | Enables logging to a file (required by mongolog) |
    | `--logappend` | Appends to the log file instead of overwriting |
    | `--profile 1` | Enables logging of slow operations (not full profiling). `--profile 1` only logs slow operations to file, it does **not** populate `system.profile` collection |
    | `--slowms 100` | Sets slow operation threshold (in milliseconds) |
    | `--dbpath` | Required if no config file is used |

## Add MongoDB with mongolog to PMM
After configuring MongoDB to log slow operations to a file, the final step is to register your instance with PMM using the mongolog query source. This tells PMM to collect query analytics from log files instead of the system profiler.
{.power-number}

1. Register the MongoDB instance with `mongolog` as the query source. Use `--query-source=mongolog` to enable log-based query analytics, MongoDB credentials for metadata collection, and the MongoDB endpoint (defaults to `127.0.0.1:27017`):

```sh
pmm-admin add mongodb \
  --query-source=mongolog \
  --username=pmm \
  --password=your_secure_password \
  127.0.0.1
```
2. Verify the configuration with `pmm-admin status`. In the output, look for `mongodb_profiler_agent`. It should show the agent is running with mongolog as the query source.

## Configure log rotation

Proper log rotation is critical for mongolog to continue functioning. Ensure `mongolog` continues reading logs after rotation:

```txt
/var/log/mongodb/mongod.log {
   daily
   rotate 7
   compress
   delaycompress
   copytruncate
   missingok
   notifempty
   create 640 mongodb mongodb
}
```
### Critical requirements

- Use `copytruncate` as this preserves file handle for mongolog
- Avoid moving/renaming log files because this breaks mongolog's file tail
- Do not delete active log files during rotation

## View metrics in PMM

Once configured, slow query metrics from `mongolog` appear in Query Analytics (QAN) with identical functionality to profiler-based collection, regardless of which query source you choose:

- query fingerprints and statistics
- performance metrics and trends  
- database and collection breakdowns
- full query details and examples

Performance impact is virtually zero since metrics are sourced from existing log files.

## Compare profiler vs mongolog

| Feature                    | `profiler`           | `mongolog`          |
|----------------------------|----------------------|---------------------|
| Database connections       | Uses pool         | None required    |
| Connection pool impact     | High              | Zero             |
| Requires `system.profile`  | Yes               | No               |
| Supports `mongos`          | No                | Yes              |
| Database overhead          | Moderate-High     | Minimal          |
| File-based logging         | No                | Yes              |
| Query history durability   | Volatile          | Disk-persisted   |
| Scales with DB count       | Linear degradation| Constant         |
