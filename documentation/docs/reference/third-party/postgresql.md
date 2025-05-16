# Configure PMM with external PostgreSQL

Percona Monitoring and Management (PMM) can be configured to use an external PostgreSQL database instead of its built-in instance. This provides several advantages, including:

- enhanced high availability (HA) capabilities
- improved performance with dedicated database servers
- integration with existing database infrastructure
- better control over data retention and backups

To configure PMM Server to connect to an external PostgreSQL database running on the same host or a remote server, set up the required environment variables, configure SSL for secure connections, and ensure proper permissions for both PMM components and Grafana.

## Prerequisites
Before configuring PMM with an external PostgreSQL database, ensure you have:

- a PostgreSQL 14+ server accessible from your PMM Server
- basic understanding of Docker if using containerized deployment

## Configuration overview
To configure PMM Server to connect to an external PostgreSQL database:

- set up the external PostgreSQL server with required databases and permissions
- configure required environment variables for both PMM components and Grafana
- disable the built-in PostgreSQL server
- start PMM Server with the appropriate configuration

## Environment variables
!!! caution alert alert-warning "Important for PMM 3.2.0 and later"
    Due to a regression in Grafana 11.6 (included in PMM 3.2.0+), the `GF_DATABASE_URL` environment variable is no longer sufficient for configuring Grafana's connection to an external PostgreSQL database. When using PMM 3.2.0 or later with an external PostgreSQL, you must use the individual `GF_DATABASE_*` environment variables.

### PMM PostreSQL variables
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

