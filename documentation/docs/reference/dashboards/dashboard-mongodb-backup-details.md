# MongoDB Backup Details dashboard

This dashboard helps you monitor and manage your Percona Backup for MongoDB (PBM) environment directly within PMM. It consolidates backup configuration, agent health, and backup history into a single view, so you don't need to switch between tools.

The dashboard works with both replica sets and sharded clusters. Use the filters at the top to narrow down to specific environments, clusters, or replica sets.

![MongoDB Backup Details dashboard](../../images/BackupDetails_Dashboard.png)


This dashboard requires the MongoDB service to be added with the `--cluster` parameter. If panels show no data, see [Add MongoDB service to PMM](../../install-pmm/install-pmm-client/connect-database/mongodb.html#step-3-add-mongodb-service-to-pmm).

## Backup Configured

Shows whether PBM is configured with remote backup storage for your MongoDB environment. A green **YES** indicates backups are properly configured, while a red **NO** means backup storage is not set up.

If you see **NO**, verify that PBM has a remote storage location (S3, Azure Blob, or filesystem) defined. See [Configure backup storage](https://docs.percona.com/percona-backup-mongodb/install/backup-storage.html).

## PITR Status

Shows whether Point-in-Time Recovery (PITR) oplog streaming is enabled. A green **ON** confirms PITR is active, while a red **OFF** indicates this feature is not currently enabled.

PITR allows restoration to any point in time rather than just to specific backup snapshots. If your recovery requirements need granular restore points, ensure this is enabled in your PBM configuration.

## Backup Agents

Shows the total number of PBM agents currently monitored by PMM.

Use this to verify that all expected agents across your MongoDB environment are registered and reporting. If this count doesn't match your expected number of nodes, you may have agents that failed to register or are offline.

Backups will fail if agents are down or offline. If the count is lower than expected, check agent status immediately.

## Last Successful Backup

Shows the name of the most recent successful backup.

Use this to quickly confirm your current recovery point and verify that backups are running as expected. If this shows an old backup or **N/A**, investigate why recent backups may have failed.

## Backup Agent Summary

Shows the distribution of PBM agent statuses across your environment using a donut chart. Green represents agents with **OK** status, while red indicates agents that need attention (**CHECK**).

Use this for a quick health check of your backup infrastructure. A predominantly green chart means your agents are healthy. Red segments indicate agents requiring investigation—drill down using the Backup Agent Status panel to identify specific hosts.

## Backup Agent Status

Shows the current status of each individual PBM agent, displayed as a hexagon grid with one hexagon per host. Green means **OK**, red means **CHECK**.

Use this to pinpoint exactly which agents need investigation when the Backup Agent Summary shows problems. Hover over a hexagon to see the hostname and quickly identify which nodes require attention.

## Backup Agent Status Over Time

Shows how agent status has changed over the selected time range using a color-coded timeline. Green bars indicate **OK** status, red bars show **FAIL** or **DOWN** states.

Use this to identify patterns and troubleshoot intermittent issues. This historical view helps correlate backup failures with other events like network issues or maintenance windows, and verify that previously problematic agents have stabilized.

Arbiter nodes will appear with a **FAIL** status. This is expected, as PBM agent is not required on arbiter nodes and cannot run on them.

## Backup History

Shows a historical record of backup operations across your MongoDB infrastructure, including environment, cluster, backup name, size, and duration.

Use this to verify that scheduled backups are running successfully and to identify failed operations. The size and duration columns help spot anomalies—unusually small backups may indicate incomplete data capture, while long durations may signal performance issues.

The current status reporting may not capture the full range of error states available in PBM's native tools (including "stuck" or "incompatible" backups). This will be improved in an upcoming release.

## Backup Sizes

Shows the size of each backup in a bar chart format.

Use this to track storage requirements and plan capacity. Monitor for unusual changes—unexpectedly small backups could indicate incomplete data capture, while sudden increases might signal data growth that requires storage planning.

## Backup Duration

Shows how long each backup operation took to complete, displayed in seconds.

Use this to identify performance issues in your backup process. Backups taking unusually long may signal problems with your MongoDB setup, network bandwidth, or storage performance. Tracking duration trends also helps you plan maintenance windows more accurately.

