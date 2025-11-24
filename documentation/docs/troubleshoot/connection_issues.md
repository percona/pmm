
# Connection Issues: PMM Clients Disconnecting

## Problem
When deploying hundreds of PMM clients, you may observe frequent disconnects or instability in PMM Server. This typically manifests as agents losing connection, failed metric collection, or intermittent errors in the PMM UI.

## Cause
The default configuration for PMM Server's internal PostgreSQL and Grafana cache may not be sufficient to handle a large number of concurrent connections from many PMM clients. This can lead to resource exhaustion, connection drops, or degraded performance.

## Solution
Increase the following environment variables to improve stability and support more PMM clients:

- `PMM_POSTGRES_MAX_CONNECTIONS`: Increase this value to allow more concurrent database connections. Default: 500. Example: `PMM_POSTGRES_MAX_CONNECTIONS=1000`
- `PMM_POSTGRES_SHARED_BUFFERS`: Increase shared buffers for PostgreSQL to improve performance. Default: 256MB. Example: `PMM_POSTGRES_SHARED_BUFFERS=512MB`
- `PMM_GRAFANA_CACHE_INVALIDATION_PERIOD`: Increase the cache invalidation period to reduce load on Grafana. Default: 3 seconds. Example: `PMM_GRAFANA_CACHE_INVALIDATION_PERIOD=30s`

Set these variables in your PMM Server environment (Docker, Compose, or systemd unit), then restart PMM Server and PostgreSQL for the changes to take effect.

## Example (Docker)
```sh
docker run -d \
  -e PMM_POSTGRES_MAX_CONNECTIONS=1000 \
  -e PMM_GRAFANA_CACHE_INVALIDATION_PERIOD=10s \
  -e PMM_POSTGRES_SHARED_BUFFERS=512MB \
  ...other options... \
  percona/pmm-server:latest
```

## Additional Notes
- Monitor PMM Server logs for connection errors and adjust values as needed.
- Ensure your server hardware and resources are sufficient for the increased limits.
- For very large deployments, consider using an external PostgreSQL database or [High Availability setup](../install-pmm/HA.md).
