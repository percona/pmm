# Network and firewall requirements

Before installing PMM, ensure your network configuration allows the necessary connections between PMM components. Here are the required ports and connectivity settings.

For guidance on selecting the best deployment method based on these requirements, see the [choosing your PMM deployment strategy](../install-pmm/plan-pmm-installation/choose-deployment.md).

## Required ports

This is a list of ports used by the various components of PMM. For PMM to work correctly, your system's firewall should allow TCP traffic on these ports (UDP is not needed).

### Essential ports
These ports must be accessible for basic PMM functionality:

| PMM component | TCP port      | Direction     | Description
|---------------|---------------|---------------|------------------------------------------------------------------------------------------
| PMM Server    |   80          | both          | HTTP server, used for gRPC over HTTP and web interface (**insecure**, use with caution).
| PMM Server    |  443          | both          | HTTPS server, used for gRPC over HTTPS and web interface (secure, use of SSL certificates is highly encouraged).

### Internal component ports 
These ports are used for communication between PMM components:

| PMM component | TCP port      | Direction     | Description
|---------------|---------------|---------------|-----------------------------------------------------------------
| PMM Server    | 7771          | both          | gRPC, used for communication between `pmm-agent` and `pmm-admin`.
| PMM Server    | 7772          | out           | HTTP1 server, used for older links like `logs.zip`.
| PMM Server    | 7773          | out           | Debugging.
| `pmm-agent`   | 7777          | out           | Default `pmm-agent` listen port.
| `vm-agent`    | 8428          | both          | VictoriaMetrics port.
| `pmm-agent`   | 42000 - 51999 | in            | Default range for `pmm-agent` connected agents.

## Port range configuration

The default port range for `pmm-agent` is intentionally wide to accommodate various deployment sizes. You can adjust this range to fit your environment:

- Small deployments: For monitoring fewer than 20 services, you can reduce the range significantly
- Custom range: Configure with `--ports-min` and `--ports-max` flags when starting `pmm-agent`
- Minimum allocation: Allow at least one port per monitored service/exporter

For example, to set a custom port range for 50 services:
    ```sh
    pmm-agent --ports-min=9001 --ports-max=9050
    ```

Learn more about available settings for `pmm-agent` in [Percona PMM-Agent documentation](../../use/commands/pmm-agent.md).

## Network configuration for locked-down environments
For computers in a locked-down corporate environment without direct access to the Internet:
 - make sure to [enable access to Percona Platform services](https://docs.percona.com/percona-platform/network.html)
 - configure appropriate proxy settings if PMM Server needs to access external services through a proxy
 - consider using [offline installation methods](../install-pmm-server/deployment-options/docker/isolated_hosts.md) for environments without internet access
