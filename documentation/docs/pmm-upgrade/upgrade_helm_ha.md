# Upgrade PMM High Availability (HA) cluster using Helm

!!! warning "Technical Preview: Not production-ready"
    PMM HA Cluster is in **Technical Preview**. Make sure to test this upgrade procedure in non-production environments only.

Use this procedure to upgrade PMM Server in a [PMM HA Cluster](../install-pmm/HA-clustered.md) when a new PMM release is available. This applies to clusters deployed with the `percona/pmm-ha` Helm chart.

If you deployed PMM Server as a single instance on Kubernetes using the `percona/pmm` chart, see [Upgrade PMM Server using Helm](upgrade_helm.md) instead.

## How PMM HA Helm upgrades work

The upgrade restarts each of the three PMM Server pods one at a time, waiting for each to become ready before restarting the next. 

Traffic flows only to the active leader pod. When that pod restarts, your PMM dashboards, alerts, and metric collection are briefly unavailable until a new leader takes over. This typically takes around 5-10 seconds per pod and resolves on its own.

## Before you begin

Complete these steps before upgrading to avoid data loss or extended downtime:
{.power-number}

1. Check that all three PMM Server pods are running and ready. The rollout takes one pod offline at a time, so starting with a pod already down reduces the cluster to a single pod, which cannot elect a leader and causes a full outage until you restore a second pod:

    ```sh
    kubectl get pods -n <namespace> -l app.kubernetes.io/component=pmm-server
    ```

2. Back up your data. Downgrades are not supported, so a backup is the only way to recover if something goes wrong. Your monitoring data lives in shared database clusters, not on the PMM Server pods, and each needs to be backed up separately:

    - **PostgreSQL** is backed up automatically by default. Confirm a recent backup exists before you upgrade:

        ```sh
        kubectl get perconapgbackup -n <namespace>
        ```

    - **ClickHouse** and **VictoriaMetrics** have no automatic backup. Back them up manually before upgrading if you need to be able to restore your query analytics data and metrics (for example with [clickhouse-backup](https://github.com/Altinity/clickhouse-backup) and VictoriaMetrics' [`vmbackup`](https://docs.victoriametrics.com/vmbackup/)).

3. Make sure any custom settings are in your `values.yaml` file, not applied with `kubectl patch`. Helm rewrites the cluster configuration from your values on every upgrade, so any settings applied directly to Kubernetes resources (such as exposing PMM externally via `kubectl patch` on the HAProxy Service) are silently reset.

4. To reduce upgrade time, pull the new PMM Server image in advance on the nodes where your cluster runs:

    ```sh
    # Replace <version> with the version you're upgrading to
    docker pull percona/pmm-server:<version>
    ```

## Upgrade

Follow these steps to upgrade your PMM HA Cluster:
{.power-number}

1. Update the Helm repository:

    ```sh
    helm repo update percona
    ```

2. Run the upgrade, replacing `<version>` with the target PMM version:

    ```sh
    helm upgrade pmm-ha percona/pmm-ha \
      --namespace pmm \
      --reuse-values \
      --set image.tag=<version>
    ```

    `--reuse-values` keeps your existing configuration and only changes the image version. If you manage your settings in a `values.yaml` file, pass `-f values.yaml` with the updated `image.tag` instead.

    In some chart versions, this step can fail with `Job.batch "<release>-pmm-token-init" is invalid: spec.template: ... field is immutable`. See [Troubleshoot upgrade issues](../troubleshoot/upgrade_issues.md#pmm-ha-helm-upgrade-fails-with-field-is-immutable) for the fix.

3. Watch the rollout progress. Expect a [brief interruption when the leader pod restarts](#how-pmm-ha-helm-upgrades-work):

    ```sh
    kubectl rollout status statefulset/pmm-ha -n pmm
    ```

4. After the rollout completes, confirm all three pods are running the new version:

    ```sh
    kubectl get pods -l app.kubernetes.io/name=pmm -n pmm -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[0].image}{"\n"}{end}'
    ```

5. Confirm PMM Server is reachable and returning the new version:

    ```sh
    curl -k https://<pmm-ha-haproxy-endpoint>/v1/server/version
    ```

6. Check the logs across all pods for errors:

    ```sh
    kubectl logs -l app.kubernetes.io/name=pmm -n pmm --tail=100
    ```

## Confirm the cluster is healthy

After the upgrade, verify that a leader is active and the cluster is healthy. Open the **PMM HA** status badge in the side menu to see the current leader and cluster status at a glance:

![PMM HA leader badge showing the current leader and Healthy status](../images/pmm-ha-leader-badge.png)

You can also check through the Inventory page. See [Identify the leader node](../install-pmm/install-HA-clustered.md#identify-the-leader-node).

## Upgrade the underlying databases

Running `helm upgrade pmm-ha` upgrades PMM Server only, not the databases underneath it. PostgreSQL, ClickHouse, and VictoriaMetrics are each managed by their own operators and versioned independently. This upgrade does not change their versions.

Only upgrade a database version when you have a specific reason, such as a supported-version deadline or a required feature. Otherwise, leave the versions as set in the chart. When you do need to upgrade one:

| Database | Downtime | Instructions |
|---|---|---|
| PostgreSQL | Yes for major versions (full cluster stop); no for minor versions | [Major version upgrade](https://docs.percona.com/percona-operator-for-postgresql/latest/update-db-major.html), [minor version upgrade](https://docs.percona.com/percona-operator-for-postgresql/latest/update-database.html) |
| ClickHouse | No, rolls one instance at a time | [Update the ClickHouse version](https://github.com/Altinity/clickhouse-operator/blob/master/docs/chi_update_clickhouse_version.md) |
| VictoriaMetrics | No | [Operator configuration](https://docs.victoriametrics.com/operator/configuration/) |

Keep these in mind before upgrading a database:

- **PostgreSQL major version upgrades** stop the whole cluster and cannot be reversed. Plan them as a separate maintenance window and take a full backup first.
- **VictoriaMetrics:** Do not remove `victoriaMetrics.version` from your values. Unlike the other databases, the VictoriaMetrics operator falls back to a built-in default if no version is set, so an operator upgrade can silently change the version.

## Roll back

`helm rollback` restores your previous Helm configuration but does **not** undo any data changes PMM Server made to the shared databases during the upgrade. 

If PMM already ran a data migration, you need to restore from your database backups to fully recover the previous state:
{.power-number}

1. List available revisions:

    ```sh
    helm history pmm-ha -n pmm
    ```

2. Roll back to the revision before the upgrade:

    ```sh
    helm rollback pmm-ha <revision-number> -n pmm
    ```

!!! seealso alert alert-info "See also"
    - [Understand PMM High Availability Cluster](../install-pmm/HA-clustered.md)
    - [Install PMM HA Cluster](../install-pmm/install-HA-clustered.md)
    - [Troubleshoot PMM HA Cluster issues](../troubleshoot/ha_issues.md)
    - [Upgrade PMM Server using Helm](upgrade_helm.md) (single-instance deployments)
