# MongoDB PBM Details dashboard

The MongoDB PBM Details dashboard offers an integrated view of your Percona Backup for MongoDB (PBM) environment directly within Percona Monitoring and Management (PMM).

The dashboard consolidates key information—such as backup configuration, status, performance metrics, and agent health—into a single, easy-to-use interface.

By accessing PBM insights directly from PMM, you can efficiently monitor and manage your MongoDB backups without switching between tools.


![PBM dashboard](../../images/PBM_Dashboard.png)

## Backup Configured

Shows whether backups are properly configured for your MongoDB environment. A green "Yes" indicates that PBM is properly set up and functioning, while a **No** in red signals that backups are not configured. 

## PITR Enabled

Displays whether Point-in-Time Recovery (PITR) is enabled for your MongoDB environment. A green **Yes** confirms PITR is active, while a **No** in red indicates this feature is not currently enabled.

PITR allows for more granular recovery options, enabling restoration to any point in time rather than just to specific backup points. 

## Agent Status

Monitors the operational status of each PBM agent connected to your MongoDB cluster nodes using a color-coded timeline visualization. 

The panel shows each replica set node (e.g., `rs101:27017`, `rs102:27017`, `rs103:27017`) with an **Ok** status in green for functioning agents. This helps you quickly identify any problematic agents that may be affecting backup operations.

## Size Bytes

Displays the size of your MongoDB backups in a bar chart format. 

The panel shows the exact size of each backup (e.g., 10.7 MB), helping you track storage requirements and identify any unusual changes in backup size that might indicate problems with your data or backup process.

## Duration

Shows how long each backup operation takes to complete, displayed in seconds. 

This performance metric helps you plan maintenance windows and identify any backups that are taking longer than expected, which could indicate performance issues in your MongoDB environment.

## Backup History

Provides a tabular view of recent backup operations with columns for **Name** (timestamp of the backup) and **Status** (Done, Error, etc.). 

This historical record helps you verify that scheduled backups are running successfully and lets you quickly identify any failed backup operations that may require attention.

The current status reporting in this panel may not yet capture the full range of error states available in PBM's native tools (including "stuck" or "incompatible" backups). This will be improved with an upcoming release to provide a more complete picture of your backup status.

## Last Successful Backup

Shows details of the most recent successful backup operation, including its timestamp. This gives you confidence in your recovery capabilities by confirming when your last good backup was taken, ensuring you know your current recovery point objective (RPO) status.

This dashboard works with both replica sets and sharded clusters, providing unified visibility into your MongoDB backup infrastructure. 

To use it effectively, select the appropriate environments, clusters, and replica sets using the filters at the top of the dashboard.