# Troubleshoot upgrade issues

## PMM Server not updating correctly

If PMM Server is not updating correctly, check the container logs for errors:

```sh
docker logs pmm-server
```

If the issue persists, restore from a backup and retry the upgrade. See [Restore PMM Server](../install-pmm/install-pmm-server/deployment-options/docker/restore_container.md).

## Corrupted credentials after encryption key rotation

If you ran `pmm-encryption-rotation` before upgrading to PMM 3.9.1, TLS/SSL certificates and keys or cloud credentials for some services may be corrupted. Remove and re-add the affected services to recreate them:

```sh
pmm-admin remove <service-type> <service-name>
pmm-admin add <service-type> ... # supply the original TLS/cloud credentials
```
