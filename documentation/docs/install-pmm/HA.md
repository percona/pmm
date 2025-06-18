# Install PMM in High Availability (HA) mode

When your database monitoring goes down, you lose visibility into critical performance issues just when you need it most. HA ensures your PMM monitoring stays online even when servers fail, networks disconnect, or hardware breaks.

Implement HA to build a resilient PMM deployment that keeps monitoring your databases no matter what happens to individual components.

## Understand what PMM HA can and can't do

Before you invest time in setting up HA for PMM, evaluate whether its benefits justify the added complexity for your specific use case.

Critical systems requiring sub-second failover gain the most value from PMM HA, while environments that can tolerate brief monitoring gaps (seconds to minutes) may find simpler solutions more appropriate. Consider your RTO requirements and incident response processes when deciding whether HA justifies the operational investment:

### What PMM HA provides

- Continuous monitoring visibility during server failures, preventing blind spots when you need observability most.
-  Automatic failover that restarts services or switches to backup systems without manual intervention.
- Zero metric loss during brief outages, thanks to PMM's client-side caching that preserves data until connectivity resumes.
- Reduced operational risk by maintaining monitoring coverage during critical incidents.

### What PMM HA cannot solve

- Even with perfect HA, you'll still only detect issues after PMM's minimum one-minute alerting interval.
- Complete network partitions that isolate entire segments of your infrastructure from monitoring.
- Increased operational overhead since HA introduces additional complexity in deployment, maintenance, and troubleshooting.

## HA deployment options

Choose the option that best fits your infrastructure and requirements:

=== "Docker restart"

    **Best for**: Development environments, single-server deployments, teams getting started with HA.

    Docker's built-in restart capabilities combined with PMM's client-side data buffering provide a simple yet effective way to improve availability without complex infrastructure changes. This leverages Docker automatic container recovery and PMM Client-side data caching:

     - Docker automatically restarts the PMM Server container after crashes or system reboots.
     - PMM Clients buffer metrics locally when the server is unavailable, preventing data loss during outages.

    To increase PMM availability and ensure the PMM Server automatically restarts after minor issues, launch the PMM Server in Docker with the `--restart=always` flag.

    When the PMM Server becomes unavailable, PMM Clients automatically:

     - detect the connection failure
     - begin caching metrics data locally
     - continue attempting to reconnect
     - transfer all cached data once the connection is restored

    This solution works well for environments where brief interruptions are acceptable and post-incident analysis is more important than real-time availability. For mission-critical deployments requiring higher availability, consider the more advanced options described in the following sections.

    For deployment instructions, see [Install PMM Server with Docker](../install-pmm/install-pmm-server/deployment-options/docker/index.md).

=== "Kubernetes (production)"

    **Best for**: Production environments, cloud-native architectures, teams with existing Kubernetes infrastructure.

    Kubernetes provides enterprise-grade high availability through automated container orchestration, self-healing capabilities, and intelligent workload distribution across multiple nodes. This leverages Kubernetes pod management and PMM's data persistence:

     - Kubernetes automatically restarts failed pods and reschedules them to healthy nodes.
     - persistent volumes preserve all PMM data, configurations, and dashboards across pod restarts.
     - health probes ensure only healthy instances receive traffic.
     - PMM Clients cache metrics locally during server unavailability.

    When infrastructure issues occur, Kubernetes automatically:

     - detects pod or node failures through health checks (within 30 seconds)
     - marks failed resources as unavailable
     - reschedules the PMM pod to a healthy node
     - mounts the existing persistent volume to restore state
     - routes traffic once readiness checks pass

    During failover (typically 2-5 minutes), data integrity is maintained through:

     - PMM Client-side caching of up to 24 hours of metrics.
     - persistentVolumeClaims that retain all historical data.
     - automatic metric synchronization once connection restores.
     - preservation of all configurations and custom dashboards.

    This solution works well for production environments that can tolerate brief monitoring interruptions during automatic failover. 
    
    The trade-off between operational simplicity and high availability makes it ideal for most production workloads. For zero-downtime requirements or multi-region deployments, consider the advanced clustering options in the following sections.

    For deployment instructions, see [Install PMM Server with Helm on Kubernetes clusters](../install-pmm/install-pmm-server/deployment-options/helm/index.md).

