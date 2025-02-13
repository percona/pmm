# Get started with PMM

To get up and running with Percona Monitoring and Management (PMM) in no time, install PMM on Bare Metal/Virtual using the Easy-install script for Docker.

This is the simplest and most efficient way to install PMM with Docker.

??? info "Alternative installation options"
     For alternative setups or if you're not using Docker, explore the additional installation options detailed in the **Setting up** chapter:

    - [Deploy on Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md)
    - [Deploy based on a Docker image](../install-pmm/install-pmm-server/deployment-options/docker/index.md)
    - [Deploy on Virtual Appliance](../install-pmm/install-pmm-server/deployment-options/virtual/index.md)
    - [Deploy on Kubernetes via Helm](../install-pmm/install-pmm-server/deployment-options/helm/index.md)
    - [Run a PMM instance hosted at AWS Marketplace](../install-pmm/install-pmm-server/aws/aws.md)

#### Prerequisites

Before you start installing PMM, verify that your system meets the compatibility requirements.

??? info "Verify system compatibility"
    - **Disk**: Approximately 1 GB of storage per monitored database node with data retention set to one week. By default, retention is 30 days.
    - **Memory**: A minimum of 2 GB per monitored database node. The increase in memory usage is not proportional to the number of nodes. For example, the data from 20 nodes should be easily handled with 16 GB.
    - **Ports**: Your system’s firewall should allow TCP traffic on port 443.

## Install PMM

The Easy-install script only runs on Linux-compatible systems. To use it, run the command with `sudo` privileges or as `root`:
{ .power-number }

1. Download and install PMM using `cURL` or `wget`:

    === "cURL"

        ```sh
        curl -fsSL https://raw.githubusercontent.com/percona/pmm/refs/heads/v3/get-pmm.sh | /bin/bash
        ```

    === "wget"

        ```sh
        wget -qO - https://raw.githubusercontent.com/percona/pmm/refs/heads/v3/get-pmm.sh | /bin/bash    
        ```

2. After the installation is complete, log into PMM with the default `admin:admin` credentials.

??? info "What's happening under the hood?"
     This script does the following:

     * Installs Docker if it is not installed on your system.
     * Stops and renames any currently running PMM Docker container from `pmm-server` to `pmm-server-{timestamp}`. This old `pmm-server` container is not a recoverable backup.
     * Pulls and runs the latest PMM Docker image.

## Connect database

Once PMM is set up, choose the database or the application that you want it to monitor:

