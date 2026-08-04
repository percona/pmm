# Troubleshoot upgrade issues

## PMM Server not updating correctly

If PMM Server is not updating correctly, check the container logs for errors:

```sh
docker logs pmm-server
```

If the issue persists, restore from a backup and retry the upgrade. See [Restore PMM Server](../install-pmm/install-pmm-server/deployment-options/docker/restore_container.md).
