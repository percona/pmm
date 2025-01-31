# Install PMM in High Availability (HA) mode

High Availability (HA) is a critical feature for monitoring systems, ensuring resilience and seamless operation even when one or more instances encounter issues. HA minimizes downtime and maintains continuous visibility into the performance and health of your Percona Monitoring and Management (PMM) infrastructure.

This guide explains available HA options for PMM and provides step-by-step instructions for setting up PMM in HA mode.

## HA options PMM

Since HA can add complexity, before considering HA for PMM, keep in mind that:

- Critical systems requiring immediate response benefit from sub-second failover HA, while less critical applications with some tolerance for downtime (seconds or minutes) have more flexibility.

- PMM itself has a one-minute minimum alerting interval, so even with perfect HA, the fastest you'll know about an issue is one minute after it occurs.

- Consider your specific uptime needs, performance requirements, and potential data loss you can tolerate, while also keeping in mind PMM's limitations.

### 1. Simple Docker restart with data caching

The most straightforward approach to increase availability in PMM is to launch the PMM Server within Docker using the `--restart=always` flag. See [Install PMM Server with Docker](../install-pmm/install-pmm-server/baremetal/docker/index.md) for more information.

This ensures that the PMM Server automatically restarts if a minor issue occurs. Additionally, PMM's data caching feature stores data locally on the PMM Client when the connection to the PMM Server is interrupted.

Once the connection is restored, the cached data is transferred to the PMM Server, ensuring no data loss during the restart process.

This option is suitable for scenarios where the primary concern is the ability to investigate potential issues later. However, it's important to note that this approach is limited by the underlying physical infrastructure. If the failure stems from a hardware issue, automatic recovery might be challenging.

### 2. Leverage Kubernetes for enhanced isolation

If you are running PMM in a Kubernetes (K8s) environment, PMM offers a Helm chart that facilitates running PMM with enhanced isolation. See [Install PMM Server with Docker](../install-pmm/install-pmm-server/baremetal/docker/index.md).

In this setup, even if the physical infrastructure encounters a problem, K8s automatically handles failover, migrating the PMM instance to a healthy node. 

While restarts within K8s can take up to several minutes (depending on your infrastructure configuration), PMM's data caching ensures that information is preserved during this transition. Alerts will still be triggered to keep you informed about any issues that started during PMM's restart and continue after PMM is back.

### 3. Fully-clustered PMM in Kubernetes (in development)

If you have a large deployment with numerous instances and distributed locations, you might find that a fully clustered PMM setup in Kubernetes is better suited to your needs. We are actively developing this solution, which is slated for release later with PMM 3.x, to cater specifically to users managing extensive and complex monitoring environments.

This option will provide a comprehensive HA solution, including clustered database setups (ClickHouse, VictoriaMetrics, and PostgreSQL). In this setup, multiple PMM instances will be configured, with one being the leader and the others as followers.

Leader election will be managed using the Raft consensus algorithm, ensuring a smooth transition of the leader role if the current leader fails. The architecture will consist of:

- multiple PMM instances for redundancy
- clustered PostgreSQL for storing metadata and configuration data
- clustered ClickHouse for storing query performance metrics (Query Analytics)
- clustered VictoriaMetrics for storing operational metrics from monitored databases and hosts
- HAProxy for managing and directing network traffic to the current leader PMM instance

### 4. Manual HA setup

