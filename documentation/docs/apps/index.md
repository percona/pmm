# Database management apps   

!!! warning "Tech Preview"
    These apps are not production-ready. Use them for testing and feedback only.

PMM Apps extend PMM with database management capabilities. They enable you to run and track operations on your database hosts directly from PMM, without SSH access.

## Available apps

- [MySQL Backups](mysql-backup.md): run and schedule MySQL backups using XtraBackup, Mydumper, or Binlog, and restore from them.
- [Support diagnostics](support-diagnostics.md): collect diagnostic data from your hosts and send it directly to your Percona support case in ServiceNow.

More operations and database types will be added in future releases.

The **Apps** menu follows PMM Extensions: it lists each app that is enabled there, so an app you disable in PMM Extensions leaves the menu. If PMM can't reach PMM Extensions when it loads, the menu still shows **Support diagnostics** and **MySQL Backups**.

## Get started

To enable the available apps and the **Apps** menu, [deploy PMM Extensions](deploy-pmm-extensions.md).