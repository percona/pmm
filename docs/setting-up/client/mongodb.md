# MongoDB

## Configuring MongoDB for Monitoring in PMM Query Analytics

In Query Analytics, you can monitor MongoDB metrics and queries. Run the `pmm-admin add` command to use these monitoring services.

**Supported versions of MongoDB**

Query Analytics supports MongoDB version 3.2 or higher.

## Setting Up the Required Permissions

For MongoDB monitoring services to work in Query Analytics, you need to set up the `mongodb_exporter` user.

Here is an example for the MongoDB shell that creates and assigns the appropriate roles to the user.

```js
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

## Enabling Profiling

For [MongoDB](https://www.mongodb.com) to work correctly with Query Analytics, you need to enable profiling in your `mongod` configuration. (Profiling is not enabled by default because it may reduce the performance of your MongoDB server.)

### Enabling Profiling on Command Line

You can enable profiling from command line when you start the `mongod`
server. This command is useful if you start `mongod` manually.

Run this command as root or by using the `sudo` command

```sh
mongod --dbpath=DATABASEDIR --profile 2 --slowms 200 --rateLimit 100
```

Note that you need to specify a path to an existing directory that stores
database files with the `--dpbath`. When the `--profile` option is set to
2, `mongod` collects the profiling data for all operations. To decrease the
load, you may consider setting this option to 1 so that the profiling data
are only collected for slow operations.

The `--slowms` option sets the minimum time for a slow operation. In the
given example, any operation which takes longer than 200 milliseconds is a
slow operation.

The `--rateLimit` option, which is available if you use PSMDB instead
of MongoDB, refers to the number of queries that the MongoDB profiler
collects. The lower the rate limit, the less impact on the performance.
However, the accuracy of the collected information decreases as well.

### Enabling Profiling in the Configuration File

If you run `mongod` as a service, you need to use the configuration file
which by default is `/etc/mongod.conf`.

In this file, you need to locate the `operationProfiling:` section and add the
following settings:

```
operationProfiling:
   slowOpThresholdMs: 200
   mode: slowOp
```

These settings affect `mongod` in the same way as the command line options. Note that the configuration file is in the [YAML](http://yaml.org/spec/) format. In this format the indentation of your lines is important as it defines levels of nesting.

Restart the `mongod` service to enable the settings.

Run this command as root or by using the `sudo` command

```sh
service mongod restart
```

## Adding MongoDB Service Monitoring

Add monitoring as follows:

```sh
pmm-admin add mongodb --username=pmm --password=pmm
```

where username and password are credentials for the monitored MongoDB access, which will be used locally on the database host. Additionally, two positional arguments can be appended to the command line flags: a service name to be used by PMM, and a service address. If not specified, they are substituted automatically as `<node>-mongodb` and `127.0.0.1:27017`.

The command line and the output of this command may look as follows:

```sh
pmm-admin add mongodb --username=pmm --password=pmm mongo 127.0.0.1:27017
```

```
MongoDB Service added.
Service ID  : /service_id/f1af8a88-5a95-4bf1-a646-0101f8a20791
Service name: mongo
```

Beside positional arguments shown above you can specify service name and service address with the following flags: `--service-name`, `--host` (the hostname or IP address of the service), and `--port` (the port number of the service). If both flag and positional argument are present, flag gains higher priority. Here is the previous example modified to use these flags:

```sh
pmm-admin add mongodb --username=pmm --password=pmm --service-name=mongo --host=127.0.0.1 --port=27017
```

> You can add a MongoDB instance using a UNIX socket with the `--socket` option:
>
> ```sh
> pmm-admin add mongodb --socket=/tmp/mongodb-27017.sock
> ```

## Passing SSL parameters to the MongoDB monitoring service

SSL/TLS related parameters are passed to an SSL enabled MongoDB server as
monitoring service parameters along with the `pmm-admin add` command when adding
the MongoDB monitoring service.

Run this command as root or by using the `sudo` command

```sh
pmm-admin add mongodb --tls
```

**Supported SSL/TLS Parameters**

`--tls`
: Enable a TLS connection with mongo server

`--tls-skip-verify`
: Skip TLS certificates validation

`--tls-certificate-key-file=PATHTOCERT`
: Path to TLS certificate file.

`--tls-certificate-key-file-password=IFPASSWORDTOCERTISSET`
: Password for TLS certificate file.

`--tls-ca-file=PATHTOCACERT`
: Path to certificate authority file.
