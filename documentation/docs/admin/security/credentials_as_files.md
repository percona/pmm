# Provide PMM Server credentials in files

Use credential files to keep PMM Server's passwords and Grafana's secret key out of the container settings, for example when a secrets manager such as Vault manages them.

PMM Server already copies the values of environment variables to files that only it can read, but the variables themselves stay visible in the container settings, for example with `docker inspect` or in the pod specification. A credential that you provide only in a file never appears there.

You can mount the file into the container yourself, or let a secrets manager agent, such as Vault Agent, render it.

## Choose the credentials to provide

Before you create the files, decide which credentials you need. For a credential that you don't provide in a file, PMM Server uses its environment variable or, without one, its default, as in earlier versions. Choose the files to provide based on your setup:

- To set the password of the `admin` user, provide `admin-password` before the first start. Grafana sets it only when it creates its database.
- If PMM Server uses an external database, for example in high availability (HA) mode, provide the four database passwords. With the built-in databases, don't provide them: PMM Server doesn't apply them to its built-in ClickHouse, so a password other than the default breaks Query Analytics (QAN).
- To set Grafana's secret key, provide `grafana-secret-key` on a new installation, or on an existing one only with the key that it already uses. Unless you set one earlier, that's Grafana's built-in default, `SW2YcwTIb9zpOOhoPsMm`. A different key makes existing data source and contact point secrets unreadable.

PMM Server finds each credential by its file name, so name your files exactly as in the **File name** column:

| File name | What it is | File variable | Earlier environment variable |
|-----------|------------|---------------|------------------------------|
| `admin-password` | Password of the `admin` user | `PMM_ADMIN_PASSWORD_FILE` | `GF_SECURITY_ADMIN_PASSWORD`, `PMM_ADMIN_PASSWORD` |
| `grafana-secret-key` | Grafana's `[security] secret_key`, which encrypts data source and contact point secrets | `PMM_GRAFANA_SECRET_KEY_FILE` | `GF_SECURITY_SECRET_KEY` |
| `postgres-password` | Password of the PostgreSQL user of PMM Server | `PMM_POSTGRES_PASSWORD_FILE` | `PMM_POSTGRES_DBPASSWORD` |
| `grafana-db-password` | Password of the PostgreSQL user of Grafana | `PMM_GRAFANA_DB_PASSWORD_FILE` | `GF_DATABASE_PASSWORD` |
| `clickhouse-password` | Password of the ClickHouse user of PMM Server | `PMM_CLICKHOUSE_PASSWORD_FILE` | `PMM_CLICKHOUSE_PASSWORD` |
| `clickhouse-datasource-password` | Password of the read-only ClickHouse user of the Query Analytics data source | `PMM_CLICKHOUSE_DATASOURCE_PASSWORD_FILE` | `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` |

## Provide the credential files

After you choose the credentials, put them in files that the PMM Server container can read. The steps depend on where PMM Server runs:

