# Connect Valkey databases to PMM

Valkey is a high-performance open-source alternative to Redis. Because Valkey is a Redis fork that maintains full protocol compatibility, PMM monitors both databases using the same proven methods and dashboards.

Connect your Valkey or Redis instances to PMM to track performance, analyze commands, and monitor cluster health. PMM's 10 dedicated **Valkey/Redis** dashboards to help you spot memory issues, diagnose slow queries, and maintain healthy replication across your Valkey/Redis deployments.


### Prerequisites

Before connecting Valkey or Redis to PMM, review the prerequisites for your monitoring setup:

=== "Local Valkey/Redis monitoring"
    - [PMM Server is installed](../../install-pmm-server/index.md) and running.
    - [PMM Client is installed](../../install-pmm-client/index.md) and the nodes are registered with PMM Server.
    - Access to the Valkey instance (localhost or network accessible)

=== "Remote Valke/Redis monitoring"
    - [PMM Server is installed](../../install-pmm-server/index.md) and running
    - PMM Server has direct network access to the Valkey instance
    - You have Valkey authentication credentials if ACL is enabled

### Security setup

For security best practices, configure proper authentication for your Valkey or Redis instance.

??? info "Password security"
    - Use a strong, unique password for Valkey connections
    - At least 12 characters long
    - Mix of uppercase and lowercase letters
    - Include numbers and special characters
    - Avoid common words or patterns
    - Never use default, test, or example passwords in production

#### ACL configuration (Valkey 6.0+)

If you're using Valkey's ACL feature, create a dedicated monitoring user:
```sh
# Create monitoring user with read-only permissions
ACL SETUSER pmm on >StrongPassword123! ~* +@read +info +config|get +slowlog +latency
```

### Add service to PMM
You can add your Valkey or Redis service to PMM either through the user interface or via the command line.

=== "Via UI"

    To add the service from the user interface:
    {.power-number}
    
    1. Go to **PMM Configuration > PMM Inventory > Services > Add Service**.
    
    2. Select **Valkey/Redis** service type.
    
    3. Fill in the **Main details** section:

        - **Service name**: defaults to `hostname` if you don't specify a custom descriptive name for your database
        - **Nodes**: Select the PMM node where the agent is running
        - **Agents**: Select the PMM agent that should monitor this instance
        - **Hostname**: The hostname or IP address of your Valkey/Redis instance
        - **Port**: The port number (default: `6379`)
        - **Username**: The username for authentication (optional)
        - **Password**: The password for authentication (optional)

    4. Configure **Labels** (optional):
        
        - **Environment**: Specify the environment (e.g., `production`, `staging`)
        - **Cluster**: Specify the cluster name if applicable
        - **Replication set**: For replica configurations
        - **Region**: Geographic region
        - **Availability Zone**: Cloud availability zone
        - **Custom labels**: Add custom key-value pairs in the format `key1:value1`, one per line

    5. Configure **Additional options**:
        
        - Check **Skip connection check** to bypass connectivity validation
        - Check **Use TLS for database connections** to enable TLS
        - Check **Skip TLS certificate and hostname validation** if using self-signed certificates. For production environments, make sure to always use properly signed certificates. Only skip certificate validation in development or testing scenarios.

    6. Click **Add service** to complete the setup.

=== "Via command line"

    === "Basic setup"
    
        Add a local Valkey instance with default settings:
        ```sh
        pmm-admin add valkey \
          --address=localhost:6379 \
          --environment=production \
          Valkey-Primary
        ```
    
    === "With authentication"
    
        Add a Valkey instance with authentication:
        ```sh
        pmm-admin add valkey \
          --address=localhost:6379 \
          --username=pmm \
          --password=StrongPassword123! \
          --environment=production \
          Valkey-Secure
        ```
    
    === "Remote monitoring"
    
        Add a remote Valkey instance:
        ```sh
        pmm-admin add valkey \
          --address=valkey-server.example.com:6379 \
          --username=pmm \
          --password=StrongPassword123! \
          --environment=production \
          Remote-Valkey
        ```
    
    === "With custom labels"
    
        Add an instance with environment and custom labels:
        ```sh
        pmm-admin add valkey \
          --address=localhost:6379 \
          --username=pmm \
          --password=StrongPassword123! \
          --environment=production \
          --custom-labels="role=primary,datacenter=east" \
          Valkey-Primary
        ```
    
    === "TLS connection"
    
        Add an instance with TLS security:
        ```sh
        pmm-admin add valkey \
          --address=valkey-server.example.com:6379 \
          --username=pmm \
          --password=StrongPassword123! \
          --tls \
          --tls-ca=/path/to/ca.pem \
          Valkey-TLS
        ```

#### Using inventory commands

PMM also provides inventory commands for more granular control:

