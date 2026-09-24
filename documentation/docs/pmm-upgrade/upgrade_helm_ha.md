# Upgrade PMM High Availability (HA) cluster using Helm

!!! warning "Technical Preview: Not production-ready"
    PMM HA Cluster is in **Technical Preview**. Make sure to test this upgrade procedure in non-production environments only.

Use this procedure to upgrade PMM Server in a [PMM HA Cluster](../install-pmm/HA-clustered.md) when a new PMM release is available. This applies to clusters deployed with the `percona/pmm-ha` Helm chart.

If you deployed PMM Server as a single instance on Kubernetes using the `percona/pmm` chart, see [Upgrade PMM Server using Helm](upgrade_helm.md) instead.

## How PMM HA Helm upgrades work

The upgrade restarts each of the three PMM Server pods one at a time, waiting for each to become ready before restarting the next.

### Downtime expectations

During the rollout, traffic flows only to whichever pod is the active leader. When the leader pod restarts, your PMM dashboards, alerts, and metric collection are briefly unavailable until a new leader takes over.

PMM is briefly unreachable when the leader pod restarts. Traffic only flows to the active leader. When that pod restarts, your PMM dashboards and alerts are briefly unreachable until another pod takes over. This typically takes around 5-10 seconds per pod and resolves on its own.

## Before you begin

Complete these steps before starting the upgrade:
{.power-number}

1. Check that all three PMM Server pods are running and ready:

    ```sh
    kubectl get pods -n <namespace> -l app.kubernetes.io/component=pmm-server
    ```

    !!! danger "Don't upgrade with a pod already down"
        The upgrade takes one pod offline at a time. If a pod is already down when you start, the upgrade brings the cluster to a single running pod, which is not enough to elect a leader. The result is a full outage (`503` from HAProxy) until you restore a second pod. Fix the unhealthy pod first. If you end up in this state, see [No quorum: cluster is unreachable after losing multiple replicas](../troubleshoot/ha_issues.md#no-quorum-cluster-is-unreachable-after-losing-multiple-replicas).

2. Back up your data. Downgrades are not supported, so a backup is the only way to recover if something goes wrong. Your monitoring data lives in shared database clusters, not on the PMM Server pods, and each needs to be backed up separately:

    - **PostgreSQL** is backed up automatically by default. Confirm a recent backup exists before you upgrade:

        ```sh
        kubectl get perconapgbackup -n <namespace>
        ```

    - **ClickHouse** and **VictoriaMetrics** have no automatic backup. Back them up manually before upgrading if you need to be able to restore your query analytics data and metrics (for example with [clickhouse-backup](https://github.com/Altinity/clickhouse-backup) and VictoriaMetrics' [`vmbackup`](https://docs.victoriametrics.com/vmbackup/)).

3. To reduce upgrade time, pull the new PMM Server image in advance on the nodes where your cluster runs:

    ```sh
    # Replace <version> with the version you're upgrading to
    docker pull percona/pmm-server:<version>
    ```

4. Make sure any settings you've customized (such as external access) are saved in your `values.yaml` file, not applied with `kubectl patch`.

    !!! danger "Settings applied with kubectl patch are lost on upgrade"
        `helm upgrade` rewrites the cluster's configuration from your Helm values. If you previously exposed PMM HA externally by running `kubectl patch` directly on the HAProxy Service instead of setting `haproxy.service.type` in your values, the upgrade silently resets it to the default, cutting off external access for your PMM Clients and dashboards. Keep all custom settings in `values.yaml`. See [Configure external access](../install-pmm/install-HA-clustered.md#configure-external-access).

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

3. Watch the rollout progress. Each pod restarts one at a time:

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

!!! caution alert alert-warning "Upgrade may fail with \"field is immutable\""
    In some chart versions, `helm upgrade` fails with `Job.batch "<release>-pmm-token-init" is invalid: spec.template: ... field is immutable`. See [Troubleshoot upgrade issues](../troubleshoot/upgrade_issues.md#pmm-ha-helm-upgrade-fails-with-field-is-immutable) for the fix.

## Confirm the cluster is healthy

After the upgrade, verify that a leader is active and the cluster is healthy. Open the **PMM HA** status badge in the side menu to see the current leader and cluster status at a glance:

![PMM HA leader badge showing the current leader and Healthy status](../images/pmm-ha-leader-badge.png)

You can also check through the Inventory page. See [Identify the leader node](../install-pmm/install-HA-clustered.md#identify-the-leader-node).

## Upgrade the underlying databases

`helm upgrade pmm-ha` upgrades PMM Server only. The databases that store your monitoring data (PostgreSQL, ClickHouse, and VictoriaMetrics) are managed separately and are not touched by this upgrade.

Only upgrade a database version when you have a specific reason, such as a supported-version deadline or a required feature. Otherwise, leave the versions as set in the chart. When you do need to upgrade one:

| Database | Downtime | Instructions |
|---|---|---|
| PostgreSQL | Yes for major versions (full cluster stop); no for minor versions | [Major version upgrade](https://docs.percona.com/percona-operator-for-postgresql/latest/update-db-major.html), [minor version upgrade](https://docs.percona.com/percona-operator-for-postgresql/latest/update-database.html) |
| ClickHouse | No, rolls one instance at a time | [Update the ClickHouse version](https://github.com/Altinity/clickhouse-operator/blob/master/docs/chi_update_clickhouse_version.md) |
| VictoriaMetrics | No | [Operator configuration](https://docs.victoriametrics.com/operator/configuration/) |

A PostgreSQL major version upgrade stops the whole cluster and cannot be reversed. Plan it as a separate maintenance window and take a full backup first.

!!! info "Keep the VictoriaMetrics version pin in place"
    Unlike the other databases, the VictoriaMetrics operator has a built-in default version it will use if the version is not explicitly set. Do not remove `victoriaMetrics.version` from your values, or an operator upgrade may silently change the VictoriaMetrics version.

## Roll back

`helm rollback` restores your previous Helm configuration but does **not** undo any data changes PMM Server made to the shared databases during the upgrade. If PMM already ran a data migration, you need to restore from your database backups to fully recover the previous state.
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
