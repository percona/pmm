# Adding a MySQL or PostgreSQL Remote DB instance to PMM

There is a quick method for users to add DBaaS instances to PMM without having to hook into the Cloud Provider’s API, and with no need to have PMM Client installed or any exporters running on the monitored node. The drawback of this approach is that you will not have visibility of host-level metrics (CPU, memory, and disk activity will not be captured nor displayed in PMM).

!!! note
    There is an alternative and more complex approach available for MySQL Server, which involves API-aware addition of an Amazon RDS / Aurora DB instance.

Both methods can be accessed in the Metrics Monitor navigation menu by selecting the *PMM Add Instance* item in a PMM Dropdown group:

![](_images/metrics-monitor.menu.pmm1.png)

Two database servers are currently supported by this method: PostgreSQL and MySQL.

![](_images/metrics-monitor.add-rds-or-remote-instance.png)

## Adding a Remote PostgreSQL Instance

To add a remote PostgreSQL DB instance, you will need to fill in three fields: Hostname, Username, and Password, and optionally override the default Port and Name fields:

![](_images/metrics-monitor.add-remote-postgres-instance.png)

## Adding a Remote MySQL Instance

To add a remote MySQL DB instance, you will need to fill in three fields: Hostname, Username, and Password, and optionally override the default Port and Name fields:

![](_images/metrics-monitor.add-remote-mysql-instance.png)

## Viewing Remote MySQL and PostgreSQL Instances

Amazon RDS and remote instances can be seen in the RDS and
Remote Instances list, which can be accessed in the Metrics Monitor navigation
menu by selecting the *PMM RDS and Remote Instances* item from the
PMM Dropdown menu:

![](_images/metrics-monitor.menu.pmm2.png)

Remote ones have remote keyword as a Region:

![](_images/metrics-monitor.add-rds-or-remote-instance1.png)
