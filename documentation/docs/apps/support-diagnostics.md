# Collect and share database diagnostics

!!! warning "Tech Preview"
    This feature is not production-ready. Use it for testing and feedback only.

Support diagnostics runs targeted diagnostic scripts on your database hosts and sends the results directly to your Percona support case in ServiceNow, without SSH access or manual file uploads.

It is available under **Apps > Support diagnostics** in the sidebar, part of PMM's growing set of [database management apps](index.md).

## When to use Support diagnostics vs PMM Dump

Use **Support diagnostics** when Percona Support asks you to run diagnostics on your databases. It runs specific scripts on the host and ships the output to your case.

Use **[PMM Dump](../troubleshoot/pmm_dump.md)** when Percona Support asks for monitoring data from PMM itself, such as metrics or dashboards.

## Prerequisites

Before you start, make sure that:

- PMM Client 3.10.0 or later is installed on the monitored host.
- You have an open support case in Percona's ServiceNow.

## Connect PMM to ServiceNow

Support diagnostics uploads results to your support case through Percona's ServiceNow. PMM needs two credentials for this connection: a client token and a REST API key. You connect once per PMM instance, and every PMM user can then send diagnostics to your organization's support cases.

### Generate the credentials

Generate both credentials on the **Percona Token Management** page of the Percona Support portal:
{.power-number}

1. Sign in to the Percona Support portal and open **Percona Token Management**.
2. In **Token type**, select **Percona Connector**.
3. Click **Generate client token**. The **Your client token** dialog shows the **Client token** and the **REST API key**.
4. Copy both values and store them securely. The dialog shows them only once.

!!! hint "No access to the Percona Support portal?"
    [Open a Percona Support ticket](https://per.co.na/support) to request the credentials. Percona Support generates them for you and sends them over a secure channel. Anyone in your organization with portal access can also generate them and pass them to the PMM administrator.

### Enter the credentials in PMM

You must be a PMM administrator to connect PMM to ServiceNow. To enter the credentials:
{.power-number}

1. Go to **Configuration > Settings** and open the **ServiceNow connection** tab.
2. Paste the REST API key into **ServiceNow API key**.
3. Paste the client token into **Client token**.
4. Leave **ServiceNow endpoint (optional)** empty, unless Percona Support gave you a different endpoint.
5. Click **Verify and connect**. The tab shows **ServiceNow connected**.

To confirm that PMM can reach ServiceNow with the saved credentials, click **Test connection**. If the credentials have expired or were revoked, generate new ones and click **Renew credentials** to replace them.

## Run a diagnostic collection

To collect and send diagnostics to your support case:
{.power-number}

1. Go to **Apps > Support diagnostics** in the sidebar.
2. Click **New incident** and enter your ServiceNow case number.
3. Select the diagnostic scripts to run.
4. Click **Execute batch**. PMM runs the selected scripts on the target host.
5. Review the results, then select the ones to send.
6. Click **Send**. PMM uploads the selected results directly to your support case.

No files to download or upload manually.
