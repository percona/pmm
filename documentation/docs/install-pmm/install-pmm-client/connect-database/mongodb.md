# Connect MongoDB instance

Connect a MongoDB instance to PMM to monitor a [MongoDB] or [Percona Server for MongoDB] database server.

## Prerequisites

Before you start, ensure you have:

- [PMM Server installed](../../install-pmm-server/index.md) and running with a known IP address or hostname accessible from the Client node.
- [PMM Client installed](../../install-pmm-client/index.md) and the [nodes are registered with PMM Server](../../register-client-node/index.md).
- Superuser (root) access on the Client host.
- `adminUserAnyDatabase` or superuser role privilege on the MongoDB database servers that you want to monitor.
- MongoDB server version 4.0 or higher.

## Create PMM account and set permissions

We recommend using a dedicated account to connect PMM Client to the monitored database instance. The permissions required depend on which PMM features you plan to use.

Run the example commands below in a mongo shell session to:

-  Create custom roles with the privileges required for creating/restoring backups and working with Query Analytics (QAN).
-  Create/update a database user with these roles, plus the built-in  `clusterMonitor` role.
  
!!! caution alert alert-warning "Important"
    Values for username (`user`) and password (`pwd`) are examples. Replace them before using these code snippets.

=== "Monitoring and QAN privileges"
    This role grants the essential minimum privileges needed for monitoring and QAN:

        ```{.javascript data-prompt=">"}
        db.getSiblingDB("admin").createRole({
        "role": "explainRole",
        "privileges": [
            {
                "resource": { "db": "", "collection": "" },
                "actions": [ "dbHash", "find", "listIndexes", "listCollections", "collStats", "dbStats", "indexStats" ]
            },
            {
                "resource": { "db": "", "collection": "system.version" },
                "actions": [ "find" ]
            },
            {
                "resource": { "db": "", "collection": "system.profile" },
                "actions": [ "dbStats", "collStats", "indexStats" ]
            }         
        ],
        "roles": [ { role: "directShardOperations", db: "admin" } ]
        })
        ```    
        
=== "Full backup management privileges"
    This role provides the necessary privileges for using PMM's backup management features. It is required only if you plan to use this feature:

        ```{.javascript data-prompt=">"}
        db.getSiblingDB("admin").createRole({
            "role": "pbmAnyAction",
            "privileges": [
            {
                "resource": { "anyResource": true  },
                "actions": [ "anyAction" ]
            }
            ],
            "roles": []
        });
        ```

### Create/update user and assign created roles

Create or update a user with the minimum required privileges for monitoring by assigning the following roles:

=== "MongoDB < 8.0"

    ```{.javascript data-prompt=">"}
    db.getSiblingDB("admin").({
        "user": "pmm",
        "pwd": "pmm",
        "roles": [
            { "db": "admin", "role": "explainRole" },
            { "db": "local", "role": "read" },
            { "db": "admin", "role": "clusterMonitor" }
        ]
    })
    ```

    If you intend to use PMM's backup management features, also grant these additional permissions: 
    ```{.javascript data-prompt=">"}
    db.getSiblingDB("admin").createUser({
        "user": "pmm",
        "pwd": "pmm",
        "roles": [
            { "db" : "admin", "role": "explainRole" },
            { "db" : "local", "role": "read" },
            { "db" : "admin", "role" : "readWrite", "collection": "" },
            { "db" : "admin", "role" : "backup" },
            { "db" : "admin", "role" : "clusterMonitor" },
            { "db" : "admin", "role" : "restore" },
            { "db" : "admin", "role" : "pbmAnyAction" }
        ]
    })      
    ```

