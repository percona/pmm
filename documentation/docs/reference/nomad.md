# Configure Nomad

Percona Monitoring and Management (PMM) includes HashiCorp Nomad to enable future extensibility and enhanced service capabilities.

[Nomad](https://www.nomadproject.io/) is a workload orchestrator designed to deploy and manage containers and non-containerized applications. In PMM, Nomad provides the underlying infrastructure to:

- improve resource allocation across monitoring components
- enable future PMM extensibility 
- manage distributed monitoring agents more efficiently

Nomad is **disabled by default** in PMM and has no impact on system performance when not enabled. 

## Prerequisites

Before enabling Nomad, check that PMM Server has a public address configured under **PMM Configuration > Advanced Settings**. This is required for Nomad to function properly and enable communication between Nomad components.

### Enable Nomad

If you're an advanced user who needs Nomad for specific use cases, follow these steps to enable Nomad in PMM:
{ .power-number }

1. Start PMM Server with the `PMM_ENABLE_NOMAD` environment variable:
   ```
   -e PMM_ENABLE_NOMAD=1
   ```

2. Expose the Nomad port:
   ```
   -p 4647:4647
   ```

3. Go to PMM's **Advanced Settings** and set the public address.

??? info "Docker run command" 

    ```
    docker run -d \
    -e PMM_ENABLE_NOMAD=1 \
    -p 4647:4647 \
    -p 443:8443 \
    --name pmm-server \
    percona/pmm-server:3
    ```

### Disable Nomad

To disable Nomad:

```
-e PMM_ENABLE_NOMAD=0
```

When Nomad is disabled on the PMM Server, the Nomad agent on PMM Clients will automatically stop.

## System requirements

When Nomad is enabled, PMM Client nodes have the following additional requirements:

-  `iproute` package must be installed
-  access to cgroup must be available

## Verification

To verify that Nomad is running correctly:
{ .power-number }

1. Check that the Nomad API is available at:
   ```
   https://<PMM_SERVER_URL>/nomad/v1/nodes
   ```

2. Confirm that Nomad agents appear in the node list.

## Internal architecture

When enabled, PMM runs the following Nomad components:

- **Nomad server** on PMM Server - manages the cluster and schedules workloads
- **Nomad client** on PMM Server - executes jobs (workloads) on remote instances
- **Nomad client** on PMM Clients - executes distributed workloads

Communication between these components is secured and managed automatically when configured with the proper public address.

## API access
The Nomad API is available through the PMM Server's HTTPS port via the `/nomad` prefix. This allows you to access Nomad endpoints without requiring a separate port for the Nomad API.

- Nomad endpoints are only available to users with admin privileges
- All Nomad API endpoints are accessible under the `/nomad` path
- The standard Nomad API documentation applies, but all requests must use the `/nomad`prefix

??? info "Example API request" 

    `https://<PMM_SERVER_URL>/nomad/v1/jobs`

## Future compatibility

Nomad is included in PMM to support future extensibility features. Nomad will remain within PMM to provide infrastructure for upcoming enhancements and to deliver improved services for existing Percona customers.

## Related links

- [Nomad documentation](https://developer.hashicorp.com/nomad/docs)
