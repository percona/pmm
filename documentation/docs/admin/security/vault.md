# Read credentials from HashiCorp Vault

PMM can read the secrets it needs from [HashiCorp Vault](https://developer.hashicorp.com/vault/docs) instead of storing them. This covers the credentials of the databases you monitor, the token that PMM Client uses to connect to PMM Server, and the passwords that PMM Server itself starts with.

## How it works

PMM reads three kinds of secrets from Vault:

- **Database credentials**: When you add a service with a Vault path, PMM Server stores only that path. PMM Client reads the username and password from Vault whenever it starts an exporter or a Query Analytics (QAN) agent, runs an action, checks a connection, or takes a backup. The credentials never reach PMM Server.
- **PMM Server token**: PMM Client reads the token it connects to PMM Server with from a Vault reference in its configuration, each time it connects.
- **PMM Server secrets**: PMM Server reads the environment variables you set to a Vault reference when its container starts, before any of its components use them.

PMM Client caches each secret. Secrets with a lease, such as dynamic database credentials, are reused until two thirds of the lease have passed. Every other secret is reused for the cache time to live (TTL), one minute by default. At the same interval, PMM Client checks whether the secrets changed, and restarts the exporters and agents whose credentials did. Rotated passwords and renewed dynamic credentials therefore reach them without any action on your part.

If Vault is unreachable, running exporters and agents keep the credentials they have. Those that cannot start are retried at every check until Vault is back.

## Prerequisites

To read secrets from Vault, you need:

- PMM Server and PMM Client 3.10.0 or later
- A Vault secret for each set of credentials, in a [KV](https://developer.hashicorp.com/vault/docs/secrets/kv) version 1 or 2 secrets engine, or credentials issued by the [database secrets engine](https://developer.hashicorp.com/vault/docs/secrets/databases)
- A Vault policy that grants `read` on those paths to the identity PMM logs in with

## Choose an authentication method

PMM supports the following Vault authentication methods:

| Method | Use it when | Settings |
|--------|-------------|----------|
| `token` (default) | A Vault Agent keeps a token file up to date, or for testing | `--vault-token` or `--vault-token-file` |
| `approle` | PMM runs on a virtual machine or a bare-metal host | `--vault-role-id` and `--vault-secret-id` or `--vault-secret-id-file` |
| `kubernetes` | PMM runs in Kubernetes | `--vault-role`, and `--vault-jwt-file` when the service account token is not in its default location |
| `aws` | PMM runs in AWS with an IAM role | `--vault-role`, and `--vault-aws-region` or `--vault-aws-server-id` when your Vault requires them |

PMM reads a token file again whenever Vault rejects the token, so it picks up the tokens that a Vault Agent renews. Every method logs in again before its token expires.

## Configure PMM Client

Pass the Vault settings to `pmm-agent setup`, so that they are saved to the PMM Client configuration file:

```sh
pmm-agent setup \
  --vault-address=https://vault.example.com:8200 \
  --vault-auth-method=approle \
  --vault-role-id=8c3f9a1e-... \
  --vault-secret-id-file=/etc/pmm/vault-secret-id \
  ...
```

Each flag has an environment variable, such as `PMM_AGENT_VAULT_ADDRESS` for `--vault-address`. The settings are stored in the `vault` section of `pmm-agent.yaml`:

```yaml
vault:
    address: https://vault.example.com:8200
    auth-method: approle
    role-id: 8c3f9a1e-...
    secret-id-file: /etc/pmm/vault-secret-id
```

The other settings are:

- `--vault-namespace`: Vault Enterprise namespace
- `--vault-ca-cert`: CA certificate that Vault's certificate is signed with
- `--vault-auth-mount`: path the authentication method is mounted at, if it is not the method name
- `--vault-cache-ttl`: how long secrets without a lease are reused, and how often PMM Client checks for changed secrets

!!! hint
    To keep the PMM Client configuration file free of Vault credentials too, use a token file kept up to date by a Vault Agent, or the `kubernetes` or `aws` method.

## Monitor a database with credentials from Vault

To monitor a database with credentials stored in Vault:
{.power-number}

1. Store the credentials in Vault. The secret must have a `password` key and can have a `username` key:

    ```sh
    vault kv put secret/pmm/mysql-prod username=pmm password='...'
    ```

    With the database secrets engine, configure a role that issues credentials with the privileges PMM needs instead.

2. Add the service with `--vault-path` instead of `--password`. The path is the one of the Vault API, which for a KV version 2 secret includes `data/`:

    ```sh
    pmm-admin add mysql --vault-path=secret/data/pmm/mysql-prod mysql-prod 10.0.0.5:3306
    ```

    For credentials issued by the database secrets engine, use the path of the role's credentials, such as `database/creds/pmm`.

3. Check that the service is monitored. The **Status** of its exporter on the **Inventory** page shows **Running**.

When the secret has a `username` key, its value is used. Otherwise the username you give with `--username` is used.

The `--vault-path` flag is available for `pmm-admin add mysql`, `mongodb`, `postgresql`, `proxysql`, and `valkey`, and for the `pmm-admin inventory add agent` and `pmm-admin inventory change agent` commands of the agents that connect to a database. In the API, set the `vault_path` field instead of `password`.

To move an existing exporter or agent to Vault, set its Vault path. PMM Server removes the stored password:

```sh
pmm-admin inventory change agent mysqld-exporter <agent-id> --vault-path=secret/data/pmm/mysql-prod
```

To move it back, set a password. PMM Server removes the Vault path.

### Remote databases

PMM Server's own PMM Client monitors Amazon RDS and Azure databases. To read their credentials from Vault, pass the PMM Client settings to the PMM Server container as environment variables, such as `PMM_AGENT_VAULT_ADDRESS` and `PMM_AGENT_VAULT_AUTH_METHOD`.

## Read the PMM Server token from Vault

PMM Client connects to PMM Server with the token it receives when it registers. To keep the token in Vault, store it there and replace it in the PMM Client configuration with a reference of the form `vault:<path>#<key>`:

```sh
pmm-agent setup --skip-registration \
  --server-username=service_token \
  --server-password='vault:secret/data/pmm/agents#node1-token' \
  ...
```

PMM Client reads the token each time it connects to PMM Server. You can also use a reference for the password that `pmm-agent setup` registers with.

## Read PMM Server secrets from Vault

PMM Server reads its secrets from environment variables. To read one of them from Vault, set it to a reference of the form `vault:<path>#<key>`, and configure the Vault access with the `PMM_VAULT_*` variables:

```sh
docker run -d --name pmm-server \
  -e PMM_VAULT_ADDRESS=https://vault.example.com:8200 \
  -e PMM_VAULT_AUTH_METHOD=approle \
  -e PMM_VAULT_ROLE_ID=4e2a0b7d-... \
  -e PMM_VAULT_SECRET_ID_FILE=/run/secrets/vault-secret-id \
  -e PMM_POSTGRES_DBPASSWORD='vault:secret/data/pmm/server#postgres_password' \
  -e PMM_CLICKHOUSE_PASSWORD='vault:secret/data/pmm/server#clickhouse_password' \
  -e GF_SECURITY_ADMIN_PASSWORD='vault:secret/data/pmm/server#admin_password' \
  ...
  percona/pmm-server:3
```

Every variable whose name starts with `PMM_` or `GF_` can be a reference. The `PMM_VAULT_*` variables are named after the PMM Client flags: `PMM_VAULT_ADDRESS`, `PMM_VAULT_NAMESPACE`, `PMM_VAULT_CA_CERT`, `PMM_VAULT_TLS_SKIP_VERIFY`, `PMM_VAULT_AUTH_METHOD`, `PMM_VAULT_AUTH_MOUNT`, `PMM_VAULT_TOKEN`, `PMM_VAULT_TOKEN_FILE`, `PMM_VAULT_ROLE_ID`, `PMM_VAULT_SECRET_ID`, `PMM_VAULT_SECRET_ID_FILE`, `PMM_VAULT_ROLE`, `PMM_VAULT_JWT_FILE`, `PMM_VAULT_AWS_REGION`, and `PMM_VAULT_AWS_SERVER_ID`.

If PMM Server cannot read a secret, it does not start, and its log names the variable. After the secrets are read, `PMM_VAULT_TOKEN` and `PMM_VAULT_SECRET_ID` are removed from the environment of PMM Server's components.

PMM Server reads these secrets only when it starts. To apply a changed secret, restart the container.

## Limitations

Vault support has the following limitations:

- PMM Server does not read its [encryption key](data_encryption.md) from Vault. To keep it in Vault, have a Vault Agent write it to the key file.
- The credentials of backup storage locations, and the AWS and Azure keys of remote databases, are stored by PMM Server as before.
- A changed secret takes up to twice the cache TTL to reach the exporters and agents that use it.
