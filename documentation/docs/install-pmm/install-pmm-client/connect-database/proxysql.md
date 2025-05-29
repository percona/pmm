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

where `username` and `password` are credentials for the administration interface of the monitored ProxySQL instance. 
You should configure a read-only account for monitoring using the [`admin-stats_credentials`](https://proxysql.com/documentation/global-variables/admin-variables/#admin-stats_credentials) variable in ProxySQL

Additionally, two positional arguments can be appended to the command line flags: a service name to be used by PMM, and a service address. If not specified, they are substituted automatically as `<node>-proxysql` and `127.0.0.1:6032`.

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

### Connection examples

**TCP connection with custom service name:**
```sh
pmm-admin add proxysql \
  --username=pmm \
  --password=pmm \
  --service-name=my-new-proxysql \
  --host=127.0.0.1 \
  --port=6032