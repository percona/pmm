# Configure PMM with external PostgreSQL

Percona Monitoring and Management (PMM) can be configured to use an external PostgreSQL database instead of its built-in instance. 
To configure PMM Server to connect to an external PostgreSQL database running on the same host or a remote server, set up the required environment variables, configure SSL for secure connections, and ensure proper permissions for both PMM components and Grafana.

## Environment variables
!!! caution alert alert-warning "Important for PMM 3.2.0 and later"
    Due to a regression in Grafana 11.6 (included in PMM 3.2.0+), the `GF_DATABASE_URL` environment variable is no longer sufficient for configuring Grafana's connection to an external PostgreSQL database. When using PMM 3.2.0 or later with an external PostgreSQL, you must use the individual `GF_DATABASE_*` environment variables as described below.

To use PostgreSQL as an external database instance, use the following environment variables:

| Environment variable         | Flag                                                                                                    | Description                                                                                                                                                                                      |
| ---------------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| PMM_POSTGRES_ADDR                | [postgres-addr](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-HOST)               | Hostname and port for external PostgreSQL database.                                                                                                                                              |
| PMM_POSTGRES_DBNAME              | [postgres-name](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-DBNAME)             | Database name for external or internal PostgreSQL database.                                                                                                                                      |
| PMM_POSTGRES_USERNAME            | [postgres-username](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-USER)           | PostgreSQL user name to connect as.                                                                                                                                                              |
| PMM_POSTGRES_DBPASSWORD          | [postgres-password](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-PASSWORD)       | Password to be used for database authentication.                                                                                                                                                 |
| PMM_POSTGRES_SSL_MODE            | [postgres-ssl-mode](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-SSLMODE)        | This option determines whether or with what priority a secure SSL TCP/IP connection will be negotiated with the database. Currently supported: `disable`, `require`, `verify-ca`, `verify-full`. |
| PMM_POSTGRES_SSL_CA_PATH         | [postgres-ssl-ca-path](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-SSLROOTCERT) | This parameter specifies the name of a file containing SSL certificate authority (CA) certificate(s).                                                                                            |
| PMM_POSTGRES_SSL_KEY_PATH        | [postgres-ssl-key-path](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-SSLKEY)     | This parameter specifies the location for the secret key used for the client certificate.                                                                                                        |
| PMM_POSTGRES_SSL_CERT_PATH       | [postgres-ssl-cert-path](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-SSLCERT)   | This parameter specifies the file name of the client SSL certificate.                                                                                                                            |
| PMM_DISABLE_BUILTIN_POSTGRES |                                                                                                         | Environment variable to disable built-in PMM Server database. Note that Grafana depends on built-in PostgreSQL. And if the value of this variable is "true", then it is necessary to pass all the parameters associated with Grafana to use external PostgreSQL.                                                                                                                                    |