=== "Clustered (future)"

    **Best for**: Large enterprises, geographically distributed teams, maximum resilience requirements.

    A fully clustered PMM deployment is under development to provide true high availability with zero downtime and horizontal scalability. This enterprise-grade architecture will leverage Kubernetes orchestration and distributed database technologies:

     - multiple active PMM instances with automatic leader election via Raft consensus
     - clustered databases ensure no single point of failure across all data stores
     - geographic distribution support for multi-region deployments
     - automatic failover with zero data loss and minimal service interruption

    The clustered architecture will include:

     - **PMM instances**: multiple servers in active-passive configuration with automatic leader election.
     - **PostgreSQL cluster**: replicated metadata and configuration storage with automatic failover.
     - **ClickHouse cluster**: distributed query analytics data across multiple shards and replicas.
     - **VictoriaMetrics cluster**: horizontally scaled metrics storage with configurable replication factor.
     - **HAProxy**: intelligent load balancing and automatic routing to the current leader.

    This solution will address enterprise requirements for mission-critical monitoring infrastructure where any downtime is unacceptable. It will support complex scenarios including disaster recovery, multi-datacenter deployments, and regulatory compliance requiring data residency.

    This feature is currently in development. For immediate high availability needs, consider the Kubernetes deployment option described above, which provides robust automatic recovery suitable for most production environments.

