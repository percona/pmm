---
title: Logs
slug: pmm-server-logs
categorySlug: pmm-server-maintenance
parentDocSlug: pmm-server-troubleshooting
---
Download the logs and components configuration to troubleshoot any issues with the PMM Server.

## Accessing logs

PMM Server offers three ways to access and download logs:

1. Through direct URL, by visiting `https://<address-of-your-pmm-server>/logs.zip`.
2. By calling the Logs endpoint. This method enables you to customize the log content using the `line-count` parameter: For example:

   - Default 50,000 lines: `https://<pmm-server>/logs.zip`
   - Custom number of lines: `https://<pmm-server>/logs.zip?line-count=10000`
   - Unlimited, full log: `https://<pmm-server>/logs.zip?line-count=-1`
3. Through the UI, by selecting the **Help > PMM Logs** option from the main menu.
  If you need to share logs with Percona Support via an SFTP server, you can also use the **PMM Dump** option from the Help menu to generate a compressed tarball file with an export of your PMM metrics and QAN data.
  For more information, see [Export PMM data with PMM Dump](https://docs.percona.com/percona-monitoring-and-management/how-to/PMM_dump.html) topic in the product documentation.

## Log structure

The downloaded logs package contains the following structure:

[block:code]
{
  "codes": [
    {
      "code": "# tree\n├── clickhouse-server.err.log\n├── clickhouse-server.log\n├── clickhouse-server.startup.log\n├── client\n│   ├── list.txt\n│   ├── pmm-admin-version.txt\n│   ├── pmm-agent-config.yaml\n│   ├── pmm-agent-version.txt\n│   └── status.json\n├── cron.log\n├── dashboard-upgrade.log\n├── grafana.log\n├── installed.json\n├── nginx.conf\n├── nginx.log\n├── nginx.startup.log\n├── pmm-agent.log\n├── pmm-agent.yaml\n├── pmm-managed.log\n├── pmm-ssl.conf\n├── pmm-update-perform-init.log\n├── pmm-version.txt\n├── pmm.conf\n├── pmm.ini\n├── postgresql.log\n├── postgresql.startup.log\n├── prometheus.base.yml\n├── prometheus.log\n├── qan-api2.ini\n├── qan-api2.log\n├── supervisorctl_status.log\n├── supervisord.conf\n├── supervisord.log\n├── systemctl_status.log\n├── victoriametrics-promscrape.yml\n├── victoriametrics.ini\n├── victoriametrics.log\n├── victoriametrics_targets.json\n├── vmalert.ini\n└── vmalert.log",
      "language": "text"
    }
  ]
}
[/block]

[block:callout]
{
  "type": "info",
  "title": "PMM Server Version",
  "body": "PMM Server also dumps its version info to a special file `installed.json`. \n\n```shell\n% cat installed.json | jq\n{\n  \"version\": \"3.0.0\",\n  \"full_version\": \"3.0.0-1.2412081130.6914083.el9\",\n  \"build_time\": \"2024-12-08T11:30:45Z\",\n  \"repo\": \"local\"\n}\n```"
}
[/block]
