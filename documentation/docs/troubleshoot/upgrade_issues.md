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

## PMM HA: helm upgrade fails with "field is immutable"

If `helm upgrade` on a [PMM HA](../pmm-upgrade/upgrade_helm_ha.md) release fails with:

```text
Error: UPGRADE FAILED: cannot patch "<release>-pmm-token-init" with kind Job:
Job.batch "<release>-pmm-token-init" is invalid: spec.template: Invalid value: ...: field is immutable
```

the upgrade changed a value that the chart's `pmm-token-init` Job's pod spec depends on (for example `secret.name`), and a Kubernetes Job's pod template can't be patched in place. Delete the stale Job so Helm can recreate it, then retry the upgrade:

```sh
kubectl delete job <release>-pmm-token-init -n <namespace>
helm upgrade ... # retry your original command
```

This is intermittent because the Job self-deletes after 24 hours (`ttlSecondsAfterFinished`)—an upgrade attempted more than a day after install or the last upgrade won't hit it. Chart versions with this fix name the Job uniquely per pod-spec change instead, so the conflict can't occur.
