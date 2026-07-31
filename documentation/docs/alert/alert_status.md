# View alert status by node or service

Use the **Alert Status** page to see all active alerts for a node or service in one place, check their states, and silence them without switching views.

This page is only available when Percona Alerting is enabled. To check, go to **Configuration > Settings > Advanced settings**.


## Check alerts for a specific node

To see all alerts affecting a single node:
{.power-number}

1. Go to **Alerts > Status**.
2. Toggle **Group by node** in the toolbar.
3. Find your node and expand it to see its individual alerts and their states.

## Filter alerts by state

Use the **State** dropdown in the toolbar to show only alerts in a specific state: **Normal**, **Pending**, **Firing**, **Recovering**, **No Data**, **Error** or **Silenced**.

## Check alerts for a specific service

To filter the alert list by service:
{.power-number}

1. Go to **Alerts > Status**.
2. Click **Show/Hide filters** in the toolbar to reveal the column filters.
3. In the **Service** column filter, enter the service name.

The table updates to show only alerts associated with that service.

## Get details on an alert

Click any alert row to open the details pane. From here you can:

- **See what triggered the alert**: the **Details** tab shows the summary, description, state and duration, node, service, severity, triggered at timestamp, and the MetricsQL expression. Check **Rule configuration** to see evaluation settings, template name, folder, and rule health.
- **Debug custom templates or verify label values**: switch to **Raw data** to inspect the full label set and JSON payload.
- **Move between alerts**: use the arrow buttons in the pane header to go to the next or previous alert without closing the pane.

## Silence an alert

You need **Editor** role or higher to silence alerts.
{.power-number}

1. Click the actions menu on the alert row.
2. Click **Silence**. PMM opens the **Silences** page with the alert labels pre-filled.
3. Set the duration and confirm.

Silenced alerts stay visible in the table with a **Silenced** badge. To unsilence, open the actions menu and click **Unsilence**.

## Make custom alerts appear in this view

Built-in PMM alert templates automatically include the labels that this page uses to group alerts by node and service. 

If your custom templates do not show up correctly, add these labels to the template definition:

- `node_name`: identifies the monitored node.
- `service_name`: identifies the monitored service.

For instructions, see [Alert rules and alert templates](../alert/templates.md).