# Connect databases to PMM

PMM supports monitoring for MySQL, PostgreSQL, MongoDB, Valkey/Redis, major cloud database services, and infrastructure components.

## Supported database versions

PMM supports database versions that are within their active support lifecycle. For the full list of supported versions per product, see the [Percona Release Lifecycle Overview](https://www.percona.com/release-lifecycle-overview/).

Connecting a database version that has reached end of life may work, but is not tested or supported. We recommend upgrading to a supported version to ensure full compatibility with PMM features.

## Supported database technologies

| Database | Local monitoring | Remote monitoring | Query Analytics | Backup integration |
|----------|:----------------:|:-----------------:|:---------------:|:-----------------:|
| [MySQL](mysql/mysql.md)¹ | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> |
| [PostgreSQL](postgresql.md) | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [MongoDB](mongodb.md) | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> |
| [Valkey / Redis](valkey-redis.md) | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> | <span style="color:red">✘</span> |
| [Amazon RDS / Aurora](aws.md) | <span style="color:red">✘</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [Microsoft Azure](azure.md) | <span style="color:red">✘</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [Google Cloud SQL](google.md) | <span style="color:red">✘</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [ProxySQL](proxysql.md) | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> | <span style="color:red">✘</span> |
| [HAProxy](haproxy.md) | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> | <span style="color:red">✘</span> |
| [Linux](linux.md) | <span style="color:green">✔</span> | <span style="color:red">✘</span> | <span style="color:red">✘</span> | <span style="color:red">✘</span> |
| [External services](external.md) | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> | <span style="color:red">✘</span> |
| [Remote instances](remote.md) | <span style="color:red">✘</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> | <span style="color:red">✘</span> |

¹ Includes Percona Server for MySQL, Percona XtraDB Cluster, and MariaDB.

## Modify existing services

To change the parameters of a previously-added service, remove the service and re-add it with the new parameters.

## New to PMM?

If you're setting up monitoring for the first time, follow the installation and setup instructions in the [PMM installation overview](../../index.md).
