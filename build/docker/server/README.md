# PMM Server

Percona Monitoring and Management (PMM) is an open source database observability, monitoring, and management tool for MySQL, PostgreSQL, MongoDB and ProxySQL, and their servers. With PMM, you can identify critical performance issues faster, understand the root cause of incidents better, and troubleshoot them more efficiently.

- The tool allows you to view node to single query performance metrics and explain plans for all of your databases in a single place.
- With Query Analytics, you can quickly locate costly and slow running queries and drill into precise execution details to address bottlenecks.
- Percona Advisors equip you with performance, security and configuration recommendations that help you keep your databases performing at their best.
- Alerting and management features like backup and restore are designed to increase the velocity of your IT team.

## Starting PMM Server

```
docker pull percona/pmm-server:3
docker volume create pmm-data
docker run --detach --restart always --publish 443:8443 -v pmm-data:/srv --name pmm-server percona/pmm-server:3
```

Point your browser to https://hostname:443

This example uses the tag `:3` to pull the latest PMM 3.x version, but other, [more specific tags](https://hub.docker.com/r/percona/pmm-server/tags), are also available.

## Environment variables

You can use these environment variables (-e VAR) when running the Docker image.

| Variable                     | Description                                                                                                                 |
|------------------------------|-----------------------------------------------------------------------------------------------------------------------------|
| PMM_ENABLE_UPDATES          | Disable automatic updates                                                                                                   |
| PMM_ENABLE_TELEMETRY        | Disable built-in telemetry and disable STT if telemetry is disabled                                                         |
| PMM_ENABLE_ALERTING         | Disable Percona alerting                                                                                                    |
| PMM_METRICS_RESOLUTION       | High metrics resolution in seconds                                                                                          |
| PMM_METRICS_RESOLUTION_HR    | High metrics resolution (same as above)                                                                                     |
| PMM_METRICS_RESOLUTION_MR    | Medium metrics resolution in seconds                                                                                        |
| PMM_METRICS_RESOLUTION_LR    | Low metrics resolution in seconds                                                                                           |
| PMM_DATA_RETENTION           | How long to keep time-series data in ClickHouse. This variable accepts golang style duration format, example: 24h, 30m, 10s |
| PMM_ENABLE_VM_CACHE          | Enable cache in VM                                                                                                          |
| PMM_ENABLE_AZURE_DISCOVER     | Enable support for discovery of Azure databases                                                                             |
| PMM_PUBLIC_ADDRESS           | External IP address or the DNS name on which PMM server is running.                                                         |
| PMM_DEBUG                    | Enables a more verbose log level                                                                                            |
| PMM_TRACE                    | Enables a more verbose log level including traceback information                                                            |

## For more information please visit:

[Percona Monitoring and Management](https://docs.percona.com/percona-monitoring-and-management)

[Setting up PMM Server with Docker](https://docs.percona.com/percona-monitoring-and-management/3/install-pmm/install-pmm-server/index.html)
