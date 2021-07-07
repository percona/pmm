# Network

## Ports

This is a list of ports used by the various components of PMM.

For PMM to work correctly, your system's firewall should allow TCP traffic on these ports (UDP is not needed).

Ports to expose:

| PMM component | TCP port      | Direction     | Description
|---------------|---------------|---------------|-----------------------------------------------------------------------------------------------------------------
| PMM Server    |   80          | both          | HTTP server, used for gRPC over HTTP and web interface (**insecure**, use with caution).
| PMM Server    |  443          | both          | HTTPS server, used for gRPC over HTTPS and web interface (secure, use of SSL certificates is highly encouraged).

Other ports:

| PMM component | TCP port      | Direction     | Description
|---------------|---------------|---------------|---------------------------------------------------------------
| PMM Server    | 7771          | both          | gRPC, used for communication between `pmm-agent`, `pmm-admin`.
| PMM Server    | 7772          | out           | HTTP1 server, used for older links like `logs.zip`.
| PMM Server    | 7773          | out           | Debugging.
| `pmm-agent`   | 7777          | out           | Default `pmm-agent` listen port.
| `vm-agent`    | 8428          | both          | VictoriaMetrics port.
| `pmm-agent`   | 42000 - 51999 | in            | Default range for `pmm-agent` connected agents.

!!! caution alert alert-warning "Important"
    Depending on your architecture other ports may also need to be exposed.
    - For `pmm-agent`, the default listen port is 7777.
    - The default range for agents ports can be changed with the flag `--ports-min` and  `--ports-max`, or in the configuration file.
