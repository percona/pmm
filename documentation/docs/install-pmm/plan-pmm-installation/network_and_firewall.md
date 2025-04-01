# Network and firewall requirements

Before installing PMM, ensure your network configuration allows the necessary connections between PMM components. Here are the required ports and connectivity settings.


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

!!! caution alert alert-warning "Important"
    Depending on your architecture other ports may also need to be exposed.

    - The default port range for `pmm-agent` is large by default to accommodate any architecture size but it can be modified using the `--ports-min` and `--ports-max` flags, or by changing the configuration file. In network constraint environments, the range can be reduced to a minimum by allocating at least one port per agent monitored. Learn more about available settings for `pmm-agent` in [Percona PMM-Agent documentation](../../use/commands/pmm-agent.md).

## Network configuration for locked-down environments
For computers in a locked-down corporate environment without direct access to the Internet, make sure to enable access to Percona Platform services following the instructions in the [Percona Platform documentation](https://docs.percona.com/percona-platform/network.html).