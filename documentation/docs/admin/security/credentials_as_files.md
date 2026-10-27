# Supply credentials as files

PMM Server can read the credentials it uses from files instead of environment variables. A file can be one you mount into the container or one that a secrets manager agent, such as Vault Agent, renders. A credential that exists only in a file doesn't appear in the container's environment or in the environment of any PMM Server process.

## Credentials that PMM Server reads from files

Each credential has a file name. PMM Server links every credential it resolves at `/run/pmm/secrets/<file name>` when the container starts, and its components read it from there.

| File name | What it is | File variable | Earlier environment variable |
|-----------|------------|---------------|------------------------------|
| `admin-password` | Password of the `admin` user | `PMM_ADMIN_PASSWORD_FILE` | `GF_SECURITY_ADMIN_PASSWORD`, `PMM_ADMIN_PASSWORD` |
| `grafana-secret-key` | Grafana's `[security] secret_key`, which encrypts data source and contact point secrets | `PMM_GRAFANA_SECRET_KEY_FILE` | `GF_SECURITY_SECRET_KEY` |
| `postgres-password` | PostgreSQL user of PMM Server | `PMM_POSTGRES_PASSWORD_FILE` | `PMM_POSTGRES_DBPASSWORD` |
| `grafana-db-password` | PostgreSQL user of Grafana | `PMM_GRAFANA_DB_PASSWORD_FILE` | `GF_DATABASE_PASSWORD` |
| `clickhouse-password` | ClickHouse user of PMM Server | `PMM_CLICKHOUSE_PASSWORD_FILE` | `PMM_CLICKHOUSE_PASSWORD` |
| `clickhouse-datasource-password` | Read-only ClickHouse user of the Query Analytics data source | `PMM_CLICKHOUSE_DATASOURCE_PASSWORD_FILE` | `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` |

Supply the four database passwords only when PMM Server uses an external database, for example in high availability (HA) mode. Don't supply them with the built-in databases: PMM Server doesn't apply them to its built-in ClickHouse, so Query Analytics stops working.

The encryption key that protects the credentials of monitored services is already a file. To keep it off the data volume, set `PMM_ENCRYPTION_KEY_PATH` to a key file you mount, as described in [PMM data encryption](data_encryption.md).

## How PMM Server finds a credential

PMM Server takes the first source that exists:
{.power-number}

1. The file that the credential's file variable names. Grafana's `GF_SECURITY_ADMIN_PASSWORD__FILE`, `GF_DATABASE_PASSWORD__FILE` and `GF_SECURITY_SECRET_KEY__FILE` work the same way.
2. The file with the credential's name in the directory that `PMM_SECRETS_DIR` names.
3. The earlier environment variable.
4. Nothing: the four database passwords keep their default values, and the admin password and Grafana's secret key stay unset, as in earlier versions.

PMM Server doesn't start if a configured file is missing, empty or unreadable, or if a credential is supplied both as a file and as an environment variable. The error names every affected credential, never its value.

## Docker

Put each credential in its own file, readable by UID 1000, and mount the directory into the container:

```sh
docker run --detach --restart always \
  --publish 443:8443 \
  --volume pmm-data:/srv \
  --volume /path/to/pmm-secrets:/etc/pmm/secrets:ro \
  --env PMM_SECRETS_DIR=/etc/pmm/secrets \
  --name pmm-server \
  percona/pmm-server:3
```


## Environment variables

The earlier environment variables keep working. PMM Server copies their values to files under `/run/pmm/secrets`, readable only by the PMM Server user, and removes the variables before it starts its services, so no PMM Server process inherits them. The values remain visible where you set them, for example in `docker inspect` or the pod specification, so prefer files.

## Admin password

Grafana sets `admin-password` only when it creates its database, that is on the first start. To change the password of a running installation, use the PMM UI or `change-admin-password`.

## Grafana secret key

Supply `grafana-secret-key` on a new installation, or on an existing one only with the key it already uses. Unless you set one earlier, that is Grafana's built-in default, `SW2YcwTIb9zpOOhoPsMm`. A different key makes existing data source and contact point secrets unreadable.

## After you switch

Credentials that PMM Server used before the switch can remain in older log files under `/srv/logs`, in earlier revisions of a Helm release, and in backups. Change the credentials you supply after you switch to files, so that those copies are worthless.
