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

### Revoking a client token

Each node has its own token, so a compromised client host affects only that node.

To revoke a node's token, unregister the node from PMM Server:

```bash
pmm-admin unregister --force
```

This removes the node and revokes its token immediately. The node stops being monitored.

!!! note

    Tokens are stored on PMM Server as hashes only. The token value is shown once, when the node is registered, and cannot be retrieved afterwards.

### Replacing a client token

PMM has no in-place token rotation yet. The only way to give a node a different token is to register it again with `--force`:

```bash
pmm-admin config \
  --server-url=https://admin:admin@192.168.1.100:443 \
  --force
```

!!! caution alert alert-warning "Re-registering is destructive"

    `--force` removes the existing node and creates a new one. It is a recovery procedure, not a routine rotation procedure. Before using it, be aware that:

    - the node and its agents get **new IDs**, so metrics history recorded against the old IDs no longer joins up with the new ones in dashboards;
    - **all services on the node are removed** and must be added again with `pmm-admin add`, including their connection credentials, custom labels and Query Analytics settings;
    - scheduled backups that reference those services stop working;
    - the node is not monitored between unregistering and finishing the re-add.

    Plan for the services you will need to recreate before you start. Run `pmm-admin list` first and keep the output.

## Registering nodes without an admin account

Registering a node normally requires a PMM user with the **Admin** role, because adding a node is an administrative operation. That is awkward when the people who install PMM Clients are not the people who administer PMM: giving an ops team the ability to add hosts would mean giving them full administrative access.

An enrollment token is the narrower grant. It authorizes creating a node and obtaining that node's client token, and nothing else.

### Create an enrollment token

Creating, listing and revoking enrollment tokens requires the **Admin** role.

```bash
curl -X POST https://PMM_SERVER/v1/management/enrollmentTokens \
  -u admin:PASSWORD \
  -H 'Content-Type: application/json' \
  -d '{"description": "ops team rollout", "max_uses": 50}'
```

| Field | Meaning |
|-------|---------|
| `description` | Required. What the token is for, so a list of tokens is auditable. |
| `expires_at` | When the token stops working. Defaults to 30 minutes from creation. |
| `max_uses` | How many nodes the token may enroll. Omit or set `0` for unlimited. |

The response contains the token. **It is shown only once**: PMM Server stores a hash, not the token, so it cannot be retrieved later. If you lose it, revoke it and create another.

### Enroll a node with it

On the client host, pass the token in place of a username and password, using `service_token` as the username:

```bash
pmm-admin config \
  --server-url="https://service_token:ENROLLMENT_TOKEN@PMM_SERVER" \
  --server-insecure-tls \
  NODE_ADDRESS generic NODE_NAME
```

PMM Server issues the node its own client token and `pmm-agent` stores that in its configuration file. The enrollment token is not kept on the node and is not what the agent authenticates with afterwards.

### List and revoke

```bash
curl -u admin:PASSWORD https://PMM_SERVER/v1/management/enrollmentTokens
```

Listing shows each token's description, expiry, and how many of its uses are spent. It never shows token values. To revoke one, pass the `token_hash` from the listing:

```bash
curl -X DELETE -u admin:PASSWORD \
  https://PMM_SERVER/v1/management/enrollmentTokens/TOKEN_HASH
```

Revoking an enrollment token does not affect nodes it has already enrolled. Those hold their own client tokens, which are unrelated to the token that enrolled them.

!!! caution alert alert-warning "Treat an enrollment token as a credential"

    Anyone holding a valid enrollment token can add nodes to your PMM Server until it expires or its uses run out. Give it the shortest expiry and the smallest use count that fit the job, and revoke it when the rollout is finished.

### Clients registered before this model

Nodes registered with older PMM Server versions hold Grafana service account tokens with the **Admin** role, which carry far broader access than the scoped tokens described above.

These tokens keep working, so upgrading PMM Server does not break existing clients. Replacing them means re-registering each node, with the consequences described above, so weigh that against the exposure: an old client token can create further Grafana Admin credentials and reach every administrative API. Prioritise nodes on untrusted or externally reachable hosts.

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