=== "MongoDB 8.0+"
    MongoDB 8.0 introduced stricter security for direct shard access. For MongoDB 8.0 and later, the PMM user also requires the `directShardOperations` role to collect complete metrics from all cluster components.

    ```{.javascript data-prompt=">"}
    db.getSiblingDB("admin").createUser({
        "user": "pmm",
        "pwd": "pmm",  // Replace with a secure password
        "roles": [
            { "db": "admin", "role": "explainRole" },
            { "db": "local", "role": "read" },
            { "db": "admin", "role": "clusterMonitor" },
            { "db": "admin", "role": "directShardOperations" }
        ]
    })
    ```

    If you intend to use PMM's backup management features, grant these additional permissions: 
    ```{.javascript data-prompt=">"}
    db.getSiblingDB("admin").createUser({
        "user": "pmm",
        "pwd": "pmm",  // Replace with a secure password
        "roles": [
            { "db" : "admin", "role": "explainRole" },
            { "db" : "local", "role": "read" },
            { "db" : "admin", "role" : "readWrite", "collection": "" },
            { "db" : "admin", "role" : "backup" },
            { "db" : "admin", "role" : "clusterMonitor" },
            { "db" : "admin", "role" : "restore" },
            { "db" : "admin", "role" : "pbmAnyAction" },
            { "db" : "admin", "role": "directShardOperations" }
        ]
    })      
    ```

## Enable MongoDB profiling for Query Analytics (QAN)

To use PMM QAN, you must turn on MongoDB's [profiling feature]. By default, profiling is turned off as it can adversely affect the performance of the database server.

Choose one of the following methods to enable profiling:

=== "MongoDB configuration file (Recommended)"
    This method ensures your settings persist across server restarts and system reboots. It's the recommended approach for production environments:

    {.power-number}

    1. Edit the configuration file (usually `/etc/mongod.conf`).

    2. Create or add this to the `operationProfiling` section. ([Read more][MONGODB_CONFIG_OP_PROF].)

        ```yml
        operationProfiling:
            mode: all
            slowOpThresholdMs: 200
            rateLimit: 100 # (Only available with Percona Server for MongoDB.)
        ```

        !!! caution alert alert-warning "Important"
            This is a [YAML] file. Make sure to pay attention to indentation, as it is crucial for correct parsing.

    3. Restart the `mongod` service using the appropriate command for your system. For example, for `systemd`:

        ```sh
        systemctl restart mongod
        ```

=== "On CLI"
    Use this method when starting the MongoDB server manually:

    ```sh
    mongod --dbpath=DATABASEDIR --profile 2 --slowms 200 --rateLimit 100
    ```

    - `--dbpath`: The path to database files (usually `/var/lib/mongo`).
    - `--profile`: The MongoDB profiling level. A value of `2` tells the server to collect profiling data for *all* operations. To lower the load on the server, use a value of `1` to only record slow operations.
    - `--slowms`: An operation is classified as *slow* if it runs for longer than this number of milliseconds.
    - `--rateLimit`: (Only available with Percona Server for MongoDB.) The sample rate of profiled queries. A value of `100` means sample every 100th fast query. ([Read more][PSMDB_RATELIMIT].)

        !!! caution alert alert-warning "Caution"
            Smaller values improve accuracy but can adversely affect the performance of your server.

