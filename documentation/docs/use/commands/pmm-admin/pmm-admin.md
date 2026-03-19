# pmm-admin command overview

`pmm-admin` is the command-line tool for managing your PMM monitoring setup. Use it to add databases, check connection status, update agent configurations, and troubleshoot issues from your terminal.

`pmm-admin` is installed automatically with PMM Client.

To add services through the UI instead, see [Connect databases via the web interface](../../../install-pmm/install-pmm-client/connect-database/index.md). For programmatic access, see the [PMM API](../../../api/index.md).

Use `pmm-admin` to:

- add MySQL, PostgreSQL, MongoDB, Valkey, ProxySQL, and HAProxy services to monitoring
- check connection status between PMM Client and PMM Server
- list monitored services and their agents
- modify agent configurations without removing services
- create diagnostic archives for troubleshooting

## Syntax

Run `pmm-admin` commands in this format:

```bash
pmm-admin [FLAGS] COMMAND [COMMAND-FLAGS] [ARGUMENTS]
```

## Quick start

Try these common commands to verify your setup and start monitoring:

### Check PMM Client status

```bash
pmm-admin status
```

### Add a MySQL database

```bash
pmm-admin add mysql --username=pmm --password=pass mysql-prod 192.168.1.10:3306
```

### Add a MongoDB database

```bash
pmm-admin add mongodb --username=pmm --password=pass mongodb-prod 192.168.1.20:27017
```

### List monitored services

```bash
pmm-admin list
```

### Create diagnostic archive

```bash
pmm-admin summary
```

For complete options and flags, see [Add database services](../pmm-admin/add.md), [Manage inventory](../pmm-admin/inventory.md), [Configuration commands](../pmm-admin/config.md), and [Status and diagnostics](../pmm-admin/status.md).

## Command reference

Find all available commands for managing monitored services:

| Command | Description | Documentation |
|---------|-------------|---------------|
| `pmm-admin add` | Add database services to monitoring | [Add database services](../pmm-admin/add.md) |
| `pmm-admin inventory` | List and modify agents and services | [Manage inventory](../pmm-admin/inventory.md) |
| `pmm-admin config` | Configure local pmm-agent | [Configuration commands](../pmm-admin/config.md) |
| `pmm-admin register` | Register node with PMM Server | [Configuration commands](../pmm-admin/config.md) |
| `pmm-admin remove` | Remove service from monitoring | [Configuration commands](../pmm-admin/config.md) |
| `pmm-admin annotate` | Add event annotations | [Configuration commands](../pmm-admin/config.md) |
| `pmm-admin status` | Show PMM Client status | [Status and diagnostics](../pmm-admin/status.md) |
| `pmm-admin list` | List monitored services | [Status and diagnostics](../pmm-admin/status.md) |
| `pmm-admin summary` | Create diagnostic archive | [Status and diagnostics](../pmm-admin/status.md) |

## Command reference

### Add and remove services

- [`pmm-admin add`](pmm-admin-add.md) — Add database services to monitoring
- [`pmm-admin remove`](pmm-admin-config.md) — Remove service from monitoring

### Manage inventory

- [`pmm-admin inventory`](pmm-admin-inventory.md) — List and modify agents and services

### Configure and register

- [`pmm-admin config`](pmm-admin-config.md) — Configure local pmm-agent
- [`pmm-admin register`](pmm-admin-config.md) — Register node with PMM Server
- [`pmm-admin annotate`](pmm-admin-config.md) — Add event annotations

### Status and troubleshooting

- [`pmm-admin status`](pmm-admin-status.md) — Show PMM Client status
- [`pmm-admin list`](pmm-admin-status.md) — List monitored services
- [`pmm-admin summary`](pmm-admin-status.md) — Create diagnostic archive

## Get help

Run `--help` with any command to see available flags and usage:

```bash
pmm-admin COMMAND --help
```

For example:

```bash
pmm-admin add mysql --help
pmm-admin inventory change agent --help
```

## See also

- [PMM Client agent](../pmm-agent.md)
- [Connect databases to PMM](../../../install-pmm/install-pmm-client/connect-database/index.md)
- [Remove databases from monitoring](../../remove-services.md)