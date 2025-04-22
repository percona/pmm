# Connect databases to PMM
Percona Monitoring and Management (PMM) supports monitoring for MySQL/MariaDB, PostgreSQL, MongoDB, and various cloud database services. 

Here's how to connect various database technologies to PMM:

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

## Setup overview

If you're setting up monitoring for the first time, follow these general steps:

1. [Install PMM Server](../../install-pmm-server/index.md)
2. [Install PMM Client](../../install-pmm-client/index.md) on your database server or a system with network access to it
3. [Register the client node](../../register-client-node/index.md) with PMM Server
4. Follow the relevant database instructions:
    - [MySQL](../connect-database/mysql/mysql.md)
    - [PostgreSQL](../connect-database/postgresql.md)
    - [MongoDB](../connect-database/mongodb.md)
    - [AWS RDS/Aurora](../connect-database/aws.md)
    - [Azure Database](../connect-database/azure.md)
    - [Google Cloud SQL](../connect-database/google.md)
    - [ProxySQL](../connect-database/proxysql.md)
    - [HAProxy](../connect-database/haproxy.md)
5. [Verify monitoring is working](../../../use/using-pmm.md) through the PMM dashboard

!!! tip "Changing service parameters"
    To change the parameters of a previously-added service, remove the service and re-add it with the new parameters.

