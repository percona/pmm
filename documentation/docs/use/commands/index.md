# About PMM commands

PMM provides two command-line tools for managing your monitoring setup from the terminal. 

Use these tools to add databases, configure agents, check status, and troubleshoot issues without leaving the command line.

You can also perform most of these tasks through the [PMM web interface](https://docs.percona.com/percona-monitoring-and-management/get-started/interface.html) or the [PMM API](../../api/index.md).

## Command-line tools

`pmm-admin`: The primary CLI tool for administering PMM. Use it to add and remove database services, check connection status, list monitored services, modify agent configurations, create diagnostic archives, and annotate dashboards. Communicates directly with PMM Server.

    `pmm-admin` is installed automatically as part of the [PMM Client](../../../install-pmm/install-pmm-client/index.md) package.

    See [`pmm-admin` reference](pmm-admin/pmm-admin.md) for syntax, common flags, and links to all subcommands.

`pmm-agent`: The daemon process that runs on each monitored host. It manages exporters and agents locally, coordinating data collection and communication between PMM Client and PMM Server. 

    You typically don't interact with pmm-agent directly, `pmm-admin` communicates with PMM Server, which then sends commands to `pmm-agent`. See [Coordinate monitoring agents with pmm-agent](pmm-agent.md) for configuration options and startup flags.

## Next steps
 
- [Get started with `pmm-admin`](../commands/pmm-admin/pmm-admin.md)
- [Add databases to monitoring](pmm-admin/add.md)
- [Manage inventory and modify agents](pmm-admin/inventory.md)
- [Configure, register, and remove services](pmm-admin/config.md)
- [Check status and troubleshoot](pmm-admin/status.md)
