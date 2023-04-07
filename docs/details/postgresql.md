# PostgreSQL

You can use an external PostgreSQL database instance outside the PMM Server container running on the same or other hosts.

## Environment variables

PMM predefines certain flags that allow you to use PostgreSQL parameters as environment variables:

!!! caution alert alert-warning "Warning"
The `POSTGRES_*` environment variables are experimental and subject to change. It is recommended that you use these variables for testing purposes only and not on production. The minimum supported PostgreSQL server version is 14.

To use PostgreSQL as an external database instance, use the following environment variables:

| Environment variable         | Flag                                                                                                    | Description                                                                                                                                                                                      |
| ---------------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| POSTGRES_ADDR                | [postgres-addr](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-HOST)               | Hostname and port for external PostgreSQL database.                                                                                                                                              |
| POSTGRES_DBNAME              | [postgres-name](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-DBNAME)             | Database name for external or internal PostgreSQL database.                                                                                                                                      |
| POSTGRES_USERNAME            | [postgres-username](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-USER)           | PostgreSQL user name to connect as.                                                                                                                                                              |
| POSTGRES_DBPASSWORD          | [postgres-password](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-PASSWORD)       | Password to be used for database authentication.                                                                                                                                                 |
| POSTGRES_SSL_MODE            | [postgres-ssl-mode](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-SSLMODE)        | This option determines whether or with what priority a secure SSL TCP/IP connection will be negotiated with the database. Currently supported: `disable`, `require`, `verify-ca`, `verify-full`. |
| POSTGRES_SSL_CA_PATH         | [postgres-ssl-ca-path](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-SSLROOTCERT) | This parameter specifies the name of a file containing SSL certificate authority (CA) certificate(s).                                                                                            |
| POSTGRES_SSL_KEY_PATH        | [postgres-ssl-key-path](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-SSLKEY)     | This parameter specifies the location for the secret key used for the client certificate.                                                                                                        |
| POSTGRES_SSL_CERT_PATH       | [postgres-ssl-cert-path](https://www.postgresql.org/docs/14/libpq-connect.html#LIBPQ-CONNECT-SSLCERT)   | This parameter specifies the file name of the client SSL certificate.                                                                                                                            |
| PMM_DISABLE_BUILTIN_POSTGRES |                                                                                                         | Environment variable to disable built-in PMM server database.                                                                                                                                    |

By default, communication between the PMM server and the database is not encrypted. To secure a connection, follow [PostgeSQL SSL instructions](https://www.postgresql.org/docs/14/ssl-tcp.html) and provide `POSTGRES_SSL_*` variables.

**Example**

To use PostgreSQL as an external database:

1. Generate all necessary SSL certificates.
2. Deploy Percona Server with certificates under read-only permissions and Grafana user and Grafana group.
3. Attach `pg_hba.conf` and certificates to the PostgreSQL image.
4. Run PostgreSQL server.

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

- Run PMM Server.

```sh
docker run
--name percona-server
-e POSTGRES_ADDR=$ADDRESS:$PORT
-e POSTGRES_DBNAME=$DBNAME
-e POSTGRES_USERNAME=$USER
-e POSTGRES_DBPASSWORD=$PASSWORD
-e POSTGRES_SSL_MODE=$SSL_MODE
-e POSTGRES_SSL_CA_PATH=$CA_PATH
-e POSTGRES_SSL_KEY_PATH=$KEY_PATH
-e POSTGRES_SSL_CERT_PATH=$CERT_PATH
-e PMM_DISABLE_BUILTIN_POSTGRES=true
percona/pmm-server:2
```