=== "Manual setup (advanced)"

    **Best for**: Custom requirements that other options don't meet, integration with existing infrastructure, granular control over individual components.

    !!! caution alert alert-warning "Availability status"
        This feature is currently in [Technical Preview](../reference/glossary.md#technical-preview). Early adopters should use this feature for testing purposes only as it is subject to change.

    Manual setup provides complete control over PMM's HA architecture by deploying each component separately. This approach leverages distributed consensus protocols and external clustered databases:

     -  **Gossip protocol** enables PMM servers to discover and share information about their states. It is used for managing the PMM server list and failure detection, ensuring that all instances are aware of the current state of the cluster.
    -  **Raft consensus** ensures that PMM servers agree on a leader and that logs are replicated among all machines to maintain data consistency.
    - **External clustered databases** eliminate single points of failure for all data stores.
    - **Three PMM instances** with one leader, two followers. The leader server handles all client requests. If the leader fails, the followers take over, minimizing downtime.

    These protocols work in tandem to ensure that the PMM Server instances can effectively store and manage the data collected from your monitored databases and systems.

    The architecture separates critical services to eliminate single points of failure and provide better Service Level Agreements (SLAs):

     - **ClickHouse cluster** stores Query Analytics (QAN) metrics. This ensures that QAN data remains highly available and can be accessed even if one of the ClickHouse nodes fails.
     - **VictoriaMetrics cluster** stores Prometheus metrics. This provides a highly available and scalable solution for storing and querying metrics data.
     -  **PostgreSQL cluster** stores PMM data, such as inventory and settings. This ensures that PMM's configuration and metadata remain highly available and can be accessed by all PMM Server instances.
     - **HAProxy** routes traffic to the current leader based on health checks.

     When the leader fails, the remaining instances:

     - detect the failure through Raft consensus
     - elect a new leader from the followers
     - update HAProxy routing automatically
     - maintain service availability with minimal interruption
     - preserve all data through external clustered storage

    ### Prerequisites

    Before you begin:

     - [Install and configure Docker](https://docs.docker.com/get-docker/).
     - Prepare your environment:
         - for testing > run services on a single machine.
         - for production > deploy services on separate instances and use clustered versions of PostgreSQL, VictoriaMetrics, and ClickHouse. Keep in mind that running all services on a single machine is not recommended for production. Use separate instances and clustered components for better reliability.

    ### Step 1: Define environment variables

    Before you start, define the necessary environment variables on each instance where the services will be running. You will need these variables for the subsequent commands.

    For all IP addresses, use the format `17.10.1.x`, and for all usernames and passwords, use a string format like `example`.

    | **Variable** | **Description** |
    | :----------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------- |
    | `CH_HOST_IP` | The IP address of the instance where the ClickHouse service is running or the desired IP address for the ClickHouse container within the Docker network, depending on your setup.<br><br>Example: `17.10.1.2` |
    | `VM_HOST_IP` | The IP address of the instance where the VictoriaMetrics service is running or the desired IP address for the VictoriaMetrics container within the Docker network, depending on your setup.<br><br>Example: `17.10.1.3` |
    | `PG_HOST_IP` | The IP address of the instance where the PostgreSQL service is running or the desired IP address for the PostgreSQL container within the Docker network, depending on your setup.<br><br> Example: `17.10.1.4` |
    | `PG_USERNAME` | The username for your PostgreSQL server.<br><br> Example: `pmmuser` |
    | `PG_PASSWORD` | The password for your PostgreSQL server.<br><br>Example: `pgpassword` |
    | `GF_USERNAME` | The username for your Grafana database user.<br><br>Example: `gfuser` |
    | `GF_PASSWORD` | The password for your Grafana database user.<br><br>Example: `gfpassword` |
    | `PMM_ACTIVE_IP` | The IP address of the instance where the active PMM server is running or the desired IP address for your active PMM server container within the Docker network, depending on your setup.<br><br>Example: `17.10.1.5` |
    | `PMM_ACTIVE_NODE_ID` | The unique ID for your active PMM server node.<br><br>Example: `pmm-server-active` |
    | `PMM_PASSIVE_IP` | The IP address of the instance where the first passive PMM server is running or the desired IP address for your first passive PMM server container within the Docker network, depending on your setup.<br><br>Example: `17.10.1.6` |
    | `PMM_PASSIVE_NODE_ID` | The unique ID for your first passive PMM server node.<br><br>Example: `pmm-server-passive` |
    | `PMM_PASSIVE2_IP` | The IP address of the instance where the second passive PMM server is running or the desired IP address for your second passive PMM server container within the Docker network, depending on your setup.<br><br>Example: `17.10.1.7` |
    | `PMM_PASSIVE2_NODE_ID` | The unique ID for your second passive PMM server node.<br><br>Example: `pmm-server-passive2` |
    | `PMM_DOCKER_IMAGE` | The specific PMM Server Docker image for this guide.<br><br>Example: `percona/pmm-server:3` |

    ??? example "Expected output"

        ```bash
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

    ### Step 2: Create Docker network (optional)

    Create a dedicated network if you plan to run multiple PMM services on the same instance. This ensures proper communication between containers, especially for High Availability mode:
    {.power-number} 

    1.  Set up a Docker network for PMM services if you plan to run all the services on the same instance. As a result of this Docker network, your containers will be able to communicate with each other, which is essential for the High Availability (HA) mode to function properly in PMM. This step may be optional if you run your services on separate instances.

    2.  Run the following command to create a Docker network:

        ```sh
        docker network create pmm-network --subnet=17.10.1.0/16
        ```

    ### Step 3: Set up ClickHouse
    
    ClickHouse is an open-source column-oriented database management system. In PMM, ClickHouse stores Query Analytics (QAN) metrics, which provide detailed information about your queries.

    To set up ClickHouse:
    {.power-number}

    1.  Pull the ClickHouse Docker image:

        ```sh
        docker pull clickhouse/clickhouse-server:23.8.2.7-alpine
        ```

    2.  Create a Docker volume for ClickHouse data:

        ```sh
        docker volume create ch_data
        ```

    3.  Run the ClickHouse container:

        === "Run services on the same instance"

            ```sh
            docker run -d \
            --name ch \
            --network pmm-network \
            --ip ${CH_HOST_IP} \
            -p 9000:9000 \
            -v ch_data:/var/lib/clickhouse \
            clickhouse/clickhouse-server:23.8.2.7-alpine
            ```

            The `--network` and `--ip` flags assign a specific IP address within the Docker network created in Step 2.

        === "Run services on a separate instance"

            ```sh
            docker run -d \
            --name ch \
            -p 9000:9000 \
            -v ch_data:/var/lib/clickhouse \
            clickhouse/clickhouse-server:23.8.2.7-alpine
            ```

            When running on separate instances, ClickHouse binds to the default network interface.

    ### Step 4: Set up VictoriaMetrics
    
    VictoriaMetrics provides a long-term storage solution for your time-series data. In PMM, it is used to store Prometheus metrics.

    To set up VictoriaMetrics:
    {.power-number}

    1.  Pull the VictoriaMetrics Docker image:

        ```sh
        docker pull victoriametrics/victoria-metrics:v1.93.4
        ```

    2.  Create a Docker volume for VictoriaMetrics data:

        ```sh
        docker volume create vm_data
        ```

    3.  Run the VictoriaMetrics container:

        === "Run services on the same instance"

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

            The `--network` and `--ip` flags assign a specific IP address within the Docker network created in Step 2.

        === "Run services on a separate instance"

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

            When running on separate instances, VictoriaMetrics binds to the default network interface.

    ### Step 5: Set up PostgreSQL
  
    PostgreSQL is a powerful, open-source object-relational database system. PMM uses it to store data related to inventory, settings, and other features.

    To set up PostgreSQL:
    {.power-number}

    1.  Pull the PostgreSQL Docker image:

        ```sh
        docker pull postgres:14
        ```

    2.  Create a Docker volume for PostgreSQL data:

        ```bash
        docker volume create pg_data
        ```

    3.  Create a directory to store initialization SQL queries:

        ```bash
        mkdir -p /path/to/queries
        ```

        Replace `/path/to/queries` with your desired absolute path (e.g., `/opt/pmm/postgres-init`).

    4.  Create an `init.sql.template` file in the newly created directory with the following content:

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

    5.  Use **`sed`** to replace the placeholders with the environment variables and write the output to **`init.sql`**:

        ```bash
        sed -e 's/<YOUR_PG_USERNAME>/'"$PG_USERNAME"'/g' \
            -e 's/<YOUR_PG_PASSWORD>/'"$PG_PASSWORD"'/g' \
            -e 's/<YOUR_GF_USERNAME>/'"$GF_USERNAME"'/g' \
            -e 's/<YOUR_GF_PASSWORD>/'"$GF_PASSWORD"'/g' \
            init.sql.template > init.sql
        ```

    6.  Run the PostgreSQL container based on your deployment architecture:

        === "Run services on the same instance"

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

            The `--network` and `--ip` flags assign a specific IP address within the Docker network created in Step 2.

        === "Run services on a separate instance"

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

            When running on separate instances, PostgreSQL binds to the default network interface.

        !!! important "Path requirements"
            - Always use absolute paths for volume mounts (e.g., `/opt/pmm/postgres-init` instead of `./queries`).
            - The init SQL file in `/docker-entrypoint-initdb.d` executes automatically on first container startup.
            - Ensure the postgres user has read permissions on the mounted directory.

    ### Step 6: Running PMM services        

    The PMM server orchestrates the collection, storage, and visualization of metrics. In our high-availability setup, we'll have one active PMM server and two passive PMM servers:
    {.power-number}

    1. Check that environment variables from Step 1 are set on each instance where you run these commands.
    2. Pull the PMM Server Docker image:

        ```bash
        docker pull ${PMM_DOCKER_IMAGE}
        ```

    3.  Create Docker volumes for PMM-Server data:

        ```bash
        docker volume create pmm-server-active_data
        docker volume create pmm-server-passive_data
        docker volume create pmm-server-passive-2_data
        ```

    4.  Run the active PMM managed server. This server will serve as the primary monitoring server.

        === "Run services on same instance"

            ```sh
            docker run -d \
            --name ${PMM_ACTIVE_NODE_ID} \
            --hostname ${PMM_ACTIVE_NODE_ID} \
            --network pmm-network \
            --ip ${PMM_ACTIVE_IP} \
            -e PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
            -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
            -e PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
            -e PMM_CLICKHOUSE_DATABASE=pmm \
            -e PMM_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
            -e PMM_POSTGRES_USERNAME=${PG_USERNAME} \
            -e PMM_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
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

            When running on the same instance, omit `-p` port flags and use `--network` and `--ip` for internal communication.

        === "Run services on a separate instance"

            ```sh
            docker run -d \
            --name ${PMM_ACTIVE_NODE_ID} \
            -p 80:8080 \
            -p 443:8443 \
            -p 9094:9094 \
            -p 9096:9096 \
            -p 9094:9094/udp \
            -p 9096:9096/udp \
            -p 9097:9097 \
            -e PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
            -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
            -e PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
            -e PMM_CLICKHOUSE_DATABASE=pmm \
            -e PMM_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
            -e PMM_POSTGRES_USERNAME=${PG_USERNAME} \
            -e PMM_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
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

            When running on separate instances, include `-p` port mappings and omit `--network` and `--ip` flags.

    5.  Run the first passive PMM-managed server. This server will act as a standby server, ready to take over if the active server fails.

        === "Run services on the same instance"

            ```sh
            docker run -d \
            --name ${PMM_PASSIVE_NODE_ID} \
            --hostname ${PMM_PASSIVE_NODE_ID} \
            --network pmm-network \
            --ip ${PMM_PASSIVE_IP} \
            -e PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
            -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
            -e PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
            -e PMM_CLICKHOUSE_DATABASE=pmm \
            -e PMM_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
            -e PMM_POSTGRES_USERNAME=${PG_USERNAME} \
            -e PMM_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
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

            When running on the same instance, omit `-p` port flags and use `--network` and `--ip` for internal communication.

        === "Run services on a separate instance"

            ```sh
            docker run -d \
            --name ${PMM_PASSIVE_NODE_ID} \
            -p 80:8080 \
            -p 443:8443 \
            -p 9094:9094 \
            -p 9096:9096 \
            -p 9094:9094/udp \
            -p 9096:9096/udp \
            -p 9097:9097 \
            -e PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
            -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
            -e PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
            -e PMM_CLICKHOUSE_DATABASE=pmm \
            -e PMM_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
            -e PMM_POSTGRES_USERNAME=${PG_USERNAME} \
            -e PMM_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
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

            When running on separate instances, include `-p` port mappings and omit `--network` and `--ip` flags.

    6.  Run the second passive PMM-managed server. Like the first passive server, this server will also act as a standby server.

        === "Run services on the same instance"

            ```sh
            docker run -d \
            --name ${PMM_PASSIVE2_NODE_ID} \
            --hostname ${PMM_PASSIVE2_NODE_ID} \
            --network pmm-network \
            --ip ${PMM_PASSIVE2_IP} \
            -e PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
            -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
            -e PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
            -e PMM_CLICKHOUSE_DATABASE=pmm \
            -e PMM_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
            -e PMM_POSTGRES_USERNAME=${PG_USERNAME} \
            -e PMM_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
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

            When running on the same instance, omit `-p` port flags and use `--network` and `--ip` for internal communication.

        === "Run services on a separate instance"

            ```sh
            docker run -d \
            --name ${PMM_PASSIVE2_NODE_ID} \
            -p 80:8080 \
            -p 443:8443 \
            -p 9094:9094 \
            -p 9096:9096 \
            -p 9094:9094/udp \
            -p 9096:9096/udp \
            -p 9097:9097 \
            -e PMM_DISABLE_BUILTIN_CLICKHOUSE=1 \
            -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
            -e PMM_CLICKHOUSE_ADDR=${CH_HOST_IP}:9000 \
            -e PMM_CLICKHOUSE_DATABASE=pmm \
            -e PMM_POSTGRES_ADDR=${PG_HOST_IP}:5432 \
            -e PMM_POSTGRES_USERNAME=${PG_USERNAME} \
            -e PMM_POSTGRES_DBPASSWORD=${PG_PASSWORD} \
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

            When running on separate instances, include `-p` port mappings and omit `--network` and `--ip` flags.

    ### Step 7: Set up HAProxy
    
    HAProxy provides high availability for your PMM setup by directing traffic to the current leader server via the `/v1/leaderHealthCheck` endpoint:
    {.power-number}

    1.  Pull the HAProxy Docker image:

        ```bash
        docker pull haproxy:2.4.2-alpine
        ```

    2.  Create a directory to store the SSL certificate:

        ```bash
        mkdir -p /path/to/certs
        ```

        Replace `/path/to/certs` with the path where you want to store your SSL certificates.

    3.  Navigate to this directory and generate a new private key:

        ```bash
        openssl genrsa -out pmm.key 2048
        ```

        This command generates a 2048-bit RSA private key and saves it to a file named `pmm.key`.

    4.  Using the private key, generate a self-signed certificate:

        ```bash
        openssl req -new -x509 -key pmm.key -out pmm.crt -days 365
        ```

        Enter country, state, organization name, etc. when asked. Use `-days 365` option for 365-day certificate validity.

    5.  Copy your SSL certificate and private key to the directory you created in step 2. Ensure that the certificate file is named `pmm.crt` and the private key file is named `pmm.key`.

        Concatenate these two files to create a PEM file:

        ```bash
        cat pmm.crt pmm.key > pmm.pem
        ```

    6.  Create a directory to store HAProxy configuration:

        ```bash
        mkdir -p /path/to/haproxy-config
        ```

        Replace `/path/to/haproxy-config` with the path where you want to store your HAProxy configuration.

    7.  Create an HAProxy configuration file named `haproxy.cfg.template` in that directory. This configuration tells HAProxy to use the `/v1/leaderHealthCheck` endpoint of each PMM server to identify the leader:

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
            server pmm-server-active-http PMM_ACTIVE_IP:8080 check
            server pmm-server-passive-http PMM_PASSIVE_IP:8080 check backup
            server pmm-server-passive-2-http PMM_PASSIVE2_IP:8080 check backup

        backend https_back
            option httpchk
            http-check send meth POST uri /v1/leaderHealthCheck ver HTTP/1.1 hdr Host www
            http-check expect status 200
            server pmm-server-active-https PMM_ACTIVE_IP:8443 check ssl verify none
            server pmm-server-passive-https PMM_PASSIVE_IP:8443 check ssl verify none
            server pmm-server-passive-2-https PMM_PASSIVE2_IP:8443 check ssl verify none
        ```

    8.  Before starting the HAProxy container, use `sed` to replace the placeholders in `haproxy.cfg.template` with the environment variables, and write the output to `haproxy.cfg`:

        ```bash
        sed -e "s/PMM_ACTIVE_IP/$PMM_ACTIVE_IP/g" \
            -e "s/PMM_PASSIVE_IP/$PMM_PASSIVE_IP/g" \
            -e "s/PMM_PASSIVE2_IP/$PMM_PASSIVE2_IP/g" \
            /path/to/haproxy.cfg.template > /path/to/haproxy.cfg
        ```

    9.  Run the HAProxy container, using absolute paths for all volume mounts. If running services on separate instances, remove the `--network` flag:

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

        Replace `/path/to/haproxy-config` with the path to the `haproxy.cfg` file you created in step 6, and `/path/to/certs` with the path.

    HAProxy is now configured to redirect traffic to the leader PMM managed server. This ensures highly reliable service by redirecting requests to the remaining servers in the event that the leader server goes down.

    ### Step 8: Access PMM and verify the setup

    Once all components are running, access PMM through HAProxy and verify your high availability configuration:
    {.power-number}

     1. Access the PMM services by navigating to `https://<HAProxy_IP>` in your web browser. Replace `<HAProxy_IP>` with the IP address or hostname of the machine running the HAProxy container. HAProxy will automatically route your connection to the current leader PMM instance.
     2. Use the default credentials (`admin`/`admin`) unless changed during setup. PMM will prompt you to set a new password on first login.
     3. Verify HA status to check that your HA setup is functioning correctly. You can use `docker ps` to check the status of your Docker containers. If a container is not running, you can view its logs using the command `docker logs <container_name>` to investigate the issue.
     4. Register PMM Clients. When adding monitored nodes, always use the HAProxy address (or hostname) instead of individual PMM server IPs.

    Your PMM environment is now running in high availability mode with automatic failover capabilities. The setup provides resilience against single node failures and maintains continuous monitoring coverage.