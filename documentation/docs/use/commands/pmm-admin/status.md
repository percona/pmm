# pmm-admin status and diagnostics

Check PMM Client connection status, list monitored services, and create diagnostic archives from the command line using these `pmm-admin` commands.

To view status in the UI, see [PMM Inventory](../pmm-admin/inventory.md). For programmatic access, see the [PMM API](../../../api/index.md).

## Commands

| Command | Use it to |
|---------|-----------|
| [`pmm-admin status`](#pmm-admin-status) | Check if PMM Client can connect to PMM Server |
| [`pmm-admin list`](#pmm-admin-list) | Show all monitored services and their agents |
| [`pmm-admin summary`](#pmm-admin-summary) | Create a diagnostic archive for troubleshooting or support requests |

## pmm-admin status

Check the connection between PMM Client and PMM Server. Use this to verify PMM Client is properly configured and can communicate with the server.

### Syntax

```bash
pmm-admin status [FLAGS]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |
| `--wait=<period><unit>` | Time to wait for a successful response from pmm-agent. Use an integer followed by a unit: `ms` (milliseconds), `s` (seconds), `m` (minutes), `h` (hours). Example: `--wait=30s` |

### Example output

```
Agent ID : abcd1234-5678-90ef-ghij-klmnopqrstuv
Node ID  : /node_id/abcd1234-5678-90ef-ghij-klmnopqrstuv
Node name: db-server-01

PMM Server:
    URL    : https://192.168.1.100:443/
    Version: 3.0.0

PMM Client:
    Connected        : true
    Time drift       : -1.234ms
    Latency          : 5.678ms
    pmm-agent Version: 3.0.0
    pmm-agent Uptime : 48h30m15s
```

### Interpret the output

| Field | What it means |
|-------|---------------|
| `Connected: true` | PMM Client is communicating with PMM Server |
| `Connected: false` | Connection failed — check server URL and network |
| `Time drift` | Clock difference between client and server (keep under 1 second) |
| `Latency` | Network round-trip time to server |

### Wait for connection

Use the `--wait` flag to wait for the agent to become ready, for example in automated scripts or startup sequences:

```bash
pmm-admin status --wait=30s
```

### Troubleshoot connection issues

#### Connection refused

```bash
pmm-admin status
# Error: ...connection refused
```

Check that PMM Server is running and the URL is correct:

```bash
pmm-admin config --server-url=https://username:password@192.168.1.100:443
```

#### Certificate errors

```bash
pmm-admin status
# Error: ...certificate signed by unknown authority
```

For self-signed certificates, use:

```bash
pmm-admin config --server-url=https://admin:admin@192.168.1.100:443 --server-insecure-tls
```

#### High time drift

If time drift exceeds 1 second, synchronize clocks using NTP:

```bash
sudo systemctl start chronyd
# or
sudo ntpdate pool.ntp.org
```

## pmm-admin list

Show all services and agents registered on this node, including the agent metrics mode (push/pull). Use this to verify which databases are being monitored and check agent status.

### Syntax

```bash
pmm-admin list [FLAGS]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |

### Example output

```
Service type        Service name        Address and port        Service ID
MySQL               mysql-prod          192.168.1.10:3306       abc123
MongoDB             mongodb-prod        192.168.1.20:27017      /service_id/def456
PostgreSQL          postgres-prod       192.168.1.30:5432       /service_id/ghi789

Agent type                  Status      Metrics Mode      Agent ID                              Service ID
pmm_agent                   Connected                     /agent_id/xyz789
node_exporter               Running     push              /agent_id/node123
mysqld_exporter             Running     push              /agent_id/mysql123                    /service_id/abc123
mongodb_exporter            Running     push              /agent_id/mongo123                    /service_id/def456
postgres_exporter           Running     push              /agent_id/pg123                       /service_id/ghi789
```

### Interpret agent status

| Status | What it means |
|--------|---------------|
| `Running` | Agent is collecting metrics normally |
| `Waiting` | Agent is starting or waiting to connect |
| `Stopping` | Agent is shutting down |
| `Done` | Agent has stopped |
| `Unknown` | Cannot determine agent status |

### Filter output with grep

Find MySQL services:

```bash
pmm-admin list | grep -i mysql
```

Find agents with issues:

```bash
pmm-admin list | grep -v Running
```

## pmm-admin summary

Create a diagnostic archive containing logs, configuration, and status information. Use this when troubleshooting issues or submitting support requests to Percona.

The default archive filename is `summary_<hostname>_<year>_<month>_<date>_<hour>_<minute>_<second>.zip`, created in the current directory.

The archive includes:

- PMM Client logs
- Agent configuration files
- Service status and inventory
- System information (OS, hardware)
- Network diagnostics

### Syntax

```bash
pmm-admin summary [FLAGS]
```

### Flags

| Flag | Description |
|------|-------------|
| `--filename=PATH` | Output file path (default: auto-generated in current directory) |
| `--skip-server` | Skip collecting PMM Server logs (`logs.zip`) |
| `--pprof` | Include Go performance profiling data (for advanced debugging) |
| `--json` | Output summary metadata in JSON format |

### Examples

Create a diagnostic archive:

```bash
pmm-admin summary
```

Output:

```
Created summary file: /home/user/summary_db-server-01_2024-01-15T10-30-00.zip
```

Save to a specific location:

```bash
pmm-admin summary --filename=/tmp/pmm-diagnostic.zip
```

Create archive without contacting PMM Server:

```bash
pmm-admin summary --skip-server
```

Include profiling data for performance issues:

```bash
pmm-admin summary --pprof
```

### What's in the archive

```
summary_db-server-01_2024-01-15T10-30-00/
├── client/
│   ├── pmm-agent.log
│   ├── pmm-agent.yaml
│   ├── status.json
│   └── list.txt
├── systeminfo/
│   ├── os-release
│   ├── uname.txt
│   └── df.txt
└── server/
    └── logs.zip (if --skip-server not used)
```

### Submit to Percona Support

When opening a support ticket:
{.power-number}

1. Create the summary archive:

    ```bash
    pmm-admin summary
    ```

2. Attach the `.zip` file to your support ticket.

3. Include a description of the issue, steps to reproduce, and when it started.

## See also

- [pmm-admin overview](../pmm-admin/pmm-admin.md)
- [Add database services to monitoring](../pmm-admin/add.md)
- [Manage inventory](../pmm-admin/inventory.md)
- [Configuration commands](../pmm-admin/config.md)