By default, communication between the PMM Server and the database is not encrypted. To secure a connection, follow [PostgeSQL SSL instructions](https://www.postgresql.org/docs/14/ssl-tcp.html) and provide `POSTGRES_SSL_*` variables.

## Grafana database configuration

To use Grafana with external PostgreSQL, add `GF_DATABASE_*` environment variables in the format `postgres://USER:PASSWORD@HOST:PORT/DATABASE_NAME`.

For PMM 3.2.0 and later, Grafana requires individual database parameters instead of a single connection URL.

Use the following environment variables to configure Grafana with an external PostgreSQL database:

| Environment variable     | Description                                                      |
|--------------------------|------------------------------------------------------------------|
| GF_DATABASE_HOST         | Hostname and port of the PostgreSQL server (e.g., `host:5432`)   |
| GF_DATABASE_NAME         | Database name for Grafana                                        |
| GF_DATABASE_USER         | PostgreSQL user for Grafana                                      |
| GF_DATABASE_PASSWORD     | Password for the Grafana database user                           |
| GF_DATABASE_SSL_MODE     | SSL mode for database connection (disable, require, verify-ca, verify-full) |
| GF_DATABASE_CA_CERT_PATH | Path to CA certificate file                                      |
| GF_DATABASE_CLIENT_KEY_PATH | Path to client key file                                       |
| GF_DATABASE_CLIENT_CERT_PATH | Path to client certificate file                              |

Note: PMM automatically uses PostgreSQL for external database connections, so you don't need to specify `GF_DATABASE_TYPE`.

**Example**

To use PostgreSQL as an external database:
{.power-number}

1. Generate all necessary SSL certificates.
2. Deploy PMM Server with certificates under read-only permissions and Grafana user and Grafana group:

    ```sh
    /pmm-server-certificates# la -la
    drwxr-xr-x 1 root    root    4096 Apr  5 12:43 .
    drwxr-xr-x 1 root    root    4096 Apr  5 12:43 ..
    -rw------- 1 grafana grafana 1391 Apr  5 12:38 certificate_authority.crt
    -rw------- 1 grafana grafana 1257 Apr  5 12:38 pmm_server.crt
    -rw------- 1 grafana grafana 1708 Apr  5 12:38 pmm_server.key
    ```

3. Attach `pg_hba.conf` and certificates to the PostgreSQL image:

    ```sh
    /external-postgres-configuration# cat pg_hba.conf 
    local     all         all                                    trust
    hostnossl all         example_user all                       reject
    hostssl   all         example_user all                       cert
        
        
    /external-postgres-certificates# ls -la
    drwxr-xr-x 1 root     root     4096 Apr  5 12:38 .
    drwxr-xr-x 1 root     root     4096 Apr  5 12:43 ..
    -rw------- 1 postgres postgres 1391 Apr  5 12:38 certificate_authority.crt
    -rw------- 1 postgres postgres 1407 Apr  5 12:38 external_postgres.crt
    -rw------- 1 postgres postgres 1708 Apr  5 12:38 external_postgres.key
    ```
    
4. Create `user` and `database` for pmm-server to use. Set appropriate rights and access.

5. Install `pg_stat_statements` in PostgreSQL in order to have all metrics according to [this](../../install-pmm/install-pmm-client/connect-database/postgresql.md) handy document.

6. Run PostgreSQL server:

    ```sh
    docker run
    --name external-postgres
    -e POSTGRES_PASSWORD=secret
    <image_id>
    postgres
    -c shared_preload_libraries=pg_stat_statements
    -c pg_stat_statements.max=10000
    -c pg_stat_statements.track=all
    -c pg_stat_statements.save=off
    -c ssl=on
    -c ssl_ca_file=$CA_PATH
    -c ssl_key_file=$KEY_PATH
    -c ssl_cert_file=$CERT_PATH
    -c hba_file=$HBA_PATH
    ```

7. Run PMM Server:

    === "PMM < 3.2.0"
    ```sh
    -e GF_DATABASE_URL=$GF_DATABASE_URL
    -e GF_DATABASE_SSL_MODE=$GF_SSL_MODE
    -e GF_DATABASE_CA_CERT_PATH=$GF_CA_PATH
    -e GF_DATABASE_CLIENT_KEY_PATH=$GF_KEY_PATH
    -e GF_DATABASE_CLIENT_CERT_PATH=$GF_CERT_PATH
    percona/pmm-server:3
    ```
    === "PMM >= 3.2.0"
        ```sh
        -e GF_DATABASE_HOST=$GF_HOST:$GF_PORT
        -e GF_DATABASE_NAME=$GF_DATABASE_NAME
        -e GF_DATABASE_USER=$GF_USER
        -e GF_DATABASE_PASSWORD=$GF_PASSWORD
        -e GF_DATABASE_SSL_MODE=$GF_SSL_MODE
        -e GF_DATABASE_CA_CERT_PATH=$GF_CA_PATH
        -e GF_DATABASE_CLIENT_KEY_PATH=$GF_KEY_PATH
        -e GF_DATABASE_CLIENT_CERT_PATH=$GF_CERT_PATH
        percona/pmm-server:3
        ```