# Preview environment variables

!!! caution alert alert-warning "Warning"
     The `PMM_TEST_*` environment variables are experimental and subject to change. It is recommended that you use these variables for testing purposes only and not on production.

| Variable                               | Description
| -------------------------------------- | --------------------------------------------------------------------------------------------------------
| `PMM_TEST_HA_ENABLE`                  | Enable PMM to run in High Availability (HA) mode.
| `PMM_TEST_HA_BOOTSTRAP`                | Bootstrap HA cluster.
| `PMM_TEST_HA_NODE_ID`                  | HA Node ID.
| `PMM_TEST_HA_ADVERTISE_ADDRESS`        | HA Advertise address.
| `PMM_TEST_HA_GOSSIP_PORT`              | HA gossip port.
| `PMM_TEST_HA_RAFT_PORT`                | HA raft port.
| `PMM_TEST_HA_GRAFANA_GOSSIP_PORT`      | HA Grafana gossip port.
| `PMM_TEST_HA_PEERS`                    | HA Peers.
