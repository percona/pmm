# Connect ProxySQL databases to PMM

Monitor your ProxySQL instances with Percona Monitoring and Management (PMM) to track performance metrics and gain insights into query routing behavior.

## Prerequisites

Before adding a ProxySQL instance to PMM:

- ensure PMM Server is running and accessible
- verify PMM Client is installed on the host running ProxySQL
- configure a dedicated read-only user in ProxySQL for monitoring purposes

Use the `proxysql` alias to enable ProxySQL performance metrics monitoring.

## Add ProxySQL service

Add your ProxySQL instance to PMM using the proxysql service type:

### Basic usage

```sh
pmm-admin add proxysql --username=pmm --password=pmm
```

Replace `pmm` with the credentials for your ProxySQL administration interface. For security, configure a dedicated read-only monitoring account using the [`admin-stats_credentials`](https://proxysql.com/documentation/global-variables/admin-variables/#admin-stats_credentials) variable in ProxySQL.

!!! warning
    The monitoring user needs admin read-only permissions in ProxySQL to collect runtime server metrics. Without these permissions, PMM will skip runtime server metrics collection.

You can append two optional positional arguments: a service name and a service address. If omitted, PMM uses `<node>-proxysql` and `127.0.0.1:6032` as defaults.

## Example output

The output of this command may look as follows:

```sh
pmm-admin add proxysql --username=pmm --password=pmm
```

```text
ProxySQL Service added.
Service ID  : f69df379-6584-4db5-a896-f35ae8c97573
Service name: ubuntu-proxysql
```

## Configuration options

You can customize the ProxySQL service configuration using command-line flags. These flags provide more control than positional arguments and take higher priority when both are specified.

### Service identification flags

- `--service-name`: Custom name for the ProxySQL service in PMM
- `--host`: Hostname or IP address of the ProxySQL instance  
- `--port`: Port number for ProxySQL admin interface
- `--socket`: UNIX socket path (alternative to host/port)

### Disable collectors

Use `--disable-collectors` to exclude specific collectors from metric collection. This can reduce monitoring overhead or suppress metrics that are not relevant to your environment:

```sh
pmm-admin add proxysql \
  --username=pmm \
  --password=pmm \
  --disable-collectors=mysql_connection_list,stats_memory_metrics
```

??? info "Available ProxySQL collectors"
    `mysql_connection_list`, `mysql_connection_pool`, `mysql_status`, `runtime_mysql_servers`, `stats_command_counter`, `stats_memory_metrics`

### Connection examples

**TCP connection with custom service name:**
```sh
pmm-admin add proxysql \
  --username=pmm \
  --password=pmm \
  --service-name=my-new-proxysql \
  --host=127.0.0.1 \
  --port=6032
