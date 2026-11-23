# Update monitored services with pmm-admin

Use `pmm-admin update` to change the settings of a service that PMM already monitors, without removing the service and adding it again. The service and its agents keep their IDs, so metrics and Query Analytics history, alerts, scheduled backups, and backup artifacts stay with the service. The only exception is a [change of the cluster](#change-the-cluster-of-a-service), which removes the scheduled backups.

Starting with PMM 3.11.0, `pmm-admin update` supports MySQL, PostgreSQL, MongoDB, Valkey, ProxySQL, HAProxy, and external services.

## Syntax

Run the `update` command in the following format, where `SERVICE_TYPE` is one of: `mysql`, `postgresql`, `mongodb`, `valkey`, `proxysql`, `haproxy`, `external`, `external-serverless`:

```bash
pmm-admin update <SERVICE_TYPE> [NAME] [ADDRESS] [FLAGS]
```

The arguments depend on the service type:

| Service type | Arguments |
|--------------|-----------|
| `mysql`, `postgresql`, `mongodb`, `valkey`, `proxysql` | `[NAME] [ADDRESS]` |
| `haproxy` | `[NAME]` |
| `external`, `external-serverless` | None. Pass the service with a flag, as described in [Choose the service](#choose-the-service). |

`pmm-admin update` takes the same flags as [`pmm-admin add`](add.md) for the same service type. To list them, run `pmm-admin update <SERVICE_TYPE> --help`.

## How update works

`pmm-admin update` changes only what you ask for:

- Only the flags that you pass are changed. All other settings keep their current values. The defaults that `--help` shows apply to `pmm-admin add` only.
- A setting reaches every agent of the service that uses it. For example, a new password reaches the exporter and the Query Analytics agents.
- If you change connection settings, such as the credentials, TLS settings, or address, PMM connects to the database with the new settings before it saves them. If the connection fails, nothing is changed. To save the settings without the check, pass `--skip-connection-check`.
- `--query-source` adds and removes Query Analytics agents. A new agent connects to the database the way the exporter does.
- Agents whose settings don't change keep running.

## Turn options off and clear lists

To turn an option off, put `no-` in front of its flag, or set the flag to `false`:

```bash
pmm-admin update mongodb mongodb-prod --no-enable-all-collectors
pmm-admin update mysql mysql-prod --tls=false
```

To clear a list or the labels, pass the flag with an empty value:

```bash
pmm-admin update mongodb mongodb-prod --disable-collectors=
pmm-admin update mysql mysql-prod --custom-labels=
```

## Choose the service

`pmm-admin update` changes the service that you pass in one of the following ways:

- by ID, with `--service-id`
- by name, with the `NAME` argument or `--service-name`

Pass the service only once: `pmm-admin update` rejects the `NAME` argument together with `--service-id` or `--service-name`. If you pass neither an ID nor a name, `pmm-admin update` changes the only service of the given type on this node, and fails if the node has none or more than one.

External services have no `NAME` argument:

- For an `external` service, pass `--service-name` or `--service-id`.
- For an `external-serverless` service, pass `--external-name` or `--service-id`. One of them is required, because the service runs on a remote node of its own.

## Preview a change

To see what a change would do without applying it, add `--dry-run`. PMM validates the change, checks the connection to the database if the change affects it, and lists the settings that would change:

```bash
pmm-admin update mongodb mongodb-prod --max-collections-limit=0 --dry-run
```

## Examples

- Remove the limit on the number of monitored collections and turn off the `collstats` collector of a MongoDB service:

    ```bash
    pmm-admin update mongodb mongodb-prod --max-collections-limit=0 --disable-collectors=collstats
    ```

    The output lists what changed on the server, each setting as the flag that sets it, followed by all current settings. A value that you pass but that equals the stored one isn't listed, and if nothing changed, the output says `No settings changed.` Secrets show as `***`. A password or certificate file that you pass is always listed, as `(new value)`, even if it equals the stored one, because the output can't compare secrets. The following output is shortened:

    ```text
    MongoDB Service updated.
    Service ID  : 857e9561-de7b-47c7-b519-38a957500937
    Service name: mongodb-prod

    Changed settings:
      --disable-collectors=collstats (was empty)
      --max-collections-limit=0 (was -1)

    Current settings:
      --environment=prod
      ...
      --username=pmm
      --password=***
      --metrics-mode=pull
      ...
      --no-tls
      ...
      --enable-all-collectors
      --disable-collectors=collstats
      --max-collections-limit=0
      ...
    ```

- Change the password of a PostgreSQL service:

    ```bash
    pmm-admin update postgresql postgresql-prod --password=new_secret_pass
    ```

- Read new credentials from a file, which keeps the password out of the shell history:

    ```bash
    pmm-admin update mongodb mongodb-prod --credentials-source=/etc/pmm/mongodb-prod.json
    ```

- Renew the TLS certificates of a MongoDB service:

    ```bash
    pmm-admin update mongodb mongodb-prod \
      --tls-certificate-key-file=/etc/ssl/mongodb/client.pem \
      --tls-ca-file=/etc/ssl/mongodb/ca.pem
    ```

- Collect Query Analytics data from Performance Schema instead of the slow query log:

    ```bash
    pmm-admin update mysql mysql-prod --query-source=perfschema
    ```

- Monitor a MySQL server that moved to a new address:

    ```bash
    pmm-admin update mysql mysql-prod 10.0.0.12:3306
    ```

- Change the environment and the custom labels of a service:

    ```bash
    pmm-admin update mysql mysql-prod --environment=staging --custom-labels=team=db,tier=1
    ```

## Change the cluster of a service

Scheduled backups belong to the cluster of a service. If you change the cluster with `--cluster`, PMM removes the scheduled backups of the service, so create them again after the update. Backup artifacts are kept.

PMM rejects the change while a backup or a restore is running in the current or the new cluster. If this happens, run the update again after the backup or restore finishes.

## Use update in scripts

With `--json`, `pmm-admin update` prints the result as JSON. The `changed` field tells whether any setting changed, and `changes` lists each changed setting with its previous and new value. A password or certificate file that you pass always counts as a change, even if it equals the stored one:

```bash
pmm-admin update mysql mysql-prod --log-level=info --json
```

When a change of `--query-source` adds or removes a Query Analytics agent, `changes` lists only `--query-source`, not the settings that come or go with the agent, such as `--max-query-length`. To read the current value of every setting, use the `settings` field.

## Settings that update can't change

`pmm-admin update` can't change the following:

- the name of the service
- the node and the pmm-agent that monitor the service (`--node-id`, `--pmm-agent-id`, and `--agent-node-id` and `--service-node-id` of an `external` service)
- the address and the node properties of an `external-serverless` service

To change them, [remove the service](../../remove-services.md) and add it again with `pmm-admin add`.

The following restrictions also apply:

- `--query-source` can't be changed for `pmm-server-postgresql`, the internal PostgreSQL database of PMM Server.
- For a service on an Amazon RDS or Azure node, `--query-source` accepts only `perfschema` for MySQL and `pgstatements` for PostgreSQL, or `none`.
- `--disable-collectors` isn't supported for Valkey services.

## See also

- [Add databases with pmm-admin](add.md)
- [Manage inventory](inventory.md)
- [pmm-admin command overview](pmm-admin.md)
