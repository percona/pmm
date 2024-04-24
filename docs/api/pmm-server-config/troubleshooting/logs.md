---
title: Logs
slug: "logs"
category: 626badcabbc59c02acc1a540
---

Sometimes users need to troubleshoot an issue. PMM Server offers an ability to download the logs as well as configuration of its components. 

You can download the logs either by calling this endpoint or by visiting a dedicated URL (ex: https://pmmdemo.percona.com/logs.zip) or via the **Settings UI** as explained in the [Troubleshooting](https://docs.percona.com/percona-monitoring-and-management/how-to/troubleshoot.html#client-server-connections) section of our docs.

The structure of the logs is as follows:
[block:code]
{
  "codes": [
    {
      "code": "# tree\n├── clickhouse-server.err.log\n├── clickhouse-server.log\n├── clickhouse-server.startup.log\n├── client\n│   ├── list.txt\n│   ├── pmm-admin-version.txt\n│   ├── pmm-agent-config.yaml\n│   ├── pmm-agent-version.txt\n│   └── status.json\n├── cron.log\n├── dashboard-upgrade.log\n├── grafana.log\n├── installed.json\n├── nginx.conf\n├── nginx.log\n├── nginx.startup.log\n├── pmm-agent.log\n├── pmm-agent.yaml\n├── pmm-managed.log\n├── pmm-ssl.conf\n├── pmm-update-perform-init.log\n├── pmm-update-perform.log\n├── pmm-version.txt\n├── pmm.conf\n├── pmm.ini\n├── postgresql.log\n├── postgresql.startup.log\n├── prometheus.base.yml\n├── prometheus.log\n├── qan-api2.ini\n├── qan-api2.log\n├── supervisorctl_status.log\n├── supervisord.conf\n├── supervisord.log\n├── systemctl_status.log\n├── victoriametrics-promscrape.yml\n├── victoriametrics.ini\n├── victoriametrics.log\n├── victoriametrics_targets.json\n├── vmalert.ini\n└── vmalert.log",
      "language": "text"
    }
  ]
}
[/block]

[block:callout]
{
  "type": "info",
  "title": "PMM Server Version",
  "body": "PMM Server also dumps its version info to a special file `installed.json`. \n\n```shell\n% cat installed.json | jq\n{\n  \"version\": \"2.26.0\",\n  \"full_version\": \"2.26.0-17.2202021129.6914083.el7\",\n  \"build_time\": \"2022-02-02T11:30:45Z\",\n  \"repo\": \"local\"\n}\n```"
}
[/block]
