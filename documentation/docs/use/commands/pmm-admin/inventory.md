# pmm-admin inventory

Use `pmm-admin inventory` from the command line to list registered services and agents, and modify agent configurations without removing and re-adding services.

To manage inventory in the UI, see [PMM Inventory](../../inventory.md). For programmatic access, see the [PMM API](../../../api/index.md).

## Commands

| Command | Use it to |
|---------|-----------|
| [`pmm-admin inventory list`](#pmm-admin-inventory-list) | View all registered nodes, services, and agents |
| [`pmm-admin inventory change agent`](#pmm-admin-inventory-change-agent) | Modify agent configuration without removing the service |

## pmm-admin inventory list

View all nodes, services, and agents registered with PMM Server:

```bash
pmm-admin inventory list [FLAGS]
```

### Flags

| Flag | Description |
|------|-------------|
| `--agents` | List only agents |
| `--services` | List only services |
| `--nodes` | List only nodes |

### Examples

List all inventory items:

```bash
pmm-admin inventory list
```

List only agents:

```bash
pmm-admin inventory list --agents
```

List only services:

```bash
pmm-admin inventory list --services
```

## pmm-admin inventory change agent

Modify agent configuration without removing and re-adding the service. Use this to update collector settings, enable or disable features, or change connection parameters.

!!! note "PMM 3.7.0+"
    This command is available starting with PMM 3.7.0.

### Syntax

```bash
pmm-admin inventory change agent <AGENT_TYPE> <AGENT_ID> [FLAGS]
```

### Supported agent types

Currently supports MongoDB agents only:

- `mongodb-exporter`
- `mongodb-profiler-agent`
- `qan-mongodb-profiler-agent`
- `rta-mongodb-agent`

### Behavior

- Only the flags you specify are updated. All other settings remain unchanged
- Changes take effect immediately without restarting the agent
- The command fails with a clear error if the agent ID doesn't exist or the type doesn't match

### When to use change agent vs remove/add

| Scenario | Recommended approach |
|----------|---------------------|
| Enable/disable a collector | `change agent` |
| Update collection limits | `change agent` |
| Change TLS settings | `change agent` |
| Change database credentials | `remove` then `add` |
| Change service name | `remove` then `add` |
| Switch to different database instance | `remove` then `add` |

### Finding the agent ID

Get the agent ID from the inventory list:

```bash
pmm-admin inventory list --agents
```

Look for the agent ID in the output (format: `/agent_id/...`).

### Examples

Enable all MongoDB collectors:

```bash
pmm-admin inventory change agent mongodb-exporter /agent_id/abc123 \
  --enable-all-collectors
```

Disable a specific collector:

```bash
pmm-admin inventory change agent mongodb-exporter /agent_id/abc123 \
  --disable-collectors=topmetrics
```

Change collection limit:

```bash
pmm-admin inventory change agent mongodb-exporter /agent_id/abc123 \
  --max-collections-limit=500
```

Update stats collections:

```bash
pmm-admin inventory change agent mongodb-exporter /agent_id/abc123 \
  --stats-collections=db1,db2.collection1
```

### Available flags for MongoDB agents

| Flag | Description |
|------|-------------|
| `--enable-all-collectors` | Enable all collectors |
| `--disable-collectors` | Comma-separated list of collectors to disable |
| `--max-collections-limit` | Max collections to monitor (-1=PMM decides, 0=unlimited) |
| `--stats-collections` | Limit stats to specific databases/collections |
| `--tls` | Enable TLS |
| `--tls-skip-verify` | Skip TLS certificate validation |
| `--tls-ca-file` | Path to CA certificate |
| `--tls-certificate-key-file` | Path to combined cert/key file |

## See also

- [pmm-admin add](add.md) 
- [Configuration commands](config.md) 
- [Status and diagnostics](status.md)