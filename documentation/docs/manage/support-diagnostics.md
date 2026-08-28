# Support Diagnostics

!!! warning "Tech Preview"
    This feature is not production-ready. Use for testing and feedback only.

Support Diagnostics runs targeted diagnostic scripts directly on your database hosts and sends the results straight to your Percona support case in ServiceNow, without connecting to any server or uploading files yourself.

This capability is part of the [Management framework](index.md) integration.

## Support Diagnostics vs PMM Dump

Support Diagnostics runs specific diagnostic scripts on your database hosts to investigate a particular issue, and ships the output to your support case. 

[PMM Dump](../../get-help.md) exports PMM's own monitoring data (metrics and dashboards) compressed for Percona to analyze. 
If Percona Support asks you for monitoring data from PMM, use PMM Dump. If they ask you to run diagnostics on your databases, use Support Diagnostics.

## Before you start

- PMM Client 3.10.0 or later must be installed on the monitored host.
- You must have an open support case in Percona's ServiceNow.

## Run a diagnostic collection

1. Go to **Management > Support Diagnostics** in the left navigation.
2. Select the target host and your ServiceNow case number.
3. Click **Run**.

PMM collects the diagnostic data from the host and uploads the results directly to your support case. No files to download or upload manually.