=== "In MongoDB shell (temporary)"

    This method enables profiling until the next server restart. Profiling must be enabled for **each** database you want to monitor. For example, to enable the profiler in the `testdb`, run this:

    ```json
    use testdb
    db.setProfilingLevel(2, {slowms: 0})
    ```

    !!! note alert alert-primary ""
        If you have already [added a service](#add-mongodb-service-to-pmm), you should remove it and re-add it after changing the profiling level.

## Add MongoDB service to PMM

After configuring your database server, add a MongoDB service using either the user interface or the command line.

!!! caution alert alert-warning "Important"
    To monitor MongoDB sharded clusters, PMM requires access to all cluster components. Make sure to add all config servers, shards, and at least one mongos router. Otherwise, PMM will not be able to correctly collect metrics and populate dashboards.

=== "Add service via UI"

    To add a service with the UI:
    {.power-number}

    1. Select **PMM Configuration > Add Service > MongoDB**.

    2. Fill in the required fields.

    3. Click **Add service**.

    ![!](../../../images/PMM_Add_Instance_MongoDB.jpg)

=== "Add service via CLI"

    Use `pmm-admin` to add the database server as a service using one of these example commands:

    ==="Basic MongoDB instance"
        ```sh
        pmm-admin add mongodb \
        --username=pmm \
        --password=your_secure_password
        ```

    ==="Sharded cluster component"
        ```sh
        pmm-admin add mongodb \
        --username=pmm \
        --password=your_secure_password \
        --cluster=my_cluster_name \
        --replication-set=rs1  # Optional: specify replication set name
        ```

    ==="SSL/TLS secured MongoDB"
        ```sh
        pmm-admin add mongodb \
        --username=pmm \
        --password=your_secure_password \
        --tls \
        --tls-certificate-key-file=/path/to/client.pem \
        --tls-certificate-key-file-password=cert_password \  # If certificate has password
        --tls-ca-file=/path/to/ca.pem \
        --authentication-mechanism=MONGODB-X509 \  # For X.509 authentication
        --authentication-database=$external      # For X.509 authentication
        ```

    When successful, PMM Client will print `MongoDB Service added` with the service's ID and name. Use the `--environment` and `--custom-labels` options to set tags for the service to help identify them.

    !!! hint alert alert-success "Tips"
        - When adding nodes to a sharded cluster, ensure to add each node separately using the `--cluster mycluster` option. This allows the [MongoDB Cluster Summary](../../../reference/dashboards/dashboard-mongodb-cluster-summary.md) dashboard to populate correctly. 
        - You can also use the `--replication-set` option to specify a replication set, although they are automatically detected. For instance, you can use `--replication-set config` for your config servers; `--replication-set rs1` for your servers in the first replica set, `--replication-set rs2` for your servers in the second replica set, and so on.
        - When running mongos routers in containers, specify the `diagnosticDataCollectionDirectoryPath` to ensure that pmm-agent can properly capture mongos metrics. For example: `mongos --setParameter diagnosticDataCollectionDirectoryPath=/var/log/mongo/mongos.diagnostic.data/`


## Verify MongoDB service configuration

1. Check service registration:
    ==="From the user interface"

        To check the service from the UI:
        {.power-number}

        1. Select **PMM Configuration > PMM Inventory**. 
        2. In the **Services** tab, verify the **Service name**, **Addresses**, and any other relevant values used when adding the service.
        3. In the **Options** column, expand the **Details** section and check that the Clients are using the desired data source.
        4. If your MongoDB instance is configured to use TLS, click on the **Use TLS for database connection** check box and fill in TLS certificates and keys.
        If you use TLS, the authentication mechanism is automatically set to `MONGODB-X509`.

        ![!](../../../images/PMM_Add_Instance_MongoDB_TLS.jpg)

    ==="On the command line"

        Look for your service in the output of this command:

        ```sh
        pmm-admin inventory list services --service-type=mongodb
        ```
2. Verify data collection:
    {.power-number}

    1. Open the **MongoDB Instances Overview** dashboard.
    2. Set the **Service Name** to the newly-added service.

3. Verify Query Analytics for the service:
    {.power-number}

    1. Open **PMM Query Analytics**.
    2. In the **Filters** panel:
        1. Under **Service Name**, select your service.
        2. Under **Service Type** select `mongodb`.
    3. Verify that query data appears (may take a few minutes to populate).

## Remove MongoDB service

==="Via UI"

    To remove the services added through the PMM interface:
    {.power-number}

    1. Go to PMM Configuration > PMM Inventory > Services**.
    2. In the **Status** column, check the tick box for the service you want to remove and click **Delete**.
    3. On the confirmation pop-up, click **Delete service** and select **Force mode** if you also to also delete associated Clients.
    

==="Via CLI"
    Replace `SERVICE_NAME` with the name you used when adding the service. You can list all services with `pmm-admin`:

    ```sh
    pmm-admin remove mongodb SERVICE_NAME
    ```

!!! seealso alert alert-info "See also"
    - [`pmm-admin add mongodb`](../../../use/commands/pmm-admin.md#mongodb)
    - [Troubleshooting connection difficulties]

[MongoDB]: https://www.mongodb.com/
[Percona Server for MongoDB]: https://www.percona.com/software/mongodb/percona-server-for-mongodb
[profiling feature]: https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/
[YAML]: http://yaml.org/spec/
[MONGODB_CONFIG_OP_PROF]: https://docs.mongodb.com/manual/reference/configuration-options/#operationprofiling-options
[PSMDB_RATELIMIT]: https://www.percona.com/doc/percona-server-for-mongodb/LATEST/rate-limit.html#enabling-the-rate-limit
[PMM_ADMIN_MAN_PAGE]: ../../../use/commands/pmm-admin.md#mongodb
[Troubleshooting connection difficulties]: ../../../troubleshoot/config_issues.md#connection-difficulties
