# Upgrade PMM Server HA Cluster on Kubernetes

Upgrade a [PMM Server High Availability (HA) Cluster](../install-pmm/HA-clustered.md) deployed with the [`percona/pmm-ha` chart](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm-ha) when you want to move to a new PMM release.

!!! caution "Requires PMM 3.10.0 and `percona/pmm-ha` chart 1.8.0 or later"
    In-place Helm upgrades for HA clusters are not supported on earlier versions. If you're running an earlier version, you cannot use this procedure.

## How PMM HA Helm upgrades work

The upgrade restarts each of the three PMM Server pods one at a time, waiting for each to become ready before restarting the next. HAProxy, PostgreSQL, VictoriaMetrics, ClickHouse, and the operators are not restarted and remain running throughout.

If this PMM release updated operator versions, make sure to first [upgrade the operators](#upgrade-the-operators). 

Traffic flows only to the active leader pod. When that pod restarts, your PMM dashboards, alerts, and metric collection are briefly unavailable until a new leader takes over. This typically takes around 5-10 seconds per pod and resolves on its own.

## Prerequisites

To avoid losing data or a long outage, follow these steps before you upgrade:
{.power-number}

1. Check that your cluster runs Kubernetes 1.32 or later and that you use Helm 3.17.0 or later. The PMM 3.10.0 chart doesn't install on earlier versions:

    ```sh
    kubectl version
    helm version
    ```

2. Check that all three PMM Server pods are running and ready. The upgrade restarts the pods one at a time, so if one is already down, PMM becomes unreachable while the next one restarts:

    ```sh
    kubectl get pods -n <namespace> -l app.kubernetes.io/component=pmm-server
    ```

3. [Back up your PMM HA Cluster](../install-pmm/backup-restore-HA-clustered.md#back-up-pmm-ha-cluster) before upgrading. Downgrades are not supported, so a backup is your only recovery option. If the upgrade fails, [restore the backup](../install-pmm/backup-restore-HA-clustered.md#restore-pmm-ha-cluster).

4. Check whether this PMM release updated operator versions. Compare the operator versions in the [supported platforms table](../install-pmm/HA-clustered.md#check-if-your-platform-is-supported) for your current and target chart versions. If they differ, complete [Upgrade the operators](#upgrade-the-operators) before running `helm upgrade pmm-ha`.

5. Keep all custom settings in your `values.yaml` file. Settings applied with `kubectl patch` are silently reset on every upgrade.
## Upgrade the operators

Each PMM HA database (VictoriaMetrics, ClickHouse, and PostgreSQL) is managed by a Kubernetes operator that handles its lifecycle: scaling, failover, and version upgrades. These operators run as a separate Helm release (`pmm-operators`) installed from the `pmm-ha-dependencies` chart — `helm upgrade pmm-ha` does not touch them. Run these steps only if the operator versions in the [supported platforms table](../install-pmm/HA-clustered.md#check-if-your-platform-is-supported) differ between your current and target chart versions.

To upgrade the operators:
{.power-number}

1. Find the release name of your operators installation, usually `pmm-operators`:

    ```sh
    helm list --namespace <namespace>
    ```

2. Upgrade it:

    ```sh
    helm upgrade pmm-operators percona/pmm-ha-dependencies --namespace <namespace>
    ```

3. Wait for the new operator pods to roll out:

    ```sh
    kubectl rollout status deployment -l app.kubernetes.io/name=pg-operator -n <namespace> --timeout=300s
    kubectl rollout status deployment -l app.kubernetes.io/name=altinity-clickhouse-operator -n <namespace> --timeout=300s
    kubectl rollout status deployment -l app.kubernetes.io/name=victoria-metrics-operator -n <namespace> --timeout=300s
    ```

Upgrading the operators does not restart or change the database clusters they manage. The databases keep running under the new operator version automatically.

!!! caution "PostgreSQL operator major version upgrades"
    If the `pmm-ha-dependencies` chart bumps the Percona Operator for PostgreSQL to a new major version (for example, 2.x to 3.0.0), review the [operator upgrade notes](https://docs.percona.com/percona-operator-for-postgresql/latest/update.html) before upgrading. Major operator versions can include CRD changes that require manual steps.

## Upgrade PMM HA Cluster

Once you've completed the steps in [Before you begin](#before-you-begin), run the upgrade:
{.power-number}

1. Update the Helm repository:

    ```sh
    helm repo update percona
    ```

2. Find the chart version that ships the target PMM version. The **APP VERSION** column shows the PMM version:

    ```sh
    helm search repo percona/pmm-ha --versions
    ```

3. Run the upgrade, replacing `<chart-version>` with the chart version from the previous step and `<version>` with the target PMM version. Without `--version`, Helm installs the newest chart in the repository. Use `--reset-then-reuse-values` to keep your existing configuration and pick up the new chart's defaults, or `-f values.yaml` if you manage settings in a values file:

    ```sh
    helm upgrade pmm-ha percona/pmm-ha \
      --namespace pmm \
      --version <chart-version> \
      --reset-then-reuse-values \
      --set image.tag=<version>
    ```

    If this fails with `Job.batch "<release>-pmm-token-init" is invalid: spec.template: ... field is immutable`, see [Troubleshoot upgrade issues](../troubleshoot/upgrade_issues.md#pmm-ha-helm-upgrade-fails-with-field-is-immutable).

4. Track the rollout progress. Expect a brief interruption when the leader pod restarts:

    ```sh
    kubectl rollout status statefulset/pmm-ha -n pmm
    ```

5. Confirm all three pods are running the new version:

    ```sh
    kubectl get pods -l app.kubernetes.io/name=pmm -n pmm -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[0].image}{"\n"}{end}'
    ```

6. Confirm PMM Server is reachable and returning the new version:

    ```sh
    curl -k https://<pmm-ha-haproxy-endpoint>/v1/server/version
    ```

7. Check the PMM Server logs on each pod for errors. PMM Server writes them to files under `/srv/logs/`, so `kubectl logs` doesn't show them:

    ```sh
    for p in pmm-ha-0 pmm-ha-1 pmm-ha-2; do
      echo "--- $p"
      kubectl exec -n pmm "$p" -c pmm-ha -- tail -50 /srv/logs/pmm-managed.log
    done
    ```

### Built-in alert rules during the upgrade

The nodes are upgraded one at a time, so for a short while they run different PMM versions. If the new version changes a [built-in alert rule](../alert/built_in_alerts.md), the rule switches back and forth between its old and new settings until every node runs the new version. You don't need to do anything. The switching stops on its own.

## Confirm the cluster is healthy

After the upgrade, verify that the cluster is healthy and that a leader is active:
{.power-number}

1. Check that the **PMM HA** badge in the side menu shows **Healthy**.

    ![PMM HA badge showing Healthy status](../images/HA-Status.png)

2. Go to **PMM HA > Identify Nodes** and check that one PMM Server node has the **Leader** badge next to its name. For details, see [Check HA roles in Inventory](../install-pmm/install-HA-clustered.md#check-ha-roles-in-inventory).

3. Go to **PMM HA > Overview** and check that all component panels in the **Overview** section are green on the [PMM HA Health Overview](../reference/dashboards/dashboard-ha-health-overview.md) dashboard.

If you cannot log in after upgrading, see [Cannot log in with the admin password](../troubleshoot/ha_issues.md#cannot-log-in-with-the-admin-password).

## Upgrade the underlying databases

Upgrading the operators does not change the database versions. Only upgrade a database when you have a specific reason, such as a version deadline or a required feature.

If you need to upgrade one:

| Database | Downtime | Instructions | Planning notes |
|---|---|---|---|
| PostgreSQL | Yes for major versions (full cluster stop); no for minor versions | [Major version upgrade](https://docs.percona.com/percona-operator-for-postgresql/latest/update-db-major.html), [minor version upgrade](https://docs.percona.com/percona-operator-for-postgresql/latest/update-database.html) | Cannot be reversed. Plan as a separate maintenance window and take a full backup first. |
| ClickHouse | No, rolls one instance at a time | [Update the ClickHouse version](https://github.com/Altinity/clickhouse-operator/blob/master/docs/chi_update_clickhouse_version.md) | |
| VictoriaMetrics | No | [Operator configuration](https://docs.victoriametrics.com/operator/configuration/) | Do not remove `victoriaMetrics.version` from your values. The operator falls back to a built-in default if not set, so an operator upgrade can silently change the version. |

## Roll back to a previous version

If the upgrade causes issues, you can restore your previous Helm configuration. Keep in mind that this only restores the PMM Server configuration, not any database changes PMM made during the upgrade. 

To fully return to the pre-upgrade state, roll back the Helm release first, then [restore the backup](../install-pmm/backup-restore-HA-clustered.md#restore-pmm-ha-cluster) you took in [Before you begin](#before-you-begin). A backup can only be restored into the same PMM version it was taken from.

To roll back the Helm release to the previous version:
{.power-number}

1. List available revisions:

    ```sh
    helm history pmm-ha -n pmm
    ```

2. Roll back to the revision before the upgrade:

    ```sh
    helm rollback pmm-ha <revision-number> -n pmm
    ```

## Related topics

- [Understand PMM High Availability Cluster](../install-pmm/HA-clustered.md)
- [Install PMM HA Cluster](../install-pmm/install-HA-clustered.md)
- [Troubleshoot PMM HA Cluster issues](../troubleshoot/ha_issues.md)