!!! caution alert alert-warning "Important"
    Manual setup for HA is feature is currently in [Technical Preview](https://docs.percona.com/percona-monitoring-and-management/details/glossary.html#technical-preview). Early adopters are advised to use this feature for testing purposes only as it is subject to change.

If none of the above options work for your specific use case, consider setting up PMM in HA mode manually by following the steps below.

To enable communication and coordination among the PMM Server instances, two key protocols are used:

- **Gossip protocol**: Enables PMM servers to discover and share information about their states. It is used for managing the PMM server list and failure detection, ensuring that all instances are aware of the current state of the cluster.
- **Raft protocol**: Ensures that PMM servers agree on a leader and that logs are replicated among all machines to maintain data consistency.

These protocols work in tandem to ensure that the PMM Server instances can effectively store and manage the data collected from your monitored databases and systems. 

In an HA configuration, three PMM Server instances are configured: one as the leader and the others as followers. The leader server handles all client requests. If the leader fails, the followers take over, minimizing downtime.

To eliminate single points of failure and provide better service level agreements (SLAs), the critical services typically bundled with PMM Server are extracted and set up as separate, clustered instances:

- ClickHouse: A clustered setup of ClickHouse is used to store Query Analytics (QAN) metrics. This ensures that QAN data remains highly available and can be accessed even if one of the ClickHouse nodes fails.
- VictoriaMetrics: A clustered setup of VictoriaMetrics is used to store Prometheus metrics. This provides a highly available and scalable solution for storing and querying metrics data.
- PostgreSQL: A clustered setup of PostgreSQL is used to store PMM data, such as inventory and settings. This ensures that PMM's configuration and metadata remain highly available and can be accessed by all PMM Server instances.

#### Prerequisites
Before you begin:

- [Install and configure Docker](https://docs.docker.com/get-docker/).
- Prepare your environment:
    - for testing, you can run services on a single machine
    - for production, deploy services on separate instances and use clustered versions of PostgreSQL, VictoriaMetrics, and ClickHouse. Keep in mind that running all services on a single machine is not recommended for production. Use separate instances and clustered components for better reliability.

To set up PMM in HA mode manually:

#### **Step 1: Define environment variables**

Before you start with the setup, define the necessary environment variables on each instance where the services will be running. These variables will be used in subsequent commands. 

For all IP addresses, use the format `17.10.1.x`, and for all usernames and passwords, use a string format like `example`.

| **Variable**        | **Description**
| ------------------------------------------------| -------------------------------------------------------------------------------------------------------------------------------
| `CH_HOST_IP`                                     | The IP address of the instance where the ClickHouse service is running or the desired IP address for the ClickHouse container within the Docker network, depending on your setup.</br></br>Example: `17.10.1.2`
| `VM_HOST_IP`                                     | The IP address of the instance where the VictoriaMetrics service is running or the desired IP address for the VictoriaMetrics container within the Docker network, depending on your setup.</br></br>Example: `17.10.1.3`
| `PG_HOST_IP`                                     | The IP address of the instance where the PostgreSQL service is running or the desired IP address for the PostgreSQL container within the Docker network, depending on your setup.</br></br> Example: `17.10.1.4`
| `PG_USERNAME`                                    | The username for your PostgreSQL server.</br></br> Example: `pmmuser`
| `PG_PASSWORD`                                   | The password for your PostgreSQL server. </br></br>Example: `pgpassword`
| `GF_USERNAME`                                   | The username for your Grafana database user.</br></br>Example: `gfuser`
| `GF_PASSWORD`                                   | The password for your Grafana database user.</br></br>Example: `gfpassword`
| `PMM_ACTIVE_IP`                                 | The IP address of the instance where the active PMM server is running or the desired IP address for your active PMM server container within the Docker network, depending on your setup.</br></br>Example: `17.10.1.5`
| `PMM_ACTIVE_NODE_ID`                            | The unique ID for your active PMM server node.</br></br>Example: `pmm-server-active`
| `PMM_PASSIVE_IP`                                   | The IP address of the instance where the first passive PMM server is running or the desired IP address for your first passive PMM server container within the Docker network, depending on your setup. </br></br>Example: `17.10.1.6`
| `PMM_PASSIVE_NODE_ID`                                  | The unique ID for your first passive PMM server node.</br></br>Example: `pmm-server-passive`
| `PMM_PASSIVE2_IP`                                         | The IP address of the instance where the second passive PMM server is running or the desired IP address for your second passive PMM server container within the Docker network, depending on your setup.</br></br>Example: `17.10.1.7`
| `PMM_PASSIVE2_NODE_ID`                                    | The unique ID for your second passive PMM server node.</br></br>Example: `pmm-server-passive2`
| `PMM_DOCKER_IMAGE` &nbsp; &nbsp; &nbsp; &nbsp;                                      | The specific PMM Server Docker image for this guide.</br></br>Example: `percona/pmm-server:3`

??? example "Expected output"
        
    ```
    export CH_HOST_IP=17.10.1.2
    export VM_HOST_IP=17.10.1.3
    export PG_HOST_IP=17.10.1.4
    export PG_USERNAME=pmmuser
    export PG_PASSWORD=pgpassword
    export GF_USERNAME=gfuser
    export GF_PASSWORD=gfpassword
    export PMM_ACTIVE_IP=17.10.1.5
    export PMM_ACTIVE_NODE_ID=pmm-server-active
    export PMM_PASSIVE_IP=17.10.1.6
    export PMM_PASSIVE_NODE_ID=pmm-server-passive
    export PMM_PASSIVE2_IP=17.10.1.7
    export PMM_PASSIVE2_NODE_ID=pmm-server-passive2
    export PMM_DOCKER_IMAGE=percona/pmm-server:3
    ```

!!! note alert alert-primary "Note"
    Ensure that you have all the environment variables from Step 1 set in each instance where you run these commands.

#### **Step 2: Create Docker network (Optional)**
Create a dedicated network if you plan to run multiple PMM services on the same instance. This ensures proper communication between containers, especially for High Availability mode.
{.power-number}

1. Set up a Docker network for PMM services if you plan to run all the services on the same instance. As a result of this Docker network, your containers will be able to communicate with each other, which is essential for the High Availability (HA) mode to function properly in PMM. This step may be optional if you run your services on separate instances.

2. Run the following command to create a Docker network:

    ```sh
    docker network create pmm-network --subnet=17.10.1.0/16
    ```

#### **Step 3: Set up ClickHouse**

ClickHouse is an open-source column-oriented database management system. In PMM, ClickHouse stores Query Analytics (QAN) metrics, which provide detailed information about your queries.

To set up ClickHouse:
{.power-number}

1. Pull the ClickHouse Docker image.

    ```sh
    docker pull clickhouse/clickhouse-server:23.8.2.7-alpine
    ```

2. Create a Docker volume for ClickHouse data.

    ```sh
    docker volume create ch_data
    ```

3. Run the ClickHouse container.

    === "Run services on same instance"

        ```sh
        docker run -d \
        --name ch \
        --network pmm-network \
        --ip ${CH_HOST_IP} \
        -p 9000:9000 \
        -v ch_data:/var/lib/clickhouse \
        clickhouse/clickhouse-server:23.8.2.7-alpine
        ```
    
    === "Run services on a seperate instance"

        ```sh
        docker run -d \
        --name ch \
        -p 9000:9000 \
        -v ch_data:/var/lib/clickhouse \
        clickhouse/clickhouse-server:23.8.2.7-alpine
        ```

    !!! note alert alert-primary "Note"
        - If you run the services on the same instance, the `--network` and `--ip` flags assign a specific IP address to the container within the Docker network created in the previous step. This IP address is referenced in subsequent steps as the ClickHouse service address. 
        - The `--network` and `--ip` flags are not required if the services are running on separate instances since ClickHouse will bind to the default network interface.

#### **Step 4: Set up VictoriaMetrics**

VictoriaMetrics provides a long-term storage solution for your time-series data. In PMM, it is used to store Prometheus metrics.

To set up VictoriaMetrics:
{.power-number}

1. Pull the VictoriaMetrics Docker image:

    ```sh
    docker pull victoriametrics/victoria-metrics:v1.93.4
    ```

2. Create a Docker volume for VictoriaMetrics data:

    ```sh
    docker volume create vm_data
    ```

3. Run the VictoriaMetrics container:

    You can either run all the services on the same instance or a separate instance.

    
    === "Run services on same instance"

        ```sh
        docker run -d \
        --name vm \
        --network pmm-network \
        --ip ${VM_HOST_IP} \
        -p 8428:8428 \
        -p 8089:8089 \
        -p 8089:8089/udp \
        -p 2003:2003 \
        -p 2003:2003/udp \
        -p 4242:4242 \
        -v vm_data:/storage \
        victoriametrics/victoria-metrics:v1.93.4 \
        --storageDataPath=/storage \
        --graphiteListenAddr=:2003 \
        --opentsdbListenAddr=:4242 \
        --httpListenAddr=:8428 \
        --influxListenAddr=:8089
        ```
    
    === "Run services on a seperate instance"

        ```sh
        docker run -d \
        --name vm \
        -p 8428:8428 \
        -p 8089:8089 \
        -p 8089:8089/udp \
        -p 2003:2003 \
        -p 2003:2003/udp \
        -p 4242:4242 \
        -v vm_data:/storage \
        victoriametrics/victoria-metrics:v1.93.4 \
        --storageDataPath=/storage \
        --graphiteListenAddr=:2003 \
        --opentsdbListenAddr=:4242 \
        --httpListenAddr=:8428 \
        --influxListenAddr=:8089
        ```

    !!! note alert alert-primary "Note"
        - If you run the services on the same instance,  the `--network` and `--ip` flags are used to assign a specific IP address to the container within the Docker network created in Step 2. This IP address is referenced in subsequent steps as the VictoriaMetrics service address. 
        - The `--network` and `--ip` flags are not required if the services are running on separate instances, as VictoriaMetrics will bind to the default network interface.

#### **Step 5: Set up PostgreSQL**

PostgreSQL is a powerful, open-source object-relational database system. In PMM, it's used to store data related to inventory, settings, and other features.

To set up PostgreSQL:
{.power-number}

1. Pull the Postgres Docker image:

    ```sh
    docker pull postgres:14
    ```
    
2. Create a Docker volume for Postgres data:
    
    ```bash
    docker volume create pg_data
    ```
    
3. Create a directory to store init SQL queries:
    
    ```bash
    mkdir -p /path/to/queries
    ```
    
    Replace `/path/to/queries` with the path where you want to store your `init` SQL queries.
    
4. Create an `init.sql.template` file in newly created directory with the following content:
    
    ```sql
    CREATE DATABASE "pmm-managed";
    CREATE USER <YOUR_PG_USERNAME> WITH ENCRYPTED PASSWORD '<YOUR_PG_PASSWORD>';
    GRANT ALL PRIVILEGES ON DATABASE "pmm-managed" TO <YOUR_PG_USERNAME>;
    CREATE DATABASE grafana;
    CREATE USER <YOUR_GF_USERNAME> WITH ENCRYPTED PASSWORD '<YOUR_GF_PASSWORD>';
    GRANT ALL PRIVILEGES ON DATABASE grafana TO <YOUR_GF_USERNAME>;
    
    \c pmm-managed
    
    CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
    ```

5. Use **`sed`** to replace the placeholders with the environment variables and write the output to **`init.sql`**:
    
    ```bash
    sed -e 's/<YOUR_PG_USERNAME>/'"$PG_USERNAME"'/g' \
        -e 's/<YOUR_PG_PASSWORD>/'"$PG_PASSWORD"'/g' \
        -e 's/<YOUR_GF_USERNAME>/'"$GF_USERNAME"'/g' \
        -e 's/<YOUR_GF_PASSWORD>/'"$GF_PASSWORD"'/g' \
        init.sql.template > init.sql
    ```
    
6. Run the PostgreSQL container:

    You can either run all the services on the same instance or on a seperate instance.

    !!! note alert alert-primary "Note"
        It is recommended to use absolute paths instead of relative paths for volume mounts.


    === "Run services on same instance"

        ```sh
          docker run -d \
            --name pg \
            --network pmm-network \
            --ip ${PG_HOST_IP} \
            -p 5432:5432 \
            -e POSTGRES_PASSWORD=${PG_PASSWORD} \
            -v /path/to/queries:/docker-entrypoint-initdb.d/ \
            -v pg_data:/var/lib/postgresql/data \
            postgres:14 \
            postgres -c shared_preload_libraries=pg_stat_statements
        ```
        
    === "Run services on a seperate instance"
    
        ```sh
           docker run -d \
            --name pg \
            -p 5432:5432 \
            -e POSTGRES_PASSWORD=${PG_PASSWORD} \
            -v /path/to/queries:/docker-entrypoint-initdb.d \
            -v pg_data:/var/lib/postgresql/data \
            postgres:14 \
            postgres -c shared_preload_libraries=pg_stat_statements
        ```
    
    Replace **`/path/to/queries`** with the path to your **`init.sql`** file. This command mounts the **`init.sql`** file to the **`docker-entrypoint-initdb.d`** directory, which is automatically executed upon container startup.
    
    
    !!! note alert alert-primary "Note"
        - If you run the services on the same instance, the `--network` and `--ip` flags are used to assign a specific IP address to the container within the Docker network created in Step 2. This IP address is referenced in subsequent steps as the PostgreSQL service address.
        - The `--network` and `--ip` flags are not required if the services are running on separate instances, as PostgreSQL will bind to the default network interface.

#### **Step 6: Running PMM Services**

The PMM server orchestrates the collection, storage, and visualization of metrics. In our high-availability setup, we'll have one active PMM server and two passive PMM servers.
{.power-number}

1. Pull the PMM Server Docker image:
    
    ```bash
    docker pull ${PMM_DOCKER_IMAGE}
    ```
    
2. Create a Docker volume for PMM-Server data:
    
    ```bash
    docker volume create pmm-server-active_data
    docker volume create pmm-server-passive_data
    docker volume create pmm-server-passive-2_data
    ```
    
3. Run the active PMM managed server. This server will serve as the primary monitoring server.
    
    You can either run all the services on the same instance or a separate instance.
    
    === "Run services on same instance"

        ```sh
        docker run -d \
        --name ${PMM_ACTIVE_NODE_ID} \
        --hostname ${PMM_ACTIVE_NODE_ID} \
        --network pmm-network \
        --ip ${PMM_ACTIVE_IP} \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES=1 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_DATABASE=pmm \
        -e PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE=10000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE=2 \
        -e PERCONA_TEST_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
        -e PERCONA_TEST_POSTGRES_USERNAME=${PG_USERNAME} \
        -e PERCONA_TEST_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
        -e GF_DATABASE_URL=postgres://${GF_USERNAME}:${GF_PASSWORD}@${PG_HOST_IP}:5432/grafana \
        -e PMM_VM_URL=http://${VM_HOST_IP}:8428 \
        -e PMM_TEST_HA_ENABLE=1 \
        -e PMM_TEST_HA_BOOTSTRAP=1 \
        -e PMM_TEST_HA_NODE_ID=${PMM_ACTIVE_NODE_ID} \
        -e PMM_TEST_HA_ADVERTISE_ADDRESS=${PMM_ACTIVE_IP} \
        -e PMM_TEST_HA_GOSSIP_PORT=9096 \
        -e PMM_TEST_HA_RAFT_PORT=9097 \
        -e PMM_TEST_HA_GRAFANA_GOSSIP_PORT=9094 \
        -e PMM_TEST_HA_PEERS=${PMM_ACTIVE_IP},${PMM_PASSIVE_IP},${PMM_PASSIVE2_IP} \
        -v pmm-server-active_data:/srv \
        ${PMM_DOCKER_IMAGE}
        ```
    
    === "Run services on a seperate instance"
    
        ```sh
        docker run -d \
        --name ${PMM_ACTIVE_NODE_ID} \
        -p 80:80 \
        -p 443:443 \
        -p 9094:9094 \
        -p 9096:9096 \
        -p 9094:9094/udp \
        -p 9096:9096/udp \
        -p 9097:9097 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES=1 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_DATABASE=pmm \
        -e PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE=10000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE=2 \
        -e PERCONA_TEST_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
        -e PERCONA_TEST_POSTGRES_USERNAME=${PG_USERNAME} \
        -e PERCONA_TEST_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
        -e GF_DATABASE_URL=postgres://${GF_USERNAME}:${GF_PASSWORD}@${PG_HOST_IP}:5432/grafana \
        -e PMM_VM_URL=http://${VM_HOST_IP}:8428 \
        -e PMM_TEST_HA_ENABLE=1 \
        -e PMM_TEST_HA_BOOTSTRAP=1 \
        -e PMM_TEST_HA_NODE_ID=${PMM_ACTIVE_NODE_ID} \
        -e PMM_TEST_HA_ADVERTISE_ADDRESS=${PMM_ACTIVE_IP} \
        -e PMM_TEST_HA_GOSSIP_PORT=9096 \
        -e PMM_TEST_HA_RAFT_PORT=9097 \
        -e PMM_TEST_HA_GRAFANA_GOSSIP_PORT=9094 \
        -e PMM_TEST_HA_PEERS=${PMM_ACTIVE_IP},${PMM_PASSIVE_IP},${PMM_PASSIVE2_IP} \
        -v pmm-server-active_data:/srv \
        ${PMM_DOCKER_IMAGE}
        ```
    
4. Run the first passive PMM managed server. This server will act as a standby server, ready to take over if the active server fails.
    
    You can either run all the services on the same instance or a separate instance.
    
    === "Run services on same instance"

        ```sh
        docker run -d \
        --name ${PMM_PASSIVE_NODE_ID} \
        --hostname ${PMM_PASSIVE_NODE_ID} \
        --network pmm-network \
        --ip ${PMM_PASSIVE_IP} \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES=1 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_DATABASE=pmm \
        -e PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE=10000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE=2 \
        -e PERCONA_TEST_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
        -e PERCONA_TEST_POSTGRES_USERNAME=${PG_USERNAME} \
        -e PERCONA_TEST_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
        -e GF_DATABASE_URL=postgres://${GF_USERNAME}:${GF_PASSWORD}@${PG_HOST_IP}:5432/grafana \
        -e PMM_VM_URL=http://${VM_HOST_IP}:8428 \
        -e PMM_TEST_HA_ENABLE=1 \
        -e PMM_TEST_HA_BOOTSTRAP=0 \
        -e PMM_TEST_HA_NODE_ID=${PMM_PASSIVE_NODE_ID} \
        -e PMM_TEST_HA_ADVERTISE_ADDRESS=${PMM_PASSIVE_IP} \
        -e PMM_TEST_HA_GOSSIP_PORT=9096 \
        -e PMM_TEST_HA_RAFT_PORT=9097 \
        -e PMM_TEST_HA_GRAFANA_GOSSIP_PORT=9094 \
        -e PMM_TEST_HA_PEERS=${PMM_ACTIVE_IP},${PMM_PASSIVE_IP},${PMM_PASSIVE2_IP} \
        -v pmm-server-passive_data:/srv \
        ${PMM_DOCKER_IMAGE}
        ```
    
    === "Run services on a seperate instance"
    
        ```sh
        docker run -d \
        --name ${PMM_PASSIVE_NODE_ID} \
        -p 80:80 \
        -p 443:443 \
        -p 9094:9094 \
        -p 9096:9096 \
        -p 9094:9094/udp \
        -p 9096:9096/udp \
        -p 9097:9097 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES=1 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_DATABASE=pmm \
        -e PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE=10000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE=2 \
        -e PERCONA_TEST_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
        -e PERCONA_TEST_POSTGRES_USERNAME=${PG_USERNAME} \
        -e PERCONA_TEST_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
        -e GF_DATABASE_URL=postgres://${GF_USERNAME}:${GF_PASSWORD}@${PG_HOST_IP}:5432/grafana \
        -e PMM_VM_URL=http://${VM_HOST_IP}:8428 \
        -e PMM_TEST_HA_ENABLE=1 \
        -e PMM_TEST_HA_BOOTSTRAP=0 \
        -e PMM_TEST_HA_NODE_ID=${PMM_PASSIVE_NODE_ID} \
        -e PMM_TEST_HA_ADVERTISE_ADDRESS=${PMM_PASSIVE_IP} \
        -e PMM_TEST_HA_GOSSIP_PORT=9096 \
        -e PMM_TEST_HA_RAFT_PORT=9097 \
        -e PMM_TEST_HA_GRAFANA_GOSSIP_PORT=9094 \
        -e PMM_TEST_HA_PEERS=${PMM_ACTIVE_IP},${PMM_PASSIVE_IP},${PMM_PASSIVE2_IP} \
        -v pmm-server-passive_data:/srv \
        ${PMM_DOCKER_IMAGE}
        ```
    
5. Run the second passive PMM managed server. Like the first passive server, this server will also act as a standby server.
    
    You can either run all the services on the same instance or a separate instance.
    
    === "Run services on same instance"
    
        ```sh
        docker run -d \
        --name ${PMM_PASSIVE2_NODE_ID} \
        --hostname ${PMM_PASSIVE2_NODE_ID} \
        --network pmm-network \
        --ip ${PMM_PASSIVE2_IP} \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES=1 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_DATABASE=pmm \
        -e PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE=10000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE=2 \
        -e PERCONA_TEST_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
        -e PERCONA_TEST_POSTGRES_USERNAME=${PG_USERNAME} \
        -e PERCONA_TEST_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
        -e GF_DATABASE_URL=postgres://${GF_USERNAME}:${GF_PASSWORD}@${PG_HOST_IP}:5432/grafana \
        -e PMM_VM_URL=http://${VM_HOST_IP}:8428 \
        -e PMM_TEST_HA_ENABLE=1 \
        -e PMM_TEST_HA_BOOTSTRAP=0 \
        -e PMM_TEST_HA_NODE_ID=${PMM_PASSIVE2_NODE_ID} \
        -e PMM_TEST_HA_ADVERTISE_ADDRESS=${PMM_PASSIVE2_IP} \
        -e PMM_TEST_HA_GOSSIP_PORT=9096 \
        -e PMM_TEST_HA_RAFT_PORT=9097 \
        -e PMM_TEST_HA_GRAFANA_GOSSIP_PORT=9094 \
        -e PMM_TEST_HA_PEERS=${PMM_ACTIVE_IP},${PMM_PASSIVE_IP},${PMM_PASSIVE2_IP} \
        -v pmm-server-passive-2_data:/srv \
        ${PMM_DOCKER_IMAGE}
        ```
    
    === "Run services on a seperate instance"
    
        ```sh
        docker run -d \
        --name ${PMM_PASSIVE2_NODE_ID} \
        -p 80:80 \
        -p 443:443 \
        -p 9094:9094 \
        -p 9096:9096 \
        -p 9094:9094/udp \
        -p 9096:9096/udp \
        -p 9097:9097 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
        -e PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES=1 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_DATABASE=pmm \
        -e PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE=10000 \
        -e PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE=2 \
        -e PERCONA_TEST_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
        -e PERCONA_TEST_POSTGRES_USERNAME=${PG_USERNAME} \
        -e PERCONA_TEST_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
        -e GF_DATABASE_URL=postgres://${GF_USERNAME}:${GF_PASSWORD}@${PG_HOST_IP}:5432/grafana \
        -e PMM_VM_URL=http://${VM_HOST_IP}:8428 \
        -e PMM_TEST_HA_ENABLE=1 \
        -e PMM_TEST_HA_BOOTSTRAP=0 \
        -e PMM_TEST_HA_NODE_ID=${PMM_PASSIVE2_NODE_ID} \
        -e PMM_TEST_HA_ADVERTISE_ADDRESS=${PMM_PASSIVE2_IP} \
        -e PMM_TEST_HA_GOSSIP_PORT=9096 \
        -e PMM_TEST_HA_RAFT_PORT=9097 \
        -e PMM_TEST_HA_GRAFANA_GOSSIP_PORT=9094 \
        -e PMM_TEST_HA_PEERS=${PMM_ACTIVE_IP},${PMM_PASSIVE_IP},${PMM_PASSIVE2_IP} \
        -v /srv/pmm-data:/srv \
        ${PMM_DOCKER_IMAGE}
        ```
        

    !!! note alert alert-primary "Note"

        - Ensure to set the environment variables from Step 1  in each instance where you run these commands.
        - If you run the service on the same instance, remove the **`-p`** flags.
        - If you run the service on a separate instance, remove the **`--network`** and **`--ip`** flags.

#### **Step 7: Set up HAProxy**

HAProxy provides high availability for your PMM setup by directing traffic to the current leader server via the `/v1/leaderHealthCheck` endpoint.
{.power-number}    

1. Pull the HAProxy Docker image:
    
    ```bash
    docker pull haproxy:2.4.2-alpine
    ```
    
2. Create a directory to store the SSL certificate:
    
    ```bash
    mkdir -p /path/to/certs
    ```
    
    Replace `/path/to/certs` with the path where you want to store your SSL certificates.
    
3. Navigate to this directory and generate a new private key:
    
    ```bash
    openssl genrsa -out pmm.key 2048
    ```
    
    This command generates a 2048-bit RSA private key and saves it to a file named `pmm.key`.
    
4. Using the private key, generate a self-signed certificate:
    
    ```bash
    openssl req -new -x509 -key pmm.key -out pmm.crt -days 365
    ```
    
    Enter country, state, organization name, etc. when asked. Use `-days 365` option for 365-day certificate validity.    

5. Copy your SSL certificate and private key to the directory you created in step 2. Ensure that the certificate file is named `pmm.crt` and the private key file is named `pmm.key`. 

    Concatenate these two files to create a PEM file:
    
    ```bash
    cat pmm.crt pmm.key > pmm.pem
    ```
    
6. Create a directory to store HA Proxy configuration:
    
    ```bash
    mkdir -p /path/to/haproxy-config
    ```
    
    Replace `/path/to/haproxy-config` with the path where you want to store your HAProxy configuration.
    
7. Create an HAProxy configuration file named `haproxy.cfg.template` in that directory. This configuration tells HAProxy to use the `/v1/leaderHealthCheck` endpoint of each PMM server to identify the leader:
    
    ```
    global
        log stdout    local0 debug
        log stdout    local1 info
        log stdout    local2 info
        daemon
    
    defaults
        log     global
        mode    http
        option  httplog
        option  dontlognull
        timeout connect 5000
        timeout client  50000
        timeout server  50000
    
    frontend http_front
        bind *:80
        default_backend http_back
    
    frontend https_front
        bind *:443 ssl crt /etc/haproxy/certs/pmm.pem
        default_backend https_back
    
    backend http_back
        option httpchk
        http-check send meth POST uri /v1/leaderHealthCheck ver HTTP/1.1 hdr Host www
        http-check expect status 200
        server pmm-server-active-http PMM_ACTIVE_IP:80 check
        server pmm-server-passive-http PMM_PASSIVE_IP:80 check backup
        server pmm-server-passive-2-http PMM_PASSIVE2_IP:80 check backup
    
    backend https_back
        option httpchk
        http-check send meth POST uri /v1/leaderHealthCheck ver HTTP/1.1 hdr Host www
        http-check expect status 200
        server pmm-server-active-https PMM_ACTIVE_IP:443 check ssl verify none
        server pmm-server-passive-https PMM_PASSIVE_IP:443 check ssl verify none
        server pmm-server-passive-2-https PMM_PASSIVE2_IP:443 check ssl verify none
    ```
    
8. Before starting the HAProxy container, use `sed` to replace the placeholders in `haproxy.cfg.template` with the environment variables, and write the output to `haproxy.cfg`:
    
    ```bash
    sed -e "s/PMM_ACTIVE_IP/$PMM_ACTIVE_IP/g" \
        -e "s/PMM_PASSIVE_IP/$PMM_PASSIVE_IP/g" \
        -e "s/PMM_PASSIVE2_IP/$PMM_PASSIVE2_IP/g" \
        /path/to/haproxy.cfg.template > /path/to/haproxy.cfg    
    ```
    
9. Run the HAProxy container, using absolute paths for all volume mounts. If running services on separate instances, remove the `--network` flag:

    
    ```bash
    docker run -d \
      --name haproxy \
      --network pmm-network \
      -p 80:80 \
      -p 443:443 \
      -v /path/to/haproxy-config:/usr/local/etc/haproxy \
      -v /path/to/certs:/etc/haproxy/certs \
      haproxy:2.4.2-alpine
    ```
    
    Replace `/path/to/haproxy-config` with the path to the `haproxy.cfg` file you created in step 6, and `/path/to/certs` with the path 
    
HAProxy is now configured to redirect traffic to the leader PMM managed server. This ensures highly reliable service by redirecting requests to the remainder of the servers in the event that the leader server goes down.

#### **Step 8: Access PMM**

You can access the PMM web interface via HAProxy once all the components are set up and configured:
{.power-number}

1. Access the PMM services by navigating to `https://<HAProxy_IP>` in your web browser. Replace `<HAProxy_IP>` with the IP address or hostname of the machine running the HAProxy container.
2. You should now see the PMM login screen. Log in using the default credentials, unless you changed them during setup.
3. You can use the PMM web interface to monitor your database infrastructure, analyze metrics, and perform various database management tasks.

When you register PMM Clients, you must use the HAProxy IP address (or hostname) rather than the PMM Server address once your PMM environment has been set up in high-availability (HA) mode. Even if one PMM server becomes unavailable, clients will still be able to communicate with the servers.

You have now successfully set up PMM in HA mode using Docker containers. Your PMM environment is more resilient to failures and can continue providing monitoring services if any of the instances fail.


!!! note alert alert-primary "Note"
    Ensure that all containers are running and accessible. You can use `docker ps` to check the status of your Docker containers. If a container is not running, you can view its logs using the command `docker logs <container_name>` to investigate the issue.