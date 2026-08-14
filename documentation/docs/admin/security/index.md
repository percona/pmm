# About security in PMM


By default, PMM ships with a self-signed certificate to enable usage out of the box. While this does enable users to have encrypted connections between clients (database clients and web/API clients) and the PMM Server, it shouldn't be considered a properly secured connection.  

Taking the following precautions will ensure that you are truly secure:

- [SSL encryption with trusted certificates](../../admin/security/ssl_encryption.md) to secure traffic between clients and server;

- [Grafana HTTPS secure cookies](../../admin/security/grafana_cookies.md)
- [Encrypt the PMM Client configuration file](client_config_encryption.md) to protect stored credentials on client hosts

## How PMM Clients authenticate

Each PMM Client holds a token that PMM Server issued to it when the node was registered. The token is stored in the PMM Client configuration file, so what an attacker gains from a compromised client host is exactly what that token can do.

### What a client token can do

A client token is scoped to the node it was issued for. It can:

- connect its pmm-agent to PMM Server, and send metrics and query analytics data for that node;
- run the operations `pmm-admin` performs on behalf of that node, such as adding or removing a service, adding annotations, and reading that node's inventory.

It cannot do anything else. Specifically, a client token has no access to:

- backups, dumps and restores;
- server settings and access control;
- PMM Server logs (`logs.zip`);
- the Query Analytics and VictoriaMetrics query APIs;
- Percona Platform APIs;
- Grafana dashboards, users or service accounts;
- **any other node's inventory**. A token issued for one node cannot read or modify another node, and inventory listings return only that node's own services and agents.

### Rotating and revoking a client token

Each node has its own token, so a compromised client host affects only that node. To replace a node's token, re-register the node:

```bash
pmm-admin config \
  --server-url=https://admin:admin@192.168.1.100:443 \
  --force
```

Re-registering issues a new token and invalidates the previous one. Removing a node with `pmm-admin unregister` revokes its token outright.

!!! note

    Tokens are stored on PMM Server as hashes only. The token value is shown once, when the node is registered, and cannot be retrieved afterwards. If you lose it, re-register the node.

### Clients registered before this model

Nodes registered with older PMM Server versions hold Grafana service account tokens with the **Admin** role, which carry far broader access than the scoped tokens described above. Re-register those nodes to replace their credentials.

## Manually configure the PostgreSQL Grafana datasource

Starting with PMM 3.9.0, PMM no longer provisions a PostgreSQL Grafana datasource by default. If you need to query PMM's internal PostgreSQL database directly, you can add the datasource manually.

To prevent unauthorized data modification, configure the datasource with a database user that has **SELECT**-only permissions.

To add the PostgreSQL datasource:
{.power-number}

1. From the **Home** page, locate the Search icon (top right on the screen).
2. Type "plugins" and select **Administration > Plugins and data > Plugins**.
3. Find and open the **Postgres** plugin.
4. Select **Add new datasource**.
5. Configure the connection parameters and click **Save & test**.
6. If the test is successful, you can start using the datasource from the **Explore** page.

!!! note

    You might see this datasource labeled as unsupported. You can safely disregard that label. The PostgreSQL datasource works as expected.

