# Collect and share database diagnostics

!!! warning "Tech Preview"
    This feature is not production-ready. Use for testing and feedback only.

Support Diagnostics runs targeted diagnostic scripts on your database hosts and sends the results directly to your Percona support case in ServiceNow, without SSH access or manual file uploads.

It is available under **Apps > Support Diagnostics** in the sidebar, part of PMM's growing set of [database management apps](index.md).

## When to use Support Diagnostics vs PMM Dump

Use **Support Diagnostics** when Percona Support asks you to run diagnostics on your databases. It runs specific scripts on the host and ships the output to your case.

Use **[PMM Dump](../../docs/troubleshoot/pmm_dump.md)** when Percona Support asks for monitoring data from PMM itself, such as metrics or dashboards.

## Before you start

- PMM Client 3.10.0 or later must be installed on the monitored host.
- You must have an open support case in Percona's ServiceNow.

## Run a diagnostic collection

To collect and send diagnostics to your support case:
{.power-number}

1. Go to **Apps > Support Diagnostics** in the sidebar.
2. Select the target host and your ServiceNow case number.
3. Click **Run**.

PMM collects the diagnostic data from the host and uploads the results directly to your support case. No files to download or upload manually.
