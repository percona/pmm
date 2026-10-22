# Preview environment variables

!!! caution alert alert-warning "Preview features"
    The variables on this page are in preview and can change in future releases. Use them for testing only, not in production.

    For production-ready configuration options, see [Environment variables for PMM Server](../docker/env_var.md).

## HA variables

Use these variables to set up PMM Server nodes for High Availability (HA) yourself, for example with Docker. For a supported HA deployment, use [PMM HA Cluster](../../../HA-clustered.md) instead.

| Variable                          | Description
| --------------------------------- | --------------------------------------------------------------------------------------------------------
| `PMM_HA_ENABLE`                   | Enable PMM to run in High Availability (HA) mode.
| `PMM_HA_NODE_ID`                  | HA Node ID.
| `PMM_HA_ADVERTISE_ADDRESS`        | HA Advertise address.
| `PMM_HA_GOSSIP_PORT`              | HA gossip port.
| `PMM_HA_RAFT_PORT`                | HA raft port.
| `PMM_HA_GRAFANA_GOSSIP_PORT`      | HA Grafana gossip port.
| `PMM_HA_PEERS`                    | HA Peers.

!!! caution "All HA nodes must share one encryption key"
    HA nodes share one PostgreSQL database, but each node reads its encryption key from its own `/srv/pmm-encryption.key`. A node with a different key cannot decrypt the credentials stored by the other nodes, so PMM stops monitoring those services.

    Generate the key once as described in [PMM data encryption](../../../../admin/security/data_encryption.md#ha-clusters-use-the-same-key-on-every-node), place it at `/srv/pmm-encryption.key` on every node before starting them. With `PMM_HA_ENABLE` set, a node without a key file refuses to start rather than generating one of its own.

## External ClickHouse variables

| Variable                          | Description
| ----------------------------------| --------------------------------------------------------------------------
| `PMM_CLICKHOUSE_ADDR`             | Host and port of the external ClickHouse database instance
| `PMM_CLICKHOUSE_DATABASE`         | Database name of the external ClickHouse instance
| `PMM_CLICKHOUSE_USER`             | Database user
| `PMM_CLICKHOUSE_PASSWORD`         | Database user password
| `PMM_CLICKHOUSE_DATASOURCE_USER`  | Read-only database user for the Grafana ClickHouse data source
| `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` | Read-only data source user password

## Testing external ClickHouse connections
The ClickHouse-related preview variables are useful for testing PMM Server with an external ClickHouse instance:
{.power-number}
    
1. Set up a test ClickHouse instance.
2. Configure the connection using the variables above.
3. Launch PMM Server and verify it connects and stores metrics correctly.
4. Monitor logs to validate how metrics display.