By default, communication between the PMM Server and the database is not encrypted. To secure a connection, follow [PostgreSQL SSL instructions](https://www.postgresql.org/docs/14/ssl-tcp.html) and provide `POSTGRES_SSL_*` variables.

## Grafana database configuration
When using an external PostgreSQL database with PMM, configure both PMM's components and Grafana to use the external database.

- For PMM versions prior to 3.2.0, use a single `GF_DATABASE_URL` in the format `postgres://USER:PASSWORD@HOST:PORT/DATABASE_NAME`.
- For PMM 3.2.0 and later, Grafana requires individual database parameters instead of a single connection URL. Use the following environment variables:

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

### Configuration requirements
To successfully use an external PostgreSQL database with PMM:

- Ensure both PMM Server and Grafana database connections are configured. This means providing the appropriate `PMM_POSTGRES_*` environment variables for PMM's internal operations and the `GF_DATABASE_*` variables (or `GF_DATABASE_URL` for PMM versions prior to 3.2.0) for Grafana's data source.
- Enable the `pg_stat_statements` extension in the PostgreSQL database that PMM will connect to. This extension enables PMM to collect performance statistics.
- Do not specify `GF_DATABASE_TYPE`as PMM uses PostgreSQL for external database connection

## Set up PostgreSQL for PMM 

### 1. Prepare the PostgreSQL Server

To use PostgreSQL as an external database with PMM:
{.power-number}

1.  Pull the PostgreSQL Docker image:
    ```sh
    docker pull postgres:14
    ```

2.  Create a Docker volume for PostgreSQL data:
    ```bash
    docker volume create pg_data
    ```

3.  Create a directory where PostgreSQL will find initialization SQL scripts:
    ```sh
    mkdir -p /path/to/queries
    ```

4.  Create an `init.sql.template` file in the directory with the following content:

    ```sql
    CREATE DATABASE "pmm-managed";
    CREATE USER <YOUR_PG_USERNAME> WITH ENCRYPTED PASSWORD '<YOUR_PG_PASSWORD>';
    GRANT ALL PRIVILEGES ON DATABASE "pmm-managed" TO <YOUR_PG_USERNAME>;

    CREATE DATABASE grafana;
    CREATE USER <YOUR_GF_USERNAME> WITH ENCRYPTED PASSWORD '<YOUR_GF_PASSWORD>';
    GRANT ALL PRIVILEGES ON DATABASE grafana TO <YOUR_GF_USERNAME>;

    \c "pmm-managed"
    CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
    ```

5.  Replace the placeholders with your actual values:

    ```sh
    sed -e 's/<YOUR_PG_USERNAME>/'"$PG_USERNAME"'/g' \
        -e 's/<YOUR_PG_PASSWORD>/'"$PG_PASSWORD"'/g' \
        -e 's/<YOUR_GF_USERNAME>/'"$GF_USERNAME"'/g' \
        -e 's/<YOUR_GF_PASSWORD>/'"$GF_PASSWORD"'/g' \
        init.sql.template > init.sql
    ```

6.  Run the PostgreSQL container:

    ```sh
    docker run -d \
      --name pg \
      -p 5432:5432 \
      -e POSTGRES_PASSWORD=${PG_PASSWORD} \
      -v /path/to/queries:/docker-entrypoint-initdb.d \
      -v pg_data:/var/lib/postgresql/data \
      postgres:14 \
      postgres -c shared_preload_libraries=pg_stat_statements \
              -c pg_stat_statements.max=10000 \
              -c pg_stat_statements.track=all \
              -c pg_stat_statements.save=off
    ```

### 2. Configure SSL (optional)

If you need to secure the connection with SSL:
{.power-number}

1. Generate all necessary SSL certificates.
2. Deploy certificates with the appropriate permissions:
   ```sh
    # Example directory structure for certificates:
    /pmm-server-certificates# ls -la
    drwxr-xr-x 1 root    root    4096 Apr  5 12:43 .
    drwxr-xr-x 1 root    root    4096 Apr  5 12:43 ..
    -rw------- 1 grafana grafana 1391 Apr  5 12:38 certificate_authority.crt
    -rw------- 1 grafana grafana 1257 Apr  5 12:38 pmm_server.crt
    -rw------- 1 grafana grafana 1708 Apr  5 12:38 pmm_server.key
   ```
3. Configure PostgreSQL for SSL by updating your PostgreSQL container run command:

   ```sh
   docker run -d \
     --name pg \
     -p 5432:5432 \
     -e POSTGRES_PASSWORD=${PG_PASSWORD} \
     -v /path/to/queries:/docker-entrypoint-initdb.d \
     -v pg_data:/var/lib/postgresql/data \
     -v /path/to/certificates:/etc/postgresql/certs \
     postgres:14 \
     postgres -c shared_preload_libraries=pg_stat_statements \
              -c pg_stat_statements.max=10000 \
              -c pg_stat_statements.track=all \
              -c pg_stat_statements.save=off \
              -c ssl=on \
              -c ssl_ca_file=/etc/postgresql/certs/certificate_authority.crt \
              -c ssl_key_file=/etc/postgresql/certs/external_postgres.key \
              -c ssl_cert_file=/etc/postgresql/certs/external_postgres.crt \
              -c hba_file=/path/to/pg_hba.conf
   ```
 
4. Create a `pg_hba.conf` file that enforces SSL:

   ```sh
   local     all         all                                    trust
   hostnossl all         example_user all                       reject
   hostssl   all         example_user all                       cert
   ```
### 3. Run PMM Server with external PostgreSQL

Now that PostgreSQL is set up, configure PMM Server to use it:

=== "PMM 3.1.x and 3.0.0"
    ```sh
    docker run -d \
    -p 443:443 \
    -v pmm-data:/srv \
    -e PMM_POSTGRES_ADDR=postgres-host:5432 \
    -e PMM_POSTGRES_DBNAME=pmm-managed \
    -e PMM_POSTGRES_USERNAME=pmm_user \
    -e PMM_POSTGRES_DBPASSWORD=pmm_password \
    -e GF_DATABASE_URL=postgres://grafana_user:grafana_password@postgres-host:5432/grafana \
    -e GF_DATABASE_SSL_MODE=$GF_SSL_MODE \
    -e GF_DATABASE_CA_CERT_PATH=$GF_CA_PATH \
    -e GF_DATABASE_CLIENT_KEY_PATH=$GF_KEY_PATH \
    -e GF_DATABASE_CLIENT_CERT_PATH=$GF_CERT_PATH \
    -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
    --name pmm-server \
    percona/pmm-server:3
    ```
=== "PMM 3.2.0 and later"
    ```sh
    docker run -d \
    -p 443:443 \
    -v pmm-data:/srv \
    -e PMM_POSTGRES_ADDR=postgres-host:5432 \
    -e PMM_POSTGRES_DBNAME=pmm-managed \
    -e PMM_POSTGRES_USERNAME=pmm_user \
    -e PMM_POSTGRES_DBPASSWORD=pmm_password \
    -e GF_DATABASE_HOST=postgres-host:5432 \
    -e GF_DATABASE_NAME=grafana \
    -e GF_DATABASE_USER=grafana_user \
    -e GF_DATABASE_PASSWORD=grafana_password \
    -e GF_DATABASE_SSL_MODE=$GF_SSL_MODE \
    -e GF_DATABASE_CA_CERT_PATH=$GF_CA_PATH \
    -e GF_DATABASE_CLIENT_KEY_PATH=$GF_KEY_PATH \
    -e GF_DATABASE_CLIENT_CERT_PATH=$GF_CERT_PATH \
    -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
    --name pmm-server \
    percona/pmm-server:3
    ```
## Docker Compose example
When using Docker Compose to run PMM with an external PostgreSQL database, make sure to configure both PMM and Grafana database parameters:
{.power-number}

1. Create a `docker-compose.yml` file with the following content (adjust values as needed):

   ```yaml
   services:
     pmm-server:
       image: percona/pmm-server:3.2.0
       ports:
         - "443:443"
       volumes:
         - pmm-data:/srv
       environment:
         # PMM PostgreSQL connection variables
         - PMM_POSTGRES_ADDR=your_host:your_port
         - PMM_POSTGRES_DBNAME=your_pmm_db_name
         - PMM_POSTGRES_USERNAME=your_pmm_user
         - PMM_POSTGRES_DBPASSWORD=your_pmm_password
         # Grafana PostgreSQL connection variables (for PMM 3.2.0+)
         - GF_DATABASE_USER=your_grafana_user
         - GF_DATABASE_PASSWORD=your_grafana_password
         - GF_DATABASE_HOST=your_host:your_port
         - GF_DATABASE_NAME=your_grafana_db_name
         # Disable built-in PostgreSQL
         - PMM_DISABLE_BUILTIN_POSTGRES=1
       restart: always

   volumes:
     pmm-data:
   ```
2. Start the PMM Server service:

   ```sh
   docker-compose stop pmm-server
   docker-compose rm pmm-server
   docker-compose up -d pmm-server
   ```

3. Restart the service after making changes:
   ```sh
   docker-compose stop pmm-server
   docker-compose rm pmm-server
   docker-compose up -d pmm-server
   ```

## Troubleshooting
If you encounter issues when configuring PMM with an external PostgreSQL database, check the following:

- make sure all required environment variables are set (both PMM and Grafana variables)
- verify that PostgreSQL is running and accessible from the PMM Server container
- check that the correct database names and credentials are used
- for PMM 3.2.0+, make sure you're using the individual Grafana database parameters instead of `GF_DATABASE_URL`
- confirm that `pg_stat_statements` extension is enabled in the PostgreSQL database
- check the Grafana logs for database connection issues
