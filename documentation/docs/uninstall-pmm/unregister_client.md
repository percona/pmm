# Unregister PMM Client from PMM Server

Unregistering disconnects your PMM Client from the PMM Server and removes all monitoring services for this node. This should be done before uninstalling the PMM Client.

To unregister PMM Client from PMM Server, run the following command:

```sh
pmm-admin unregister --force
```

This command: 

- stops all monitoring services on this node
- removes the node from PMM Server's inventory
- cleans up monitoring data collection
- prevents orphaned entries on the server

!!! note
    The `--force` flag ensures unregistration even if the PMM Server is unreachable. The PMM Client software remains installed after unregistering.