=== "PMM Server on Docker"

    To provide credentials in files:
    {.power-number}

    1. Put each credential in its own file, named as in the previous table, and make the files readable by UID 1000.
    2. Mount the directory into the container, and set `PMM_SECRETS_DIR` to it. Use a directory such as `/etc/pmm/secrets`, not `/run/pmm/secrets`, which PMM Server manages itself:

        ```sh
        docker run --detach --restart always \
          --publish 443:8443 \
          --volume pmm-data:/srv \
          --volume /path/to/pmm-secrets:/etc/pmm/secrets:ro \
          --env PMM_SECRETS_DIR=/etc/pmm/secrets \
          --name pmm-server \
          percona/pmm-server:3
        ```

        To keep one credential in another directory, set its file variable to the file's path instead, for example `--env PMM_ADMIN_PASSWORD_FILE=/path/to/admin-password`.

    3. If you provided any of these credentials as environment variables before, remove those variables. PMM Server doesn't start when it gets a credential both as a file and as an environment variable.
    4. If you switched an existing installation to files, [change the credentials](#change-a-credential) that you now provide in files, except Grafana's secret key. Credentials that PMM Server used before the switch can remain in older log files under `/srv/logs` and in backups. New credentials make those copies worthless.

=== "PMM Server on Kubernetes"

    When you install PMM Server with the `pmm` Helm chart, provide the credentials from a Kubernetes Secret:
    {.power-number}

    1. Create a Secret with one key for each credential, named as in the previous table:

        ```sh
        kubectl create secret generic pmm-credentials \
          --from-literal=admin-password='your_secure_password' \
          --namespace <namespace>
        ```

    2. Create a `values-credentials.yaml` file that mounts the Secret and sets `PMM_SECRETS_DIR` to it. Leave `secret.name` empty: otherwise the chart also passes the admin password from its own `pmm-secret` as `GF_SECURITY_ADMIN_PASSWORD`, and PMM Server doesn't start when it gets a credential both as a file and as an environment variable.

        ```yaml
        secret:
          create: false
          name: ""
        pmmEnv:
          PMM_SECRETS_DIR: /etc/pmm/secrets
        extraVolumes:
          - name: pmm-config
            emptyDir: {}
          - name: pmm-credentials
            secret:
              secretName: pmm-credentials
        extraVolumeMounts:
          - name: pmm-config
            mountPath: /usr/local/percona/pmm/config
          - name: pmm-credentials
            mountPath: /etc/pmm/secrets
            readOnly: true
        ```

        Keep the `pmm-config` volume. It's part of the chart's default values, and your list replaces them.

    3. Install PMM Server with these values:

        ```sh
        helm install pmm percona/pmm -f values-credentials.yaml --namespace <namespace>
        ```

    4. If you switched an existing installation to files, [change the credentials](#change-a-credential) that you now provide in files, except Grafana's secret key. Credentials that PMM Server used before the switch can remain in older log files under `/srv/logs`, in earlier revisions of the Helm release, and in backups. New credentials make those copies worthless.

## Change a credential

When you replace a credential after setup, change it in a way that PMM Server picks up. A changed file takes effect only when PMM Server restarts, and some credentials don't change through their file at all:

- To change a database password, update its file and restart PMM Server. PMM Server links the files at `/run/pmm/secrets/<file name>` and reads them only when it starts.
- To change the password of the `admin` user, use the PMM UI or `change-admin-password`. The `admin-password` file counts only on the first start.
- Don't change `grafana-secret-key` on an existing installation. A different key makes existing data source and contact point secrets unreadable.

## Check which value PMM Server uses

If a credential doesn't work as you expect, check where PMM Server takes it from. When the container starts, PMM Server uses the first of these sources that is set:
{.power-number}

1. The file that the credential's file variable points to, such as `PMM_ADMIN_PASSWORD_FILE`
2. The file with the credential's name in the `PMM_SECRETS_DIR` directory
3. The earlier environment variable, such as `GF_SECURITY_ADMIN_PASSWORD`
4. No source: the database passwords use their default values, and the admin password and Grafana's secret key stay unset, as in earlier versions

A file and an environment variable for the same credential don't follow this order: PMM Server [doesn't start](#if-pmm-server-doesnt-start).

Grafana's `GF_SECURITY_ADMIN_PASSWORD__FILE`, `GF_DATABASE_PASSWORD__FILE` and `GF_SECURITY_SECRET_KEY__FILE` have no effect, as in earlier versions. Use the file variables in the table instead.

## Check your environment variables before you upgrade

If you upgrade to PMM 3.11.0 or later from an installation that sets its credentials in environment variables, you don't have to switch to files. Your variables keep working, but starting with PMM 3.11.0, a few values behave differently. Before you upgrade, check your variables:

- Check that no database password has leading or trailing whitespace, such as a space, or a newline from a Kubernetes Secret created from a file. PMM Server doesn't start with such a password. To fix it, see [Remove whitespace from database passwords](#remove-whitespace-from-database-passwords).
- Check `GF_SECURITY_SECRET_KEY` for leading or trailing whitespace. If it has any, PMM Server keeps the key in the environment, where Grafana reads it as written, and logs a warning. Don't move such a key to a file: a file can't keep the whitespace, so the key would change, and existing secrets would become unreadable.
- Check the admin password for leading or trailing whitespace. PMM Server drops it and logs a warning.
- Check that `GF_SECURITY_ADMIN_PASSWORD` and `PMM_ADMIN_PASSWORD` have the same value, if you set both. If they differ, PMM Server uses `GF_SECURITY_ADMIN_PASSWORD`, the variable that Grafana read in earlier versions, and logs a warning.
- Check that every file that a variable points to with Grafana's `$__file{<path>}` exists and isn't empty. PMM Server now reads these files itself when the container starts, so it doesn't start if one is missing, unreadable or empty. Other references with Grafana's syntax, such as `${OTHER_VARIABLE}`, work as before.

### Remove whitespace from database passwords

PMM Server doesn't start if one of these database passwords has spaces or line breaks at the start or end:

- `PMM_POSTGRES_DBPASSWORD`: the PostgreSQL user of PMM Server
- `GF_DATABASE_PASSWORD`: the PostgreSQL user of Grafana
- `PMM_CLICKHOUSE_PASSWORD`: the ClickHouse user of PMM Server
- `PMM_CLICKHOUSE_DATASOURCE_PASSWORD`: the read-only ClickHouse user of the Query Analytics data source

This is because PMM Server reads these passwords from files. If you set them in environment variables, PMM Server copies them to files first. A file can't keep spaces or line breaks at the start or end, so PMM Server would log in to the database with a different password than the one the database has. Instead, PMM Server stops and its error tells you which variable to fix.

Whitespace usually gets into a password by accident. For example, an environment file can have a space after `=`, and if you create a Kubernetes Secret from a file with `kubectl create secret --from-file`, the password keeps the line break at the end of the file.

To remove the whitespace when you upgrade:
{.power-number}

1. Find the affected variables. In your current PMM Server container, run this check, which prints the name of each affected variable:

    ```sh
    docker exec pmm-server python3 -c 'import os
    for k in ("PMM_POSTGRES_DBPASSWORD", "GF_DATABASE_PASSWORD", "PMM_CLICKHOUSE_PASSWORD", "PMM_CLICKHOUSE_DATASOURCE_PASSWORD"):
        v = os.environ.get(k, "")
        if v != v.strip(): print(k)'
    ```

    With Podman, use `podman exec pmm-server` instead of `docker exec pmm-server`. On Kubernetes, use `kubectl exec <pod> --namespace <namespace> --`.

2. For each variable that the check prints, find out what else to do, depending on whether you use the built-in database or an external one:

    | Variable | Built-in database | External database |
    |----------|-------------------|-------------------|
    | `PMM_POSTGRES_DBPASSWORD` | Nothing to change. PMM Server updates the password in the database when it starts. | Change the password in PostgreSQL. |
    | `GF_DATABASE_PASSWORD` | Nothing to change if Grafana worked before the upgrade. Otherwise, remove the variable, because the built-in database accepts only the default password. | Nothing to change if Grafana worked before the upgrade. Otherwise, change the password in PostgreSQL. |
    | `PMM_CLICKHOUSE_PASSWORD` | Remove the variable, because the built-in ClickHouse accepts only the default password. | Change the password in ClickHouse. |
    | `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` | Remove the variable, because the built-in ClickHouse accepts only the default password. | Change the password in ClickHouse. |

    Grafana has always ignored trailing whitespace in its password, so if Grafana worked, its database already uses the password without the whitespace.

3. If the table tells you to change the password in the database, do it right before you upgrade. From then on, the running PMM Server can't open new connections to the database.
4. Remove the whitespace where you set the variables, and upgrade PMM Server. With the `pmm` Helm chart, the variables are in `pmmEnv` or in the Secret you created.

If you use the `pmm-ha` Helm chart, change the passwords through the chart and its Secrets instead. In the following steps, `pmm-secret` and `pmmuser-credentials` are the default Secret names:
{.power-number}

1. Run the check with `kubectl exec`, as in the previous steps.
2. If the check prints `PMM_POSTGRES_DBPASSWORD`, remove the whitespace from the password in both Secrets. The PostgreSQL operator then changes the password in the database within seconds:

    ```sh
    kubectl patch secret pmm-secret --namespace <namespace> \
      -p '{"stringData":{"PG_PASSWORD":"<password>"}}'
    kubectl patch secret pmmuser-credentials --namespace <namespace> \
      -p '{"stringData":{"password":"<password>"},"data":{"verifier":null}}'
    ```

    If you changed `pg-db.pmmUserName`, the second Secret is named `<pmmUserName>-credentials`. If a tool such as External Secrets Operator manages `pmm-secret`, change `PG_PASSWORD` in that tool's source instead of patching `pmm-secret`, because the tool would undo the patch. Before you upgrade, make sure the tool has copied the new value to `pmm-secret`, for example by forcing a sync with `kubectl annotate externalsecret <name> force-sync=$(date +%s) --overwrite --namespace <namespace>`.

3. Upgrade right away. Until the new pods run, the pods of the earlier version can't open new connections to PostgreSQL. If the check printed `PMM_CLICKHOUSE_DATASOURCE_PASSWORD`, also remove the whitespace from `clickhouse.datasource.password` in this upgrade. The chart then changes the password in ClickHouse too.

If you already upgraded and PMM Server stops right after it starts, your data is safe: PMM Server stops before it changes anything. Fix the variables as described in the previous steps, including any password change in the database, and start PMM Server again. On Kubernetes, the pod restarts in a loop: after you fix the variables, delete the pod so that it starts with the fixed values. With the `pmm-ha` chart, only the first upgraded pod does this, and the other pods keep running the earlier version until you delete it.

## Troubleshooting

If PMM Server doesn't behave as expected after you provide credentials in files, find the cause and fix it.

### If PMM Server doesn't start

If PMM Server doesn't start after you provide credentials, find the cause in the container log, and fix it:
{.power-number}

1. Check the log, for example with `docker logs pmm-server`. Each error starts with `Credentials error:` and names the affected credential, never its value.
2. Check that every file you provide exists, isn't empty and is readable by UID 1000. This includes the files that file variables point to, the files in `PMM_SECRETS_DIR`, including the targets of links, and the files in `$__file{}` references.
3. Check that `PMM_SECRETS_DIR` names a directory that exists.
4. Check that you don't provide a credential both as a file and as an environment variable. If you do, remove the variable.
5. Check that no credential variable is empty or contains only whitespace, and that no database password variable has leading or trailing whitespace. If one does, remove the whitespace. For a database password, follow [Remove whitespace from database passwords](#remove-whitespace-from-database-passwords).
6. Check that your files are outside `/run/pmm/secrets`, which PMM Server manages itself, and that nothing else is in that directory. Mount your credentials in another directory, such as `/etc/pmm/secrets`, and point `PMM_SECRETS_DIR` and the file variables there.

## Related topics

- [PMM data encryption](data_encryption.md): keep the encryption key that protects the credentials of monitored services off the data volume with `PMM_ENCRYPTION_KEY_PATH`
- [Configure environment variables for PMM Server](../../install-pmm/install-pmm-server/deployment-options/docker/env_var.md#credentials-as-files)
