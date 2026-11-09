---
title: Logs
slug: pmm-server-logs
category:
  uri: pmm-server-maintenance
parent:
  uri: pmm-server-troubleshooting
---
Download the logs and components configuration to troubleshoot any issues with the PMM Server.

## Accessing logs

PMM Server offers three ways to access and download logs:

1. Through direct URL, by visiting `https://<address-of-your-pmm-server>/v1/server/logs.zip`.
2. By calling the Logs endpoint. This method enables you to customize the log content using the `line-count` parameter: For example:

   - Default 50,000 lines: `https://<pmm-server>/v1/server/logs.zip`
   - Custom number of lines: `https://<pmm-server>/v1/server/logs.zip?line-count=10000`
   - Unlimited, full log: `https://<pmm-server>/v1/server/logs.zip?line-count=-1`
3. Through the UI, by selecting the **Help > PMM Logs** option from the main menu.
  If you need to share logs with Percona Support via an SFTP server, you can also use the **PMM Dump** option from the Help menu to generate a compressed tarball file with an export of your PMM metrics and QAN data.
  For more information, see [Export PMM data with PMM Dump](https://docs.percona.com/percona-monitoring-and-management/3/troubleshoot/pmm_dump.html) topic in the product documentation.

## Log structure

The downloaded logs package contains the following structure:


```text
# tree
├── clickhouse-server.err.log
├── clickhouse-server.log
├── clickhouse-server.startup.log
├── client
│   ├── list.txt
│   ├── pmm-admin-version.txt
│   ├── pmm-agent-config.yaml
│   ├── pmm-agent-version.txt
│   └── status.json
├── cron.log
├── dashboard-upgrade.log
├── grafana.log
├── installed.json
├── nginx.conf
├── nginx.log
├── nginx.startup.log
├── pmm-agent.log
├── pmm-agent.yaml
├── pmm-managed.log
├── pmm-ssl.conf
├── pmm-init.log
├── pmm-version.txt
├── pmm.conf
├── pmm.ini
├── postgresql.log
├── postgresql.startup.log
├── prometheus.base.yml
├── prometheus.log
├── qan-api2.ini
├── qan-api2.log
├── supervisorctl_status.log
├── supervisord.conf
├── supervisord.log
├── systemctl_status.log
├── victoriametrics-promscrape.yml
├── victoriametrics.ini
├── victoriametrics.log
├── victoriametrics_targets.json
├── vmalert.ini
└── vmalert.log
```


> 📘 PMM Server Version
>
> PMM Server also dumps its version info to a special file `installed.json`.
```shell
% cat installed.json | jq
{
  "version": "3.0.0",
  "full_version": "3.0.0-1.2412081130.6914083.el9",
  "build_time": "2024-12-08T11:30:45Z",
  "repo": "local"
}
```
