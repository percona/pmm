# Read credentials from HashiCorp Vault

PMM can read the secrets it needs from [HashiCorp Vault](https://developer.hashicorp.com/vault/docs) instead of storing them. This covers the credentials of the databases you monitor, the token that PMM Client uses to connect to PMM Server, the passwords that PMM Server itself starts with, and the other secrets that PMM Server stores, such as cloud and backup storage keys.

## How it works

PMM reads four kinds of secrets from Vault:

- **Database credentials**: When you add a service with a Vault path, PMM Server stores only that path. PMM Client reads the username, the password and the TLS files from Vault whenever it starts an exporter or a Query Analytics (QAN) agent, runs an action, checks a connection, or takes a backup. If you also give a password or TLS files, PMM Client stores them in Vault for you, and PMM Server does not keep them.
- **PMM Server token**: PMM Client reads the token it connects to PMM Server with from a Vault reference in its configuration, each time it connects. When PMM Client registers, it can store the token it receives in Vault.
- **PMM Server secrets**: PMM Server reads the environment variables you set to a Vault reference when its container starts, before any of its components use them.
- **Other secrets PMM Server stores**: The settings that hold a secret PMM Server uses itself, such as the keys of a backup storage location or of an AWS or Azure account, can hold a Vault reference instead. PMM Server reads the secret whenever it uses it.

PMM Client caches each secret. Secrets with a lease, such as dynamic database credentials, are reused until two thirds of the lease have passed. Every other secret is reused for the cache time to live (TTL), one minute by default. At the same interval, PMM Client checks whether the secrets changed, and restarts the exporters and agents whose credentials did. Rotated passwords and renewed dynamic credentials therefore reach them without any action on your part. Credentials that you change through PMM reach them at once.

If Vault is unreachable, or answers with an error such as when it is sealed, PMM Client and PMM Server keep using the secrets they read last, and log that they do, so that the exporters and agents that use them keep working, including those that restart. A secret with a lease is used only until the lease ends, when its credentials stop working. A secret that PMM has not read before cannot be used until Vault is back, so the exporters and agents that use it are retried at every check. If Vault refuses a secret, such as after its policy is revoked or the secret is deleted, the refusal takes effect.

## Prerequisites

To read secrets from Vault, you need:

- PMM Server and PMM Client 3.10.0 or later
- A Vault secret for each set of credentials, in a [KV](https://developer.hashicorp.com/vault/docs/secrets/kv) version 1 or 2 secrets engine, or credentials issued by the [database secrets engine](https://developer.hashicorp.com/vault/docs/secrets/databases)
- A Vault policy that grants `read` on those paths to the identity PMM logs in with, and `create` and `update` on the paths where PMM is to store credentials for you

## Choose an authentication method

PMM supports the following Vault authentication methods:

| Method | Use it when | Settings |
|--------|-------------|----------|
| `token` (default) | A Vault Agent keeps a token file up to date, or for testing | `--vault-token` or `--vault-token-file` |
| `approle` | PMM runs on a virtual machine or a bare-metal host | `--vault-role-id` and `--vault-secret-id` or `--vault-secret-id-file` |
| `kubernetes` | PMM runs in Kubernetes | `--vault-role`, and `--vault-jwt-file` when the service account token is not in its default location |
| `aws` | PMM runs in AWS with an IAM role | `--vault-role`, and `--vault-aws-region` or `--vault-aws-server-id` when your Vault requires them |
| `azure` | PMM runs on an Azure virtual machine or scale set with a managed identity | `--vault-role`, `--vault-azure-resource` when the azure auth method of Vault is configured with a resource other than `https://management.azure.com/`, and `--vault-azure-client-id` for a user-assigned managed identity |
| `gcp` | PMM runs on a Google Compute Engine instance | `--vault-role`, a role of the `gce` type: PMM logs in with an identity token of the service account of the instance |

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

### Store the credentials in Vault through PMM

You can also give the password along with the Vault path. PMM Server checks that it can connect to the database with it, unless you skip the connection check. PMM Client then stores the username and password in the secret, creating it if needed, and keeps any other keys the secret has. PMM Server stores only the path:

```sh
pmm-admin add mysql --username=pmm --password='...' --vault-path=secret/data/pmm/mysql-prod mysql-prod 10.0.0.5:3306
```

This works the same way when you change the credentials of an exporter or agent that reads them from Vault. The new password is checked, stored in its Vault secret, and every exporter and agent of that PMM Client that reads the secret restarts with it at once:

```sh
pmm-admin inventory change agent mysqld-exporter <agent-id> --password='...'
```

A password that the database refuses is not stored, so the secret keeps working credentials. Other PMM Clients that read the same secret pick up the change at their next check.

When the exporter or agent runs on the PMM Client of the host where you run `pmm-admin`, the password and the TLS files never reach PMM Server: `pmm-admin` hands them to that PMM Client, which holds them until PMM Server asks it to store them in Vault, and PMM Server gets references to them instead. It checks the connection with those, which PMM Client resolves. For an exporter or agent of another PMM Client, or for credentials given in the PMM web interface or the API, the credentials pass through PMM Server on their way to PMM Client, although PMM Server does not store them.

PMM Client stores the credentials with its own Vault identity, so its policy needs `create` and `update` on the path. Only secrets in a KV secrets engine can be written. For credentials issued by the database secrets engine, Vault manages the password, and PMM refuses to store one.

The `--vault-path` flag is available for `pmm-admin add mysql`, `mongodb`, `postgresql`, `proxysql`, and `valkey`, and for the `pmm-admin inventory add agent` and `pmm-admin inventory change agent` commands of the agents that connect to a database. In the API, set the `vault_path` field instead of `password`. In the PMM web interface, set **Vault path** in the **Main details** of the service when you add it in **Inventory > Add Service**; the **Username** and **Password** can then stay empty.

To move an existing exporter or agent to Vault, set its Vault path. PMM Server removes the stored password. To store the current password in the secret at the same time, give it with `--password`:

```sh
pmm-admin inventory change agent mysqld-exporter <agent-id> --vault-path=secret/data/pmm/mysql-prod
```

To move it back, set an empty Vault path together with a password. PMM Server stores the password and removes the Vault path:

```sh
pmm-admin inventory change agent mysqld-exporter <agent-id> --vault-path= --password='...'
```

### Read TLS files from Vault

The TLS client key, its certificate and the CA certificate of a service can be stored in Vault too, as can the password of a MongoDB client key file. Give a reference of the form `vault:<path>#<key>` instead of a file:

```sh
pmm-admin add mysql --username=pmm --password='...' --tls \
  --tls-ca=/etc/pmm/ca.pem \
  --tls-cert=/etc/pmm/client-cert.pem \
  --tls-key='vault:secret/data/pmm/mysql-prod#tls_key' \
  mysql-prod 10.0.0.5:3306
```

PMM Server stores the reference, and PMM Client reads the content from Vault each time it uses it. A reference works with or without a Vault path, for each TLS option of MySQL, MongoDB, PostgreSQL, and Valkey services, such as `--tls-key` or `--tls-key-file`, and for `--tls-certificate-key-file-password` of MongoDB. In the API, set the reference as the value of the field.

When a service has a Vault path, the TLS files you give are stored in its Vault secret along with the password, so that the key and the certificates that go with it are rotated together. They are stored under keys named after the API fields: `tls_ca`, `tls_cert`, and `tls_key`, or `tls_ca`, `tls_certificate_key`, and `tls_certificate_key_file_password` for MongoDB. PMM Server stores references to them. When you take the service off Vault, these references stay; give the TLS files again to store them in PMM Server.

When the key or a certificate changes in Vault, the exporters and agents that use it restart with the new one, as they do for a changed password.

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

### Store the token in Vault at registration

To have PMM Client store the token in Vault when it registers, give the path of the secret with `--server-token-vault-path`. The token is stored under the `token` key, and the configuration file gets the reference to it instead of the token:

```sh
pmm-agent setup \
  --server-username=admin \
  --server-password='vault:secret/data/pmm/server#admin_password' \
  --server-token-vault-path=secret/data/pmm/agents/node1 \
  ...
```

`pmm-admin config` takes the same flag. If the configuration file already reads the token from a reference, registering again, for example with `--force`, stores the new token at that reference. If PMM Client cannot store the token, `pmm-agent setup` fails, and you can register again with `--force` once Vault is reachable.

To move a registered PMM Client to Vault, store its token in Vault and pass the reference with `--server-password`. When the reference holds the token that PMM Client already uses, `pmm-agent setup` keeps the registration and saves the reference.

`pmm-agent setup` saves the Vault token and secret ID that you give with flags to the configuration file, but not those you give with the `PMM_AGENT_VAULT_TOKEN` and `PMM_AGENT_VAULT_SECRET_ID` environment variables. The `pmm-admin summary` archive includes the configuration file with its credentials masked, while Vault references are kept.

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

Every variable whose name starts with `PMM_`, `GF_`, `VM_`, or `VMAGENT_` can be a reference. The `PMM_VAULT_*` variables are named after the PMM Client flags: `PMM_VAULT_ADDRESS`, `PMM_VAULT_NAMESPACE`, `PMM_VAULT_CA_CERT`, `PMM_VAULT_TLS_SKIP_VERIFY`, `PMM_VAULT_AUTH_METHOD`, `PMM_VAULT_AUTH_MOUNT`, `PMM_VAULT_TOKEN`, `PMM_VAULT_TOKEN_FILE`, `PMM_VAULT_ROLE_ID`, `PMM_VAULT_SECRET_ID`, `PMM_VAULT_SECRET_ID_FILE`, `PMM_VAULT_ROLE`, `PMM_VAULT_JWT_FILE`, `PMM_VAULT_AWS_REGION`, `PMM_VAULT_AWS_SERVER_ID`, `PMM_VAULT_AZURE_RESOURCE`, `PMM_VAULT_AZURE_CLIENT_ID`, and `PMM_VAULT_CACHE_TTL`.

The `PMM_AGENT_*` variables of PMM Server's own PMM Client are not read at startup: PMM Client reads the references it is given itself, such as `PMM_AGENT_SERVER_PASSWORD`, so that their values are not saved to its configuration file.

If PMM Server cannot read a secret, it does not start, and its log names the variable. PMM Server keeps reading the [other secrets it stores](#read-the-other-secrets-pmm-server-stores-from-vault) from Vault while it runs. The passwords that you set in the environment are not written to the configuration files of PMM Server's components. The username and password that `PMM_VM_URL` may carry reach vmalert and vmproxy in files outside `/srv` that only PMM Server can read, and are neither in their configuration nor on their command line.

PMM Server reads these secrets when it starts, and checks them for changes in Vault at the interval of `PMM_VAULT_CACHE_TTL` while it runs. A changed secret is applied without a restart of PMM Server, by restarting the components that use it, for a few seconds each:

| Variable | Applied by |
|----------|------------|
| `PMM_TLS_CERTIFICATE`, `PMM_TLS_CERTIFICATE_KEY`, `PMM_TLS_CA_CERTIFICATE` | Reloading nginx, without dropping connections. A certificate and a key that do not belong together are not installed, so rotate both, and PMM Server installs them once they do. |
| `PMM_CLICKHOUSE_PASSWORD` | Changing the password in the built-in ClickHouse, and restarting qan-api2 and pmm-managed |
| `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` | Changing the password in the built-in ClickHouse, and restarting Grafana |
| `GF_DATABASE_PASSWORD` | Changing the password in the built-in PostgreSQL, and restarting Grafana and pmm-managed |
| `PMM_POSTGRES_DBPASSWORD` | Restarting Grafana and pmm-managed, which changes the password in the built-in PostgreSQL |
| `PMM_VM_URL` | Restarting vmproxy and pmm-managed; vmalert reads the new credentials by itself |
| `PMM_EXTENSIONS_SECRET_KEY`, `PMM_EXTENSIONS_POSTGRES_PASSWORD` | Publishing them to the side-car, and changing the password in the built-in PostgreSQL. Restart the side-car to apply them. |
| `GF_SECURITY_ADMIN_PASSWORD` | Changing the password of the admin user, see below |
| Other `GF_*` | Restarting Grafana |
| `VM_*` | Restarting VictoriaMetrics |
| `VMAGENT_*`, other `PMM_*` | Restarting pmm-managed, which sends the `VMAGENT_*` variables to the vmagents of PMM Clients |

`GF_SECURITY_SECRET_KEY` cannot be changed while PMM Server runs: Grafana cannot read the secrets it stored with another key. A secret that a component uses on another host, such as the password of an external PostgreSQL or ClickHouse, must be changed there as well. When PMM Server restarts, it reads every secret from Vault again.

When `GF_SECURITY_ADMIN_PASSWORD` is a reference, PMM Server keeps the password of the admin user the one in Vault:

- A password changed in Vault is applied at the next check, within `PMM_VAULT_CACHE_TTL`, without a restart.
- A password changed in the PMM web interface, by the admin user or by another server admin, is stored in Vault before it takes effect. If it cannot be stored, the change fails, and the password stays the one in Vault.
- At every check, PMM Server compares the password in Vault with the one that Grafana stores for the admin user, and sets it if they differ, so a password changed by other means, such as with the Grafana command-line tool, is replaced with the one in Vault within `PMM_VAULT_CACHE_TTL`. The comparison needs no login, so it does not count toward the failed logins that Grafana allows. With the database of Grafana outside PMM Server, the password is checked by logging in at start only.

The `change-admin-password` command stores the new password in Vault too:

```sh
docker exec -t pmm-server change-admin-password <new_password>
```

### Keep the keys of PMM Server in Vault

PMM Server keeps some keys in files. The following variables give them from Vault instead:

| Variable | Key |
|----------|-----|
| `PMM_ENCRYPTION_KEY` | The [encryption key](data_encryption.md) of the secrets PMM Server stores in its database, instead of `/srv/pmm-encryption.key` |
| `PMM_CLICKHOUSE_PASSWORD` | The password of the `default` user of the built-in ClickHouse |
| `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` | The password of the `grafana` user of the built-in ClickHouse, which the ClickHouse data source of Grafana connects with |
| `GF_DATABASE_PASSWORD` | The password of the `grafana` user of the built-in PostgreSQL, which Grafana connects to its database with |
| `PMM_TLS_CERTIFICATE`, `PMM_TLS_CERTIFICATE_KEY` | The certificate of the PMM Server web interface and API, and its key, in PEM format |
| `PMM_TLS_CA_CERTIFICATE` | The certificates of the CA that issued it, optional |
| `GF_SECURITY_SECRET_KEY` | The key that Grafana encrypts the secrets it stores with, such as the passwords of alert contact points and data sources |
| `PMM_EXTENSIONS_SECRET_KEY` | The key that the PMM Extensions side-car signs its sessions with, instead of `/srv/.extensions_secret_key` |
| `PMM_EXTENSIONS_POSTGRES_PASSWORD` | The password that the PMM Extensions side-car connects to the built-in PostgreSQL with, instead of `/srv/.extensions_postgres_password` |

For example:

```sh
docker run -d --name pmm-server \
  -e PMM_VAULT_ADDRESS=https://vault.example.com:8200 \
  ...
  -e PMM_ENCRYPTION_KEY='vault:secret/data/pmm/server#encryption_key' \
  -e PMM_TLS_CERTIFICATE='vault:secret/data/pmm/server#tls_cert' \
  -e PMM_TLS_CERTIFICATE_KEY='vault:secret/data/pmm/server#tls_key' \
  percona/pmm-server:3
```

The encryption key protects most of the secrets that PMM Server stores in its database: the credentials of exporters and agents with their TLS keys, the AWS and Azure keys, and the secrets that PMM Server generates itself. With the key in Vault, none of them can be read from a copy of the PMM Server data alone. The TLS files of Valkey services and the keys of backup locations are not encrypted: store them in Vault and give references to them instead, see [Read TLS files from Vault](#read-tls-files-from-vault) and [Read the other secrets PMM Server stores from Vault](#read-the-other-secrets-pmm-server-stores-from-vault). PMM Server reads the key from Vault itself, rather than at startup, so that it is not in the environment of its components, and it never writes the key to a file. To use one:
{.power-number}

1. Store a key in Vault. For a new installation, generate one:

    ```sh
    vault kv patch secret/pmm/server encryption_key="$(docker run --rm percona/pmm-server:3 pmm-encryption-rotation --generate-key)"
    ```

    For an existing installation, store the key it already uses, the content of `/srv/pmm-encryption.key`.

2. Set `PMM_ENCRYPTION_KEY` to the reference, and start PMM Server.

3. For an existing installation, remove `/srv/pmm-encryption.key` once PMM Server is running.

In High Availability mode, give every node the same reference. To rotate the key, run `pmm-encryption-rotation` as before: it stores the new key in Vault under the same reference, so the identity of PMM Server needs `create` and `update` on its path. If the rotation fails, it puts the old key back in Vault.

PMM Server writes the passwords of the built-in ClickHouse to its configuration as SHA-256 hashes only. It applies the passwords of the built-in ClickHouse and PostgreSQL at every start, and those rotated in Vault while it runs, see above; without the variables, the default passwords apply again at the next start. The certificate and its key are written to `/srv/nginx` at every start, and when they are rotated in Vault, where nginx reads them, so that PMM Server serves the certificate in Vault. When you remove the variables, PMM Server removes the files it wrote and generates a self-signed certificate again. Set `GF_SECURITY_SECRET_KEY` before Grafana stores any secret: Grafana cannot read the secrets it stored with a different key. The superuser of the built-in PostgreSQL has no password, so there is none to store: it can only connect from inside PMM Server, over the local socket. PMM Server removes the superuser password of an existing installation, and the file in `/srv` that kept it, at the next start.

To move the secret key of PMM Extensions of an existing installation to Vault, store the content of `/srv/.extensions_secret_key` under the reference: once the key from Vault is the same, PMM Server removes the file. A different key signs every session of the side-car out, and leaves the file in place, so that removing the variable returns to it. The side-car keeps reading the key and the password from `/srv/extensions`, where PMM Server publishes them at every start.

!!! note
    The token and secret ID that you give to PMM Server and to its own PMM Client, with `PMM_VAULT_TOKEN`, `PMM_VAULT_SECRET_ID`, `PMM_AGENT_VAULT_TOKEN`, and `PMM_AGENT_VAULT_SECRET_ID`, are handed over to them in files, and are not in the environment of PMM Server's components.

### Issue the certificates of Nomad from Vault

When [Nomad](../../reference/nomad.md) is enabled, the Nomad server of PMM Server and the Nomad agents of PMM Clients authenticate each other with certificates. PMM Server issues them from a CA of its own, and keeps the private key of the CA in `/srv/nomad/certs`. To issue them from the PKI secrets engine of Vault instead, so that PMM Server keeps no CA private key, set `PMM_NOMAD_VAULT_PKI_PATH` to the issue endpoint of a role, `<mount>/issue/<role>`:

1. Create a role for the names that Nomad verifies, `server.global.nomad` and `client.global.nomad`, and for `localhost` and `127.0.0.1`:

    ```sh
    vault write pki/roles/pmm-nomad \
      allowed_domains=global.nomad allow_subdomains=true \
      allow_localhost=true allow_ip_sans=true max_ttl=720h
    ```

2. Give the identity of PMM Server `update` on the issue endpoint of the role, `pki/issue/pmm-nomad`.

3. Set `PMM_NOMAD_VAULT_PKI_PATH=pki/issue/pmm-nomad`, and start PMM Server.

PMM Server issues the certificates again when two thirds of their lifetime have passed. It reloads the Nomad server with them, and sends the new client certificate to PMM Clients, whose Nomad agents restart with it. If Vault cannot issue them, PMM Server keeps the certificates it has while they are valid, and tries again every minute. When you remove the variable, PMM Server creates a CA of its own again, and issues the certificates from it.

## Read the other secrets PMM Server stores from Vault

Some secrets are used by PMM Server itself rather than by PMM Client: it scrapes exporters with their passwords, collects AWS and Azure metrics with the account keys, and manages backups with the keys of their storage location. Instead of the secret, set the setting to a reference of the form `vault:<path>#<key>`:

| Secret | Setting |
|--------|---------|
| Password of the `/metrics` endpoint of an exporter | `--agent-password` of `pmm-admin add` and `pmm-admin inventory`, or `agent_password` in the API |
| Credentials of external and HAProxy exporters | `--password` of `pmm-admin add external` and `pmm-admin add haproxy`, or `password` in the API |
| Amazon RDS keys | `aws_access_key` and `aws_secret_key` in the API |
| Azure credentials | `azure_client_secret`, `azure_client_id`, `azure_tenant_id`, and `azure_subscription_id` in the API |
| Keys of an S3 backup storage location | `access_key` and `secret_key` in the API |

In the PMM web interface, enter the reference in the field of the secret.

For example:

```sh
pmm-admin add external --service-name=app --listen-port=9100 \
  --username=pmm --password='vault:secret/data/pmm/exporters#app'
```

PMM Server reads these secrets with the Vault access of the `PMM_VAULT_*` variables, see [Read PMM Server secrets from Vault](#read-pmm-server-secrets-from-vault), so its Vault policy needs `read` on their paths. A reference that PMM Server cannot read is refused when you add or change the setting.

PMM Server checks these secrets at the interval of the `PMM_VAULT_CACHE_TTL` variable, one minute by default. When one of them changes in Vault, PMM Server sends the new one to the exporters and agents that use it, and scrapes them with it. A scrape that falls into the second an exporter takes to restart with a rotated password can fail. It reads the keys of a backup location each time it uses them. If PMM Server cannot read a secret, and has not read it before, the exporter or agent that uses it is left out of the configuration, and the PMM Server log says why. The configuration files that the PMM Server logs archive (`logs.zip`) includes have their credentials masked.

The password of a monitored database can be a reference too, as an alternative to the Vault path: `--password='vault:secret/data/pmm/mysql-prod#password'`. Unlike the settings above, it is read by PMM Client, as for a Vault path.

## Limitations

Vault support has the following limitations:

- PMM Server keeps the certificates of Nomad in `/srv` unless they are [issued from Vault](#issue-the-certificates-of-nomad-from-vault). It publishes the secret key and the database password of PMM Extensions to `/srv/extensions`, where the side-car reads them.
- PMM Server reads the [other secrets it stores](#read-the-other-secrets-pmm-server-stores-from-vault) itself, so they reach PMM Server, unlike database credentials.
- A secret changed in Vault takes up to the cache TTL to reach the exporters and agents that use it.
- A password given with a Vault path in the PMM web interface, in the API, or with `pmm-admin` for another PMM Client passes through PMM Server on its way to PMM Client, although PMM Server does not store it. To keep it away from PMM Server entirely, run `pmm-admin` on the host of the PMM Client, or store it in Vault yourself.
