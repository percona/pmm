# Set up PMM Server

1. Check system requirements.

    **Disk**

    Approximately 1 GB of storage per monitored database node with data retention set to one week. By default, [retention](../../how-to/configure.md#data-retention) is 30 days.

    !!! hint alert alert-success "Tip"
        [Disable table statistics](../../how-to/optimize.md) to decrease the VictoriaMetrics database size.

    **Memory**

    A minimum of 2 GB per monitored database node. The increase in memory usage is not proportional to the number of nodes. For example, data from 20 nodes should be easily handled with 16 GB.

    **Architecture**

    Your CPU must support the [`SSE4.2`](https://wikipedia.org/wiki/SSE4#SSE4.2) instruction set, a requirement of ClickHouse, a third-party column-oriented database used by Query Analytics. If your CPU is lacking this instruction set you won't be able to use Query Analytics.  Additionally, since PMM 2.38.0, your CPU and any virtualization layer in use must support x86-64-v2 or your container may not start.   

1. Configure your [network](network.md).

1. Decide how you want to run PMM Server. Choose from:

    - [Docker];
    - [Podman];
    - [Helm];
    - [Virtual appliance];
    - [Amazon AWS];
    - Use the [easy install] script.

[Docker]: docker.md
[Podman]: podman.md
[Helm]: helm.md
[virtual appliance]: virtual-appliance.md
[Amazon AWS]: aws.md
[easy install]: easy-install.md
[DBbaaS]: dbaas.md

1. Authenticating using API keys.

    While adding clients to the PMM server, you use the `admin` user. However, if you change the password for the admin user from the PMM UI, then the clients will not be able to access PMM. Also, due to multiple unsuccessful login attempts Grafana will lock out the `admin` user. The solution is to use [API key](../../details/api.md#api-keys-and-authentication) for authentication. You can use API keys as a replacement for basic authentication.
