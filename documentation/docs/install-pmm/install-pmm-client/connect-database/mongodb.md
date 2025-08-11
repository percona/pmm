# Connect MongoDB databases to PMM

Connect a MongoDB instance to PMM to monitor a [MongoDB] or [Percona Server for MongoDB] database server.

## Prerequisites

Before you start, ensure you have:

- [PMM Server installed](../../install-pmm-server/index.md) and running with a known IP address or hostname accessible from the Client node.
- [PMM Client installed](../../install-pmm-client/index.md) and the nodes are registered with PMM Server.
- admin privileges to install and configure PMM Client on the host.
- preconfigured MongoDB user with appropriate monitoring privileges, or sufficient privileges to create the required roles and users.
- MongoDB server version 6.0 or higher. PMM may work with MongoDB versions as old as 4.4, but we recommend using MongoDB 6.0+ for complete feature support.

## Step 1: Set up MongoDB monitoring permissions

Set up MongoDB with a dedicated user for PMM and the required permissions. First, create custom roles with the necessary privileges, then assign them to a PMM-specific user.

Role privileges depend on:

- MongoDB version: 8.0+ requires the additional `directShardOperations` role for shard metrics
- Required features: basic monitoring only, or monitoring plus backup management.
- Query collection method: profiler or diagnostic log.

### Create monitoring role

After connecting to your MongoDB instance, create custom role with the privileges required for metric collection, working with Query Analytics (QAN) and optionally creating/restoring backups:
  
!!! caution alert alert-warning "Important"
    Values for username (`user`) and password (`pwd`) are examples. Replace them before using these code snippets.

=== "Minimum privileges"
    This role grants the essential minimum privileges needed for monitoring and QAN:
    ```javascript
    db.getSiblingDB("admin").createRole({
    "role": "pmmMonitor",
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
    "roles": [ ]
    })
    ```
        
