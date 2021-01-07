# Configuring MongoDB for Monitoring in PMM Query Analytics

In QAN, you can monitor MongoDB metrics and MongoDB queries with the `mongodb:metrics` or `mongodb:queries` monitoring services accordingly. Run the **pmm-admin add** command to use these monitoring services (for more information, see [Adding monitoring services](pmm-admin.md#pmm-admin-add)).

## Supported versions of MongoDB

QAN supports MongoDB version 3.2 or higher.

## Setting Up the Essential Permissions

For `mongodb:metrics` and `mongodb:queries` monitoring services to be able work in QAN, you need to set up the **mongodb_exporter** user. This user should be assigned the *clusterMonitor* role for the `admin` database and the *read* role for the `local` database.

The following example that you can run in the MongoDB shell, adds the **mongodb_exporter** user and assigns the appropriate roles.

```
db.createRole({
    role: "explainRole",
    privileges: [{
        resource: {
            db: "",
            collection: ""
            },
        actions: [
            "listIndexes",
            "listCollections",
            "dbStats",
            "dbHash",
            "collStats",
            "find"
            ]
        }],
    roles:[]
})

db.getSiblingDB("admin").createUser({
   user: "mongodb_exporter",
   pwd: "s3cR#tpa$$worD",
   roles: [
      { role: "explainRole", db: "admin" },
      { role: "clusterMonitor", db: "admin" },
      { role: "read", db: "local" }
   ]
})
```

Then, you need to pass the user name and password in the value of the `--uri` option when adding the `mongodb:metrics` monitoring service in the **pmm-admin add** command:

Run this command as root or by using the **sudo** command.

<!-- code=block:: bash

$ pmm-admin add mongodb-metrics --uri mongodb://mongodb_exporter:s3cR#tpa$$worD@localhost:27017 -->


## Enabling Profiling

For [MongoDB](https://www.mongodb.com) to work correctly with QAN, you need to enable profiling in your **mongod** configuration. When started without profiling enabled, QAN displays the following warning:

**NOTE**: **A warning message is displayed when profiling is not enabled**

It is required that profiling of the monitored MongoDB databases be enabled.

Note that profiling is not enabled by default because it may reduce the performance of your MongoDB server.

### Enabling Profiling on Command Line

You can enable profiling from command line when you start the **mongod** server. This command is useful if you start **mongod** manually.

Run this command as root or by using the **sudo** command

```
$ mongod --dbpath=DATABASEDIR --profile 2 --slowms 200 --rateLimit 100
```

Note that you need to specify a path to an existing directory that stores database files with the `--dpbath`. When the `--profile` option is set to **2**, **mongod** collects the profiling data for all operations. To decrease the load, you may consider setting this option to **1** so that the profiling data are only collected for slow operations.

The `--slowms` option sets the minimum time for a slow operation. In the given example, any operation which takes longer than **200** milliseconds is a slow operation.

The `--rateLimit` option, which is available if you use PSMDB instead of MongoDB, refers to the number of queries that the MongoDB profiler collects. The lower the rate limit, the less impact on the performance. However, the accuracy of the collected information decreases as well.

### Enabling Profiling in the Configuration File

If you run `mongod` as a service, you need to use the configuration file which by default is `/etc/mongod.conf`.

In this file, you need to locate the *operationProfiling:* section and add the following settings:

```
operationProfiling:
   slowOpThresholdMs: 200
   mode: slowOp
   rateLimit: 100
```

These settings affect `mongod` in the same way as the command line options described in section Enabling Profiling on Command Line. Note that the configuration file is in the [YAML](http://yaml.org/spec/) format. In this format the indentation of your lines is important as it defines levels of nesting.

Restart the *mongod* service to enable the settings.

Run this command as root or by using the **sudo** command

```
$ service mongod restart
```
