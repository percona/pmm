# Google Cloud Platform

PMM can monitor MySQL or PostgreSQL instances hosted on the [Google Cloud Platform][GOOGLE_CLOUD].

The connection can be direct, or indirect using [Cloud SQL Proxy][GOOGLE_CLOUD_SQL_PROXY].

## MySQL

1. [Set up a MySQL instance on Google Cloud][GOOGLE_CLOUD_MYSQL].

2. The database server must be accessible by PMM Client. If PMM Client is not also hosted on GCP, you will need to add a network interface with a public interface.

3. Configure *Performance Schema* on the MySQL server. Using the GCP console's *Cloud Shell* or your own `gcloud` installation, run:

    ```sh
    gcloud sql instances patch <instance_name> --database-flags performance_schema=on
    ```

4. Log into the PMM user interface.

5. Select {{icon.configuration}} *Configuration* → {{icon.inventory}} *PMM Inventory* → {{icon.addinstance}} *Add Instance*.

6. Click *MySQL Add a remote instance*.

7. Fill in the details for the remote MySQL instance.

    - Ensure *Use performance schema* is selected.

8. Click *Add service*.

9. Check for values in the *MySQL Instance Overview* dashboard and in *Query Analytics*.

## PostgreSQL

1. [Set up a PostgreSQL instance on Google Cloud][GOOGLE_CLOUD_POSTGRESQL].

2. The database server must be accessible by PMM Client. If PMM Client is not also hosted on GCP, you will need to add a network interface with a public interface.

3. Configure `pg_stat_statements`. Open an interactive SQL session with your GCP PostgreSQL server and run:

    ```sql
    CREATE EXTENSION pg_stat_statements;
    ```

4. Log into the PMM user interface.

5. Select {{icon.configuration}} *Configuration* → {{icon.inventory}} *PMM Inventory* → {{icon.addinstance}} *Add Instance*.

6. Select *PostgreSQL Add a remote instance*.

7. Fill in the details for the remote PostgreSQL instance:

    - In *Stat tracking options*, select *PG Stat Statements*.

8. Click *Add service*.

9. Check for values in the *PostgreSQL Instance Overview* dashboard and *Query Analytics*.

## Cloud SQL Proxy

### MySQL

1. Create instance on GCP.

2. Note connection as `<project_id>:<zone>:<db_instance_name>`.

3. [Enable Admin API][GOOGLE_CLOUD_ADMIN_API] and download the JSON credential file.

4. Enable *Performance Schema*.

5. Run Cloud SQL Proxy (runs on PMM Client node).

    - As a Docker container:

        ```sh
        docker run -d \
        -v ~/path/to/admin-api-file.json:/config \
        -p 127.0.0.1:3306:3306 \
        gcr.io/cloudsql-docker/gce-proxy:1.19.1 \
        /cloud_sql_proxy \
        -instances=example-project-NNNN:us-central1:mysql-for-pmm=tcp:0.0.0.0:3306 \
        -credential_file=/config
        ```

    - On Linux:

        ```sh
        wget https://dl.google.com/cloudsql/cloud_sql_proxy.linux.amd64 -O cloud_sql_proxy
        chmod +x cloud_sql_proxy
        ./cloud_sql_proxy -instances=example-project-NNNN:us-central1:mysql-for-pmm=tcp:3306 \
        -credential_file=/path/to/credential-file.json
        ```

6. Add instance.

    ```sh
    pmm-admin add mysql --host=127.0.0.1 --port=3306 \
    --username=root --password=secret \
    --service-name=MySQLGCP --query-source=perfschema
    ```

### PostgreSQL

1. Create instance on GCP.

2. Note connection as `<project_id>:<zone>:<db_instance_name>`.

3. [Enable Admin API][GOOGLE_CLOUD_ADMIN_API] and download the JSON credential file.

4. Run Cloud SQL Proxy.

    ```sh
    ./cloud_sql_proxy -instances=example-project-NNNN:us-central1:pg-for-pmm=tcp:5432 \
    -credential_file=/path/to/credential-file.json
    ```

5. Log into PostgreSQL.

6. Load extension:

    ```sql
    CREATE EXTENSION pg_stat_statements;
    ```

7. Add service:

    ```sh
    pmm-admin add postgresql --host=127.0.0.1 --port=5432 \
    --username="postgres" --password=secret --service-name=PGGCP
    ```

[GOOGLE_CLOUD_SQL]: https://cloud.google.com/sql
[GOOGLE_CLOUD]: https://cloud.google.com/
[GOOGLE_CLOUD_MYSQL]: https://cloud.google.com/sql/docs/mysql/quickstart
[GOOGLE_CLOUD_POSTGRESQL]: https://cloud.google.com/sql/docs/postgres/quickstart
[GOOGLE_CLOUD_SQL_PROXY]: https://cloud.google.com/sql/docs/mysql/connect-overview#cloud_sql_proxy
[GOOGLE_CLOUD_ADMIN_API]: https://cloud.google.com/sql/docs/mysql/admin-api#console
