# Back up and restore PMM HA Cluster

Back up your PMM HA Cluster to S3-compatible storage or a shared volume, and restore it in place or into another namespace for disaster recovery. Backups are optional and turned off by default.

## Default values used in examples

The examples assume the default namespace `pmm` and release name `pmm-ha`. If you used different values, replace `pmm` and `pmm-ha` in the commands. Resource names such as `pmm-ha-backup-tools` start with your release name.

## What gets backed up

A backup counts as complete only when every selected component succeeds. If any component fails, the backup is marked `partial` and is never used as the latest backup.

Each backup covers your whole HA installation. Every component is backed up with its own native tool:

| Component | What it holds | Backup tool |
|-----------|---------------|-------------|
| PostgreSQL | PMM Server and Grafana settings, inventory, users | `pg_dump` |
| ClickHouse | Query Analytics data | `clickhouse-backup` |
| VictoriaMetrics | Metrics | `vmbackup` |
| PMM Server `/srv` | PMM Server data on each replica | `tar` |
| Encryption key | The key PMM uses to encrypt stored credentials | Exported from its Kubernetes Secret |


## Before you begin

Before you turn on backups, make sure that:

- You have somewhere to store backups: an S3-compatible bucket (Amazon S3, MinIO, Ceph, and similar), or a `ReadWriteMany` volume such as EFS or NFS. `ReadWriteOnce` volumes such as EBS `gp3` can't be used for a shared volume.
- You can tolerate a short restart. Turning on backups adds backup containers to the ClickHouse, VictoriaMetrics, and PMM Server pods, so those pods restart during the `helm upgrade`.

## Turn on backups

Choose where to store your backups, then add the matching settings to your `values.yaml` file:

=== "S3-compatible storage"

    Use this option for any S3-compatible storage, including Amazon S3, MinIO, and Ceph:
    {.power-number}

    1. Create a bucket and an access key with read, write, list, and delete permissions on it.

    2. Store the access key in a Secret in the PMM HA namespace:

        ```sh
        kubectl create secret generic pmm-s3-secret -n pmm \
          --from-literal=access-key=<ACCESS_KEY> \
          --from-literal=secret-key=<SECRET_KEY>
        ```

    3. Add the backup settings to your `values.yaml` file. ClickHouse and VictoriaMetrics inherit their S3 settings from `centralBackupStorage.s3`, so only `enabled: true` is needed for each:

        ```yaml
        centralBackupStorage:
          enabled: true
          mode: s3
          s3:
            bucket: "pmm-backups"
            region: "us-east-1"
            endpoint: ""                    # Leave empty for Amazon S3. For example: http://minio.minio.svc:9000
            provider: "AWS"                 # AWS, Minio, Ceph, or Other
            existingSecret: "pmm-s3-secret"
        clickhouse:
          backup:
            s3:
              enabled: true
        victoriaMetrics:
          vmstorage:
            backup:
              s3:
                enabled: true
        ```

    4. Apply the changes:

        ```sh
        helm upgrade pmm-ha percona/pmm-ha --namespace pmm -f values.yaml
        ```

=== "Amazon S3 with IRSA (EKS)"

    On Amazon EKS, you can use IAM Roles for Service Accounts (IRSA) instead of access keys, so no credentials are stored in the cluster.
    {.power-number}

    1. Create an IAM policy that allows `s3:ListBucket`, `s3:GetObject`, `s3:PutObject`, and `s3:DeleteObject` on your bucket.

    2. Create an IAM role with that policy. In its trust policy, allow your cluster's OIDC provider for the PMM Server service account and the backup service accounts. A `StringLike` condition on `system:serviceaccount:pmm:pmm-ha-backup-*` covers both backup service accounts.

    3. Add the backup settings to your `values.yaml` file:

        ```yaml
        centralBackupStorage:
          enabled: true
          mode: s3
          s3:
            bucket: "pmm-backups"
            region: "eu-central-1"
            irsaRoleArn: "arn:aws:iam::<account-id>:role/pmm-ha-backup-s3"
        clickhouse:
          backup:
            s3:
              enabled: true
        victoriaMetrics:
          vmstorage:
            backup:
              s3:
                enabled: true
        ```

    4. Apply the changes:

        ```sh
        helm upgrade pmm-ha percona/pmm-ha --namespace pmm -f values.yaml
        ```

    Make sure the AWS STS regional endpoint is active for your cluster's region. Otherwise, every backup fails with `403 RegionDisabledException`.

=== "Shared volume"

    Use this option to store backups on a `ReadWriteMany` volume that you provide, such as EFS or an NFS export. The chart doesn't create `ReadWriteMany` storage for you.
    {.power-number}

    1. Create a `ReadWriteMany` PersistentVolumeClaim in the PMM HA namespace. For an NFS export, declare a PersistentVolume for it first and bind the claim to it.

    2. Add the backup settings to your `values.yaml` file:

        ```yaml
        centralBackupStorage:
          enabled: true
          mode: shared
          existingClaim: "pmm-backup-pvc"
        ```

    3. Apply the changes:

        ```sh
        helm upgrade pmm-ha percona/pmm-ha --namespace pmm -f values.yaml
        ```

    If you plan to restore into another namespace, make the export group-writable once, for example with `chmod -R g+rwX,g+s <export-path>`. On OpenShift, each namespace runs with a different user ID, so a restore from another namespace can't read backups the source namespace created otherwise.

Once backups are on, the chart creates a `pmm-ha-backup-tools` Deployment that runs the backup tool, and a `pmm-ha-backup` CronJob that stays suspended until you set a schedule.

## Back up PMM HA Cluster