=== "Full backup management privileges"
    If you plan to use PMM's backup features, create a role with full backup management privileges:

    ```javascript
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

### Create user and assign created role

After creating the role, create the PMM user and assign role based on your MongoDB version and requirements:

=== "MongoDB 8.0+ (Standard)"
    MongoDB 8.0 introduced stricter security for direct shard access. For MongoDB 8.0 and later, the PMM user also requires the `directShardOperations` role to collect complete metrics from all cluster components:

    ```javascript
    db.getSiblingDB("admin").createUser({
        "user": "pmm",
        "pwd": "<SECURE_PASSWORD>",  // Replace with a secure password
        "roles": [
            { "db": "admin", "role": "pmmMonitor" },
            { "db": "local", "role": "read" },
            { "db": "admin", "role": "clusterMonitor" },
            { "db": "admin", "role": "directShardOperations" }
        ]
    })
    ```
=== "MongoDB 8.0+ (With backups)"
    If you intend to use PMM's backup management features, create a user with grant these permissions: 

    ```javascript
    db.getSiblingDB("admin").createUser({
        "user": "pmm",
        "pwd": "<SECURE_PASSWORD>",  // Replace with a secure password
        "roles": [
            { "db": "admin", "role": "pmmMonitor" },
            { "db": "local", "role": "read" },
            { "db": "admin", "role": "readWrite", "collection": "" },
            { "db": "admin", "role": "backup" },
            { "db": "admin", "role": "clusterMonitor" },
            { "db": "admin", "role": "restore" },
            { "db": "admin", "role": "pbmAnyAction" },
            { "db": "admin", "role": "directShardOperations" }
        ]
    })      
    ```

=== "MongoDB <8.0 (Standard)"
    Create the PMM user with standard monitoring roles:

    ```javascript
    db.getSiblingDB("admin").createUser({
        "user": "pmm",
        "pwd": "<SECURE_PASSWORD>",  // Replace with a secure password
        "roles": [
            { "db": "admin", "role": "pmmMonitor" },
            { "db": "local", "role": "read" },
            { "db": "admin", "role": "clusterMonitor" }
        ]
    })
    ```
=== "MongoDB <8.0 (With backups)"
    If you intend to use PMM's backup management features, create a user with these additional permissions: 

    ```javascript
    db.getSiblingDB("admin").createUser({
        "user": "pmm",
        "pwd": "<SECURE_PASSWORD>",  // Replace with a secure password
        "roles": [
            { "db": "admin", "role": "pmmMonitor" },
            { "db": "local", "role": "read" },
            { "db": "admin", "role": "readWrite", "collection": "" },
            { "db": "admin", "role": "backup" },
            { "db": "admin", "role": "clusterMonitor" },
            { "db": "admin", "role": "restore" },
            { "db": "admin", "role": "pbmAnyAction" }
        ]
    })      
    ```

## Step 2: Configure query source for MongoDB query analytics

PMM offers two methods for collecting MongoDB queries. Choose based on your environment's requirements and constraints.

### Compare query source methods

| Feature                    | MongoDB Profiler | Diagnostic log          |
|----------------------------|----------------------|---------------------|
| Database connections       | Uses pool continuously | One connection at startup to get log path |
| Connection pool impact     | High              | Minimal             |
| Requires `system.profile`  | Yes               | No               |
| Support remote instances   | Yes               | No                  |
| Supports `mongos`          | No                | Yes              |
| Database overhead          | Moderate-High     | Minimal          |
| File-based logging         | No                | Yes              |
| Query history durability   | Volatile          | Disk-persisted   |
| Scales with DB count       | Linear degradation| Constant         |

=== "MongoDB Profiler (Default)"
    Choose this standard method for simple setups with fewer than 100 databases, remote MongoDB instances, or when you need real-time query collection. 

    The MongoDB Profiler stores query performance data in `system.profile` collections for each database. PMM continuously reads from these collections to provide query analytics.
    
    Key advantages:

    - Real-time query collection and analysis
    - No additional file system access required
    - Works with managed MongoDB services
    - Immediate data availability after profiling is enabled 

    To enable the MongoDB Profiler, choose one of the following methods:

    === "In MongoDB configuration file (Recommended)"

        This method ensures your settings persist across server restarts and system reboots. It's the recommended approach for production environments:
        {.power-number}
        
        1. Edit the configuration file (usually `/etc/mongod.conf`).
        2. Add or modify the `operationProfiling` section in the configuration file. Pay close attention to indentation as YAML is whitespace-sensitive:

            ```yml
            operationProfiling:
                mode: all             
                slowOpThresholdMs: 200
                rateLimit: 100        
            ```
            These settings control the following:

            - `mode: all` - Collects data for all operations.
            - `slowOpThresholdMs: 200` - Marks operations exceeding 200ms as "slow."
            - `rateLimit: 100` -  Limits profiling sampling rate (Percona Server for MongoDB only).
                    
            For more information about profiling configuration options, see the [MongoDB documentation][MONGODB_CONFIG_OP_PROF] and the [Percona Server for MongoDB documentation][PSMDB_RATELIMIT].

        3. Restart the `mongod` service using the appropriate command for your system. For example, for `systemd`:

            ```sh
            systemctl restart mongod
            ```

    === "On CLI"

        Use this method when starting the MongoDB server manually. Keep in mind that smaller values improve accuracy but can adversely affect the performance of your server:

        ```sh
        mongod --dbpath=DATABASEDIR --profile 2 --slowms 200 --rateLimit 100
        ```

        - `--dbpath`: The path to database files (usually `/var/lib/mongo`).
        - `--profile`: The MongoDB profiling level. A value of `2` tells the server to collect profiling data for *all* operations. To lower the load on the server, use a value of `1` to only record slow operations.
        - `--slowms`: An operation is classified as *slow* if it runs for longer than this number of milliseconds.
        - `--rateLimit`: (Only available with Percona Server for MongoDB.) The sample rate of profiled queries. A value of `100` means sample every 100th fast query. ([Read more][PSMDB_RATELIMIT])

    === "In MongoDB shell (temporary)"

        This method enables profiling until the next server restart. Profiling must be enabled for **each** database you want to monitor. For example, to enable the profiler in the `testdb`, run this:

        ```json
        use testdb
        db.setProfilingLevel(2, {slowms: 0})
        ```

    If you have already [added a service](#add-mongodb-service-to-pmm), you should remove it and re-add it after changing the profiling level.   

=== "Diagnostic Log (Recommended for scale)"
     Choose this method for production environments with 100+ databases, when experiencing connection pool issues, or when monitoring mongos routers.

    Available from PMM 3.3.0+, this method reads query data directly from MongoDB's log files instead of querying the database. This eliminates connection pool usage and reduces performance impact.

    Key advantages:

    - Zero database connections required for metrics collection
    - Eliminates connection pool errors completely
    - Scales linearly regardless of database count
    - Identical query analytics data as traditional profiler

    Prerequisites for Diagnostic Log: 

    - MongoDB 5.0+ (tested with 5.0.20-17)
    - Write access to the configured log directory for MongoDB process
    - Read access to log file for PMM Agent user

    To configure mongolog for MongoDB: 
    {.power-number}

    1. Choose one of the following methods to configure MongoDB to log slow operations to the diagnostic log file:

        === "Config file (recommended)"
            Edit your MongoDB configuration file (`mongod.conf`):

            ```yaml
            systemLog:
            destination: file
            path: /var/log/mongodb/mongod.log
            logAppend: true
            logRotate: reopen

            operationProfiling:
            mode: off
            slowOpThresholdMs: 100
            ```

            Configuration explained:

            - `destination: file` - ensures MongoDB logs to a file (required for mongolog)
            - `path` - specifies the log file location that mongolog will read
            - `logAppend: true` - appends to existing log file instead of overwriting
            - `mode: off` - logs operations to file only (does NOT populate system.profile)
            - `slowOpThresholdMs: 100` - set based on your requirements

            Restart MongoDB after making changes:

            ```sh
            systemctl restart mongod
            ```

        === "Command-line flags"
            Start `mongod` with these flags:

            ```bash
            mongod \
            --dbpath /var/lib/mongo \
            --logpath /var/log/mongodb/mongod.log \
            --logappend \
            --profile 0 \
            --slowms 100
            ```

            Flag reference:

            | Flag | Purpose |
            |----------------|--------------------------------------------------------|
            | `--logpath` | Enables logging to a file (required by mongolog) |
            | `--logappend` | Appends to the log file instead of overwriting |
            | `--profile 0` | Enables logging of slow operations (not full profiling) |
            | `--slowms 100` | Sets slow operation threshold (in milliseconds) |

    2. Create a logrotate configuration file (e.g., `/etc/logrotate.d/mongodb`) to configure log rotation:

        ```txt
        /var/log/mongodb/mongod.log {
        daily
        rotate 7
        compress
        delaycompress
        copytruncate
        missingok
        notifempty
        create 640 mongod mongod
        postrotate
            /bin/kill -SIGUSR1 `cat /var/run/mongod.pid 2>/dev/null` >/dev/null 2>&1
        endscript
        }
        ```

        Critical requirements:

        - Use `copytruncate` to preserve file handle for mongolog
        - Avoid moving/renaming log files as this breaks mongolog's file tail
        - Do not delete active log files during rotation
      
## Step 3: Add MongoDB service to PMM

After configuring your database server, add a MongoDB service using either the user interface or the command line.

!!! caution alert alert-warning "Important"
    To monitor MongoDB sharded clusters, PMM requires access to all cluster components. Make sure to add all config servers, all shards, and at least one or two mongos routers. Otherwise, PMM will not be able to correctly collect metrics and populate dashboards.

=== "Via CLI"

    Use `pmm-admin` to add the database server as a service using one of these example commands:

    === "Standalone MongoDB instance"
        ```sh
        pmm-admin add mongodb \
        --username=pmm \
        --password=your_secure_password \
        --host=127.0.0.1 \
        --port=27017 \
        --enable-all-collectors
        ```

    === "Replica Set or Sharded cluster component"
        ```sh
        pmm-admin add mongodb \
        --username=pmm \
        --password=your_secure_password \
        --host=127.0.0.1 \
        --port=27017 \        
        --cluster=my_cluster_name \
        --enable-all-collectors        
        ```

    === "Ignoring insecure server certificate"
        ```sh
        pmm-admin add mongodb \
        --username=pmm \
        --password=your_secure_password \
        --host=127.0.0.1 \
        --port=27017 \        
        --cluster=my_cluster_name \
        --enable-all-collectors \      
        --tls-skip-verify        
        ```     
        
    === "With mongolog query source"
        ```sh
        pmm-admin add mongodb \
        --username=pmm \
        --password=your_secure_password \
        --host=127.0.0.1 \
        --port=27017 \        
        --cluster=my_cluster_name \
        --enable-all-collectors \      
        --query-source=mongolog         
        ```        

    === "SSL/TLS secured MongoDB"
        ```sh
        pmm-admin add mongodb \
        --username=pmm \
        --password=your_secure_password \
        --host=fqdn_of_your_mongo_host \
        --port=27017 \          
        --tls \
        --tls-certificate-key-file=/path/to/client.pem \
        --tls-certificate-key-file-password=cert_password \  # If needed
        --tls-ca-file=/path/to/ca.pem \
        --authentication-mechanism=MONGODB-X509 \
        --authentication-database=$external \
        --cluster=my_cluster_name \
        --enable-all-collectors        
        ```
    
    When successful, PMM Client will print `MongoDB Service added` with the service's ID and name. Use the `--environment` and `--custom-labels` options to set tags for the service to help identify them.

    !!! hint alert alert-success "Tips"
        - When adding nodes to a sharded cluster, ensure to add each node using the same `--cluster mycluster` option. This allows the [MongoDB Cluster Summary](../../../reference/dashboards/dashboard-mongodb-cluster-summary.md) dashboard to populate correctly. 
        - PMM does not gather collection and index metrics if it detects you have more than 200 collections, in order to limit the resource consumption. Check the [advanced options](../../../use/commands/pmm-admin.md#advanced-options) section if you want to modify this behaviour. 
        - When running mongos routers in containers, specify the `diagnosticDataCollectionDirectoryPath` to ensure that pmm-agent can properly capture mongos metrics. For example: `mongos --setParameter diagnosticDataCollectionDirectoryPath=/var/log/mongo/mongos.diagnostic.data/`
        

=== "Via web UI"

    To add a service with the UI:
    {.power-number}

    1. Select **PMM Configuration > Add Service > MongoDB**.

    2. Fill in the required fields.

    3. Click **Add service**.

    ![!](../../../images/PMM_Add_Instance_MongoDB.jpg)

## Step 4: Verify MongoDB service configuration

After adding MongoDB service to PMM, verify that it's properly configured and collecting data. This ensures your monitoring setup is working correctly.
{.power-number}

1. Check service registration:

    === "Via command line"
        Look for your service in the output of this command:

        ```sh
        pmm-admin list
        ```

        For mongolog specifically, verify with:
        ```sh
        pmm-admin status
        ```
        Look for `mongodb_mongolog_agent` - it should show the agent is running with mongolog as the query source.

    === "Via web UI"
        To check the service from the UI:
        {.power-number}

        1. Select **PMM Configuration > Inventory > Services**. 
        2. Find your MongoDB service in the list and verify it shows **Active** status.
        3. Verify the **Service name**, **Addresses**, and other connection details are correct.
        4. In the **Options** column, expand the **Details** section to check that agents are properly connected.

2. Verify data collection:

    - On the **MongoDB Instances Overview** dashboard
    - Set the **Service Name** to the newly-added service
    - Confirm that metrics are being displayed in the dashboard

3. Verify Query Analytics for the service:

    - Open the **PMM Query Analytics** dashboard and use the filters to select your MongoDB service. 
    - Check that query data is visible (it may take a few minutes for data to appear after initial setup).
    - Performance impact is virtually zero since metrics are sourced from existing log files (for mongolog) or real-time profiler data.

## Remove MongoDB service

If you need to remove MongoDB service from PMM, follow these steps:

=== "Via command line"
    Replace `SERVICE_NAME` with the name you used when adding the service. You can list all services with `pmm-admin`:

    ```sh
    pmm-admin remove mongodb SERVICE_NAME
    ```

=== "Via web UI"
    To remove the services through the PMM interface:
    {.power-number}

    1. Go to **PMM Configuration > Inventory > Services**.
    2. In the **Status** column, check the box for the service you want to remove and click **Delete**.
    3. On the confirmation pop-up, click **Delete service** and select **Force mode** if you want to also delete associated Clients.

## Related topics

- [`pmm-admin add mongodb`](../../../use/commands/pmm-admin.md#database-commands)
- [Troubleshooting connection difficulties]

[MongoDB]: https://www.mongodb.com/
[Percona Server for MongoDB]: https://www.percona.com/software/mongodb/percona-server-for-mongodb
[profiling feature]: https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/
[YAML]: http://yaml.org/spec/
[MONGODB_CONFIG_OP_PROF]: https://docs.mongodb.com/manual/reference/configuration-options/#operationprofiling-options
[PSMDB_RATELIMIT]: https://www.percona.com/doc/percona-server-for-mongodb/LATEST/rate-limit.html#enabling-the-rate-limit
[PMM_ADMIN_MAN_PAGE]: ../../../use/commands/pmm-admin.md#database-commands
[Troubleshooting connection difficulties]: ../../../troubleshoot/config_issues.md#connection-difficulties
