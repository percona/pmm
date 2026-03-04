# Network and firewall requirements

Before installing PMM, ensure your network configuration allows the necessary connections between PMM components. Here are the required ports and connectivity settings.

For guidance on selecting the best deployment method based on these requirements, see the [choosing your PMM deployment strategy](../plan-pmm-installation/choose-deployment.md).


## System requirements

For detailed system specifications, see [Hardware and system requirements](../plan-pmm-installation/hardware_and_system.md)

Key requirements at a glance:

- Compatible with both x86_64 and ARM64 architectures
- Requires 100 MB storage for installation plus caching space
- Supports modern 64-bit Linux distributions.

This is a list of ports used by the various components of PMM. For PMM to work correctly, your system's firewall should allow TCP traffic on these ports (UDP is not needed).

### Essential ports

These are the host ports that must be accessible for basic PMM functionality:

| PMM component | Host port     | Direction     | Description
|---------------|---------------|---------------|------------------------------------------------------------------------------------------
| PMM Server    |  443 or 8443  | in            | HTTPS server for web interface and gRPC communication between PMM Client and PMM Server. Use of SSL certificates is highly encouraged.

**Container port mapping**

PMM Server containers listen on port 8443 internally. 

When running PMM in Docker or Podman, map the container port to a host port using `-p 443:8443`. 

If privileged ports (<1024) are not allowed in your environment, use:`-p 8443:8443` instead.

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

- configure appropriate proxy settings if PMM Server needs to access external services through a proxy
- consider using [offline installation methods](../install-pmm-server/deployment-options/docker/isolated_hosts.md) for environments without internet access