=== "kubectl exec"

    To back up all components immediately:

    ```sh
    kubectl exec -n pmm deploy/pmm-ha-backup-tools -- pmm-backup.sh backup
    ```

    The command prints a summary for each component and exits with an error if any component failed.

    To back up only some components, add `--postgresql`, `--clickhouse`, `--victoriametrics`, or `--pmm-server`. To exclude a component instead, use the matching `--skip-` option, for example `--skip-victoriametrics`.

=== "Kubernetes Job"

    For large installations, run the backup as a Kubernetes Job instead. A Job keeps running if the node it runs on is replaced, while an interactive `kubectl exec` session doesn't:

    ```sh
    kubectl create job --from=cronjob/pmm-ha-backup manual-$(date +%s) -n pmm
    ```

### Back up on a schedule

To back up on a schedule, add the schedule to your `values.yaml` file and run `helm upgrade`:

```yaml
centralBackupStorage:
  schedule:
    enabled: true
    cron: "0 2 * * *"    # Daily at 02:00, in the cluster's time zone
    retentionDays: 7
```

Backups older than `retentionDays` are deleted automatically after each backup.

To check scheduled runs:

```sh
kubectl get cronjob -n pmm
kubectl get jobs -n pmm
```

### List backups

To list all backups, with the latest one marked:

```sh
kubectl exec -n pmm deploy/pmm-ha-backup-tools -- pmm-backup.sh list
```

To see every file in a specific backup, add its ID:

```sh
kubectl exec -n pmm deploy/pmm-ha-backup-tools -- pmm-backup.sh list <backup-id>
```

## Restore PMM HA Cluster

!!! warning "Restoring replaces your current data"
    A restore scales PMM Server and VictoriaMetrics down, replaces the PostgreSQL, ClickHouse, VictoriaMetrics, and PMM Server data, and scales them back up. PMM is unavailable during the restore, which typically takes 8 to 10 minutes.

### Check a restore first

Before a real restore, run a dry run. It checks that every component in the backup is present and readable, without changing anything:

```sh
kubectl exec -n pmm deploy/pmm-ha-backup-tools -- \
  pmm-backup.sh restore --backup-id latest --dry-run --yes
```

=== "Restore in place"

    To restore the latest backup into the same installation:

    ```sh
    kubectl exec -n pmm deploy/pmm-ha-backup-tools -- \
      pmm-backup.sh restore --backup-id latest --yes
    ```

    To restore a specific backup, replace `latest` with its ID from `pmm-backup.sh list`. The `--yes` option confirms the restore and is required.

    If your `kubectl exec` session is interrupted, `kubectl` reports an error but the restore keeps running inside the cluster. Don't run it again — check the restore log instead. The restore has finished when the log ends with a `PMM-HA Restore Summary` block:

    ```sh
    kubectl exec -n pmm deploy/pmm-ha-backup-tools -- \
      sh -c 'tail -20 $(ls -t /backups/logs/restore_*.log | head -1)'
    ```

    To avoid this, run the restore as a Job. The chart includes an example Job definition in `examples/restore-job.yaml`.

=== "Restore into another namespace"

    For disaster recovery, restore a backup into a separate PMM HA installation in another namespace, on the same or another Kubernetes cluster:
    {.power-number}

    1. Copy the source installation's `pmm-secret` into the new namespace **before** you install PMM HA there. The PostgreSQL, ClickHouse, and VictoriaMetrics operators set their passwords from this Secret at installation time, so it must match the source:

        ```sh
        kubectl create namespace pmm-dr
        kubectl -n pmm get secret pmm-secret -o yaml \
          | sed 's/namespace: pmm/namespace: pmm-dr/' \
          | kubectl apply -f -
        ```

    2. [Install PMM HA in the new namespace](install-HA-clustered.md#install-pmm-ha-into-multiple-namespaces) with backups turned on and pointing to the same storage as the source.

    3. Restore the source installation's backup. Run the command in the new namespace, and point `--s3-prefix` at the source installation, in the format `<source-namespace>/<source-release>`:

        ```sh
        kubectl exec -n pmm-dr deploy/pmm-dr-backup-tools -- \
          pmm-backup.sh restore --backup-id latest \
            --s3-prefix pmm/pmm-ha --yes
        ```

        If you store backups on a shared volume, use `--shared-source-path pmm/pmm-ha` instead of `--s3-prefix`.

    4. Log in with the source installation's admin password. The restore replaces the Grafana database, so the admin password is now the source's. If the new namespace's `pmm-secret` holds a different admin password, update it to match, otherwise the PMM Client pods fail to register:

        ```sh
        kubectl -n pmm-dr patch secret pmm-secret --type=merge \
          -p "{\"data\":{\"PMM_ADMIN_PASSWORD\":\"$(printf %s '<source-password>' | base64)\"}}"
        ```

    The restore also resets the PMM Client pods in the new namespace so they register again, and lists them in its summary. External PMM Clients keep working after you point them at the new installation.

## Limitations

- You can only restore a backup into a PMM HA installation running the same PMM version.
- Turning on backups restarts the ClickHouse, VictoriaMetrics, and PMM Server pods.
- The shared volume option requires `ReadWriteMany` storage.
- If a restore is interrupted, PMM can stay unavailable. See [Troubleshoot backup and restore](../troubleshoot/ha_issues.md#a-restore-was-interrupted-and-pmm-stays-down).

## Related topics

- [Install PMM HA Cluster](install-HA-clustered.md)
- [Upgrade PMM HA Cluster using Helm](../pmm-upgrade/upgrade_helm_ha.md)
- [Troubleshoot PMM HA Cluster issues](../troubleshoot/ha_issues.md)