=== ":simple-mysql: MySQL"

    To connect a self-hosted MySQL database:
    { .power-number}

    1. Create database account for PMM using the following command example. This creates a database user with name `pmm`, password `<your_password>`, and the necessary permissions:

        ```sql
        CREATE USER 'pmm'@'127.0.0.1' IDENTIFIED BY '<your_password>' WITH MAX_USER_CONNECTIONS 10;
        GRANT SELECT, PROCESS, REPLICATION CLIENT, RELOAD, BACKUP_ADMIN ON *.* TO 'pmm'@'127.0.0.1';
        ```

    2. To optimize server-side resources, install PMM Client via Package Manager on the database node:    
        
        === ":material-debian: Debian-based"

            Install the following with `root` permission:
            { .power-number} 

            1. Install the [Percona Release Tool](https://docs.percona.com/percona-software-repositories/installing.html).  If this is already, make sure to update it to the latest version:

                ```sh
                wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
                dpkg -i percona-release_latest.generic_all.deb
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                apt update
                apt install -y pmm-client
                ```

        === ":material-redhat: Red Hat-based"

            Install the following with `root` permission:
            { .power-number} 

            4. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool. If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version.

                ```sh
                yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
                ```

            5. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            6. Install the PMM Client package:

                ```sh
                yum install -y pmm-client
                ```

    3. Register PMM Client:
        
        ```sh
        pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
        ```

    4. Add the MySQL database using Performance Schema:  

        ```sh 
        pmm-admin add mysql --query-source=perfschema --username=pmm --password=<your_password>
        ```
    ??? info "Alternative database connection workflows"
        While the default instructions above focus on connecting a self-hosted MySQL database, PMM offers the flexibility to connect to various MySQL databases, including [AWS RDS](../install-pmm/install-pmm-client/connect-database/aws.md), [Azure MySQL](../install-pmm/install-pmm-client/connect-database/azure.md) or [Google Cloud MySQL](../install-pmm/install-pmm-client/connect-database/google.md). 

        The PMM Client installation also comes with options: in addition to the installation via Package Manager described above, you can also install it as a Docker container or as a binary package. Explore [alternative PMM Client installation options](../install-pmm/install-pmm-client/connect-database/index.html#binary-package) for more information.

        Additionally, if direct access to the database node isn't available, opt to [Add remote instance via User Interface](../install-pmm/install-pmm-client/connect-database/mysql.html#with-the-user-interface) instead. 

=== ":simple-postgresql: PostgreSQL"

    To connect a PostgreSQL database: 
    { .power-number}

    1. Create a PMM-specific user for monitoring:
        
        ```
        CREATE USER pmm WITH SUPERUSER ENCRYPTED PASSWORD '<your_password>';
        ```

    2. Ensure that PMM can log in locally as this user to the PostgreSQL instance. To enable this, edit the `pg_hba.conf` file. If  not already enabled by an existing rule, add:

        ```conf
        local   all             pmm                                md5
        # TYPE  DATABASE        USER        ADDRESS                METHOD
        ```

    3. Set up the `pg_stat_monitor` database extension and configure your database server accordingly. 
    
        If you need to use the `pg_stat_statements` extension instead, see [Adding a PostgreSQL database](../install-pmm/install-pmm-client/connect-database/postgresql.md) and the [`pg_stat_monitor` online documentation](https://docs.percona.com/pg-stat-monitor/configuration.html) for details about available parameters.

    4. Set or change the value for `shared_preload_library` in your `postgresql.conf` file:

        ```ini
        shared_preload_libraries = 'pg_stat_monitor'
        ```

    5. Set up configuration values in your `postgresql.conf` file:

        ```ini
        pg_stat_monitor.pgsm_query_max_len = 2048
        ```

    6. In a `psql` session, run the following command to create the view where you can access the collected statistics. We recommend that you create the extension for the `postgres` database so that you can receive access to statistics from each database.

        ```
        CREATE EXTENSION pg_stat_monitor;
        ```

    7. To optimize server-side resources, install PMM Client via Package Manager on the database node:  
        
        === ":material-debian: Debian-based"

            Install the following with `root` permission: 
            { .power-number} 

            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
                dpkg -i percona-release_latest.generic_all.deb
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                apt update
                apt install -y pmm-client
                ```

        === ":material-redhat: Red Hat-based"

            Install the following with `root` permission: 
            { .power-number}   

            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
                ```
            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                yum install -y pmm-client
                ```

    8. Register PMM Client:
        
        ```sh
        pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
        ```

    9. Add the PostgreSQL database:

        ```sh 
        pmm-admin add postgresql --username=pmm --password=<your_password>
        ```
            
    For detailed instructions and advanced installation options, see [Adding a PostgreSQL database](../install-pmm/install-pmm-client/connect-database/postgresql.md).

=== ":simple-mongodb: MongoDB"

    To connect a MongoDB database:
    { .power-number}
    
    1.  Run the following command in `mongo` shell to create a role with the monitoring permissions: 
 
        ```
        db.createRole({
            "role":"explainRole",
            "privileges":[
                {
                    "resource":{
                        "db":"",
                        "collection":""
                    },
                    "actions":[
                        "collStats",
                        "dbHash",
                        "dbStats",
                        "find",
                        "listIndexes",
                        "listCollections"
                    ]
                }
            ],
            "roles":[]
        })
        ```

    2. Create a user and grant it the role created above:

        ```
        db.getSiblingDB("admin").createUser({
            "user":"pmm",
            "pwd":"<your_password>",
            "roles":[
                {
                    "role":"explainRole",
                    "db":"admin"
                },
                {
                    "role":"clusterMonitor",
                    "db":"admin"
                },
                {
                    "role":"read",
                    "db":"local"
                }
            ]
        })
        ```

    3. To optimize server-side resources, install PMM Client via Package Manager on the database node:
        { .power-number}     
        
        === ":material-debian: Debian-based"

            Install the following with `root` permission: 
                         
            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
                dpkg -i percona-release_latest.generic_all.deb
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                apt update
                apt install -y pmm-client
                ```

        === ":material-redhat: Red Hat-based"

            Install the following with `root` permission: 

            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                yum install -y pmm-client
                ```

    4. Register PMM Client:
        
        ```sh
        pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
        ```

    5. Add the MongoDB database:

        ```
        pmm-admin add mongodb --username=pmm --password=<your_password>
        ```
   
    For detailed instructions, see [Adding a MongoDB database for monitoring](../install-pmm/install-pmm-client/connect-database/mongodb.html).

=== ":simple-nginxproxymanager: ProxySQL"
    To connect a ProxySQL service:
    { .power-number}

    1. Configure a read-only account for monitoring using the [`admin-stats_credentials`](https://proxysql.com/documentation/global-variables/admin-variables/#admin-stats_credentials) variable in ProxySQL.

    2. To optimize server-side resources, install PMM Client via Package Manager on the database node:
        { .power-number}     
        
        === ":material-debian: Debian-based"
            Install the following with `root` permission: 
            { .power-number} 

            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
                dpkg -i percona-release_latest.generic_all.deb
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                apt update
                apt install -y pmm-client
                ```

        === ":material-redhat: Red Hat-based"
            Install the following with `root` permission: 
            { .power-number}      

            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                yum install -y pmm-client
                ```

    3. Register PMM Client:
        
        ```sh
        pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
        ```
            
    4. Add the ProxySQL service:

        ```
        pmm-admin add proxysql --username=pmm --password=<your_password>
        ```

    For detailed instructions, see [Enable ProxySQL performance metrics monitoring](../install-pmm/install-pmm-client/connect-database/proxysql.md).

=== ":material-database: HAProxy"
    To connect an HAProxy service:
    { .power-number}

    1. [Set up an HAproxy instance](https://www.haproxy.com/blog/haproxy-exposes-a-prometheus-metrics-endpoint). 
    2. Add the instance to PMM (default address is <http://localhost:8404/metrics>), and use the `haproxy` alias to enable HAProxy metrics monitoring.
    3. To optimize server-side resources, install PMM Client via Package Manager on the database node: 
        
        === ":material-debian: Debian-based"
            Install the following with `root` permission: 
            { .power-number} 
                                     
            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
                dpkg -i percona-release_latest.generic_all.deb
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                apt update
                apt install -y pmm-client
                ```

        === ":material-redhat: Red Hat-based"
            Install the following with `root` permission: 
            { .power-number} 
         
            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                yum install -y pmm-client
                ```

    4. Register PMM Client:
        
        ```sh
        pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
        ```

    5. Run the command below, specifying the `listen-port`` as the port number where HAProxy is running. (This flag is mandatory.)

        ```sh
        pmm-admin add haproxy --listen-port=8404
        ```

    For detailed instructions and more information on the command arguments, see the [HAProxy topic](../install-pmm/install-pmm-client/connect-database/haproxy.md).

## Check database monitoring results

After installing PMM and connecting the database, go to the database's Instance Summary dashboard. This shows essential information about your database performance and an overview of your environment.

For more information, see [PMM Dashboards](../use/dashboards-panels/index.md).

## Next steps

- [Configure PMM via the interface](../configure-pmm/configure.md)
- [Manage users in PMM](../how-to/manage-users.md)
- [Set up roles and permissions](../pmm-admin/roles/index.md)
- [Back up and restore data in PMM](../backup/index.md)