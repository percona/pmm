# Support Diagnostics

!!! warning "Tech Preview"
    This feature is not production-ready. Use for testing and feedback only.

Support Diagnostics lets you collect diagnostic data from your database hosts and send it straight to your Percona support case in ServiceNow, without having to gather it manually or upload anything.

This capability is part of the [Management framework](index.md) integration.

## Before you start

- PMM Client 3.10.0 or later must be installed on the monitored host.
- You must have an open support case in Percona's ServiceNow.

## Run a diagnostic collection

1. Go to **Management > Support Diagnostics** in the left navigation.
2. Select the target host and your ServiceNow case number.
3. Click **Run**.

PMM collects the diagnostic data from the host and uploads the results directly to your support case. No files to download or upload manually.
