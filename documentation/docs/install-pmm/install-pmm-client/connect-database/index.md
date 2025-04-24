# Connect databases to PMM
Percona Monitoring and Management (PMM) supports monitoring for MySQL/MariaDB, PostgreSQL, MongoDB, and various cloud database services. 

## Supported database technologies

The table below shows monitoring capabilities for each supported database type:

| Database type                                | Local monitoring | Remote monitoring | Query Analytics (QAN) | Performance schema | Backup integration |
|----------------------------------------------|------------------|-------------------|------------------|---------------------|---------------------|
| [MySQL/MariaDB](../connect-database/mysql/mysql.md)     | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> |
| [PostgreSQL](../connect-database/postgresql.md)          | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [MongoDB](../connect-database/mongodb.md)                | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> |
| [AWS RDS/Aurora](../connect-database/aws.md)             | <span style="color:red">✘</span>  | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [Azure Database](../connect-database/azure.md)           | <span style="color:red">✘</span>  | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [Google Cloud SQL](../connect-database/google.md)        | <span style="color:red">✘</span>  | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [ProxySQL](../connect-database/proxysql.md)              | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span>  | <span style="color:red">✘</span>  | <span style="color:red">✘</span> |
| [HAProxy](../connect-database/haproxy.md)                | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span>  | <span style="color:red">✘</span>  | <span style="color:red">✘</span> |

## Modify existing services

To change the parameters of a previously-added service, remove the service and re-add it with the new parameters.

## New to PMM?
If you're setting up monitoring for the first time, follow the installation and setup instructions in the [PMM installation overview](../../index.md).