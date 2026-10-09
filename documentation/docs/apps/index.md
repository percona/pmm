# Database management apps   

!!! warning "Tech Preview"
    These apps are not production-ready. Use for testing and feedback only.

PMM Apps extend PMM with database management capabilities, allowing you to run and track operations on your database hosts directly from PMM, without SSH access. 

## Available apps

- [MySQL Backups](mysql-backup.md): run and schedule MySQL backups using XtraBackup, Mydumper, or Binlog, and restore from them.
- [Support Diagnostics](support-diagnostics.md): collect diagnostic data from your hosts and send it directly to your Percona support case in ServiceNow.

More operations and database types will be added in future releases.

## Get Started 

To enable the available apps and the **Apps** menu, [deploy PMM Extensions](deploy-pmm-extensions.md) to enable the **Apps** menu.