=== "Add Valkey service via inventory"
    ```sh
    pmm-admin inventory add service valkey \
      --address=localhost:6379 \
      --username=pmm \
      --password=StrongPassword123! \
      Valkey-Service
    ```

=== "Add Valkey exporter agent"
    ```sh
    pmm-admin inventory add agent valkey_exporter \
      --address=localhost:6379 \
      --username=pmm \
      --password=StrongPassword123!
    ```

If the service is added successfully, PMM will display a confirmation message:

    ```sh
    Valkey Service added
    Service ID  : /service_id/abcd1234-5678-efgh-ijkl-mnopqrstuvwx
    Service name: Valkey-Primary
    ```

## Verify your Valkey/Redis service

After adding your Valkey or Redis service to PMM, verify that it's properly connected and collecting data.

### Check service status

=== "Via command line"
    Use these commands to manage and monitor your Valkey/Redis services:
    {.power-number}

    1. List all Valkey/Redis services and their status:
    
    ```bash
       pmm-admin inventory list services --service-type=valkey
    ```

    2. List all Valkey/Redis exporter agents:
    ```bash
       pmm-admin inventory list agents --agent-type=valkey_exporter
    ```

    3. Check the overall PMM Client status:
    ```bash
       pmm-admin status
    ```

    4. View all services in a simple list:
    ```bash
       pmm-admin list
    ```

=== "Via UI"
    To verify your service in the web interface:
    {.power-number}

    1. Navigate to **PMM Configuration > PMM Inventory**.
    2. In the **Services** tab, find your newly added Valkey/Redis service.
    3. Verify the **Service Name** and **Address** match your configuration.
    4. Check the **Status** column shows as *Active*.
    5. In the **Options** column, expand the **Details** section to confirm the correct agents are running.

### Verify data collection

After adding your Valkey or Redis service to PMM, verify that it's properly connected and collecting data.
{.power-number}

1. Open the **Home** dashboard and verify your Valkey/Redis service appears in the **Monitored DB Services** and **Monitored DB Instances** panels.

2. Navigate to the **Valkey/Redis** dashboards from the left menu. PMM provides 10 dedicated dashboards:
   - **Clients**: Monitor client connections and blocked clients
   - **Cluster Details**: Track cluster topology and replication health
   - **Command Details**: Analyze command-level performance and latency
   - **Load**: Monitor workload distribution and throughput
   - **Memory**: Track memory usage and eviction patterns
   - **Network**: Monitor network bandwidth consumption
   - **Overview**: Get a high-level summary of deployment health
   - **Persistence**: Verify AOF and RDB persistence operations
   - **Replication**: Monitor replication lag and synchronization
   - **Slowlog**: Track slow command execution

3. Select your Valkey or Redis service from the drop-down menu.

4. Confirm that metrics are appearing on the dashboards.

5. Check that the graphs show recent data (within the last few minutes).

## Remove a Valkey/Redis service

If you need to remove a Valkey or Redis service from monitoring:

=== "Via command line"
    
    Remove a Valkey or Redis service:
    ```bash
    pmm-admin remove valkey Valkey-Primary
    ```

    Or use inventory commands:
    ```bash
    # Remove service
    pmm-admin inventory remove service <service-id>
    
    # Remove agent
    pmm-admin inventory remove agent <agent-id>
    ```

=== "Via UI"
    To remove the service from the user interface:
    {.power-number}
    
    1. Navigate to **PMM Configuration > PMM Inventory**.
    2. Find your Valkey/Redis service in the **Services** tab.
    3. Click the **Remove** button in the **Options** column.
    4. Confirm the removal when prompted.

## Next steps

After successfully connecting your Valkey or Redis instance to PMM:

- Familiarize yourself with the [10 Valkey/Redis dashboards](../../../use/dashboards-panels/index.md) to understand your database performance baseline. Navigate to the **Valkey/Redis** section in the left menu to access all monitoring dashboards.

- [Configure alerts](../../../alert/index.md) for critical metrics like memory usage, replication lag, or slow queries.

- Use [PMM Inventory](../../../use/dashboard-inventory.md) to view and manage all your monitored services.

- Use the insights from [Valkey/Redis Command Details](../../../reference/dashboards/dashboard-valkey-redis-command-detail.md) and [Valkey Redis Slowlog](../../../reference/dashboards/dashboard-valkey-redis-slowlog.md) dashboards to identify and optimize slow queries. Check [Query Analytics](../../../use/qan/index.md) if slowlog data is available for your setup.

## Related topics

- [Valkey Official Documentation](https://valkey.io/docs/)
- [Redis Official Documentation](https://redis.io/docs/)
- [pmm-admin Command Reference](../../../use/commands/pmm-admin.md)
