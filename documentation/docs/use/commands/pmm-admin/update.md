# Update monitored services with `pmm-admin`

Use `pmm-admin update` to change the settings of a service that PMM already monitors, without removing the service and adding it again. 

The service and its agents keep their IDs, so metrics and Query Analytics history, alerts, scheduled backups, and backup artifacts stay with the service. The only exception is a [change of the cluster](#change-the-cluster-of-a-service), which removes the scheduled backups.

Starting with PMM 3.11.0, `pmm-admin update` supports MySQL, PostgreSQL, MongoDB, Valkey, ProxySQL, HAProxy, and external services.

## Syntax

Run the `update` command in this format, where `SERVICE_TYPE` is one of `mysql`, `postgresql`, `mongodb`, `valkey`, `proxysql`, `haproxy`, `external`, `external-serverless`:

```bash
pmm-admin update <SERVICE_TYPE> [NAME] [ADDRESS] [FLAGS]
```

The arguments depend on the service type:

- `mysql`, `postgresql`, `mongodb`, `valkey`, `proxysql`: `[NAME] [ADDRESS]`
- `haproxy`: `[NAME]`
- `external`, `external-serverless`: no arguments, so pass the service with a flag, as described in [Choose the service](#choose-the-service)

`pmm-admin update` takes the same flags as [`pmm-admin add`](add.md) for the same service type. To list them, run `pmm-admin update <SERVICE_TYPE> --help`.

## Choose the service

Before it changes anything, `pmm-admin update` needs to know which service to change. Tell it by name or by ID, or leave both out when the node has only one service of that type. External services work differently, so pick the tab for your case:

=== "By name"

    Pass the name as the `NAME` argument or with `--service-name`:

    ```bash
    pmm-admin update mysql mysql-prod --log-level=info
    ```

    Pass the service only once, because `pmm-admin update` rejects the `NAME` argument together with `--service-id` or `--service-name`.

=== "By ID"

    Pass the ID with `--service-id`:

    ```bash
    pmm-admin update mysql --service-id=<SERVICE_ID> --log-level=info
    ```

    Pass the service only once, because `pmm-admin update` rejects the `NAME` argument together with `--service-id`.

=== "Only service of a type"

    To change the only service of a type on this node, pass neither an ID nor a name. If the node has none or more than one service of that type, the command fails.

=== "External service"

    External services have no `NAME` argument, so pass them with a flag:

    - for an `external` service, `--service-name` or `--service-id`
    - for an `external-serverless` service, `--external-name` or `--service-id`, one of which is required because the service runs on a remote node of its own

## Change settings

Change a setting when something about the database or how you monitor it changes, for example when you rotate a password, renew a certificate, or move the database to a new address. Pass only the flags of the settings that you want to change. Everything that you don't pass keeps its current value, and the defaults that `--help` shows apply to `pmm-admin add` only.

To make a change, find it in this list and follow its bullet:

- rotate a password, renew a certificate, or move the database to a new address: pass the new credentials, TLS files, or address. PMM first connects to the database with the new settings, so a wrong value doesn't replace a working one: if the connection fails, nothing is changed. To save them without this check, also pass `--skip-connection-check`.
- change where Query Analytics collects its data: change `--query-source`. PMM adds or removes the Query Analytics agents for you, and a new agent connects to the database the way the exporter does. If you set `--query-source=pgstatmonitor` for a PostgreSQL service and PMM can't find the `pg_stat_monitor` extension, PMM uses `pg_stat_statements` instead and shows a warning in the output.
- change the environment, the labels, or the log level: pass their flags.
- [turn an option off or clear a list](#turn-options-off-and-clear-lists): use the `no-` prefix or an empty value.
- [preview a change](#preview-a-change) before you apply it: add `--dry-run`.
- [move a service to another cluster](#change-the-cluster-of-a-service): pass `--cluster`.

You don't need to change each agent of the service one by one. PMM applies a setting to every agent that uses it, for example a new password to the exporter and the Query Analytics agents, and agents whose settings you don't change keep running.

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

## Preview a change

To see what a change would do before you apply it, preview it with `--dry-run`:
{.power-number}

1. Add `--dry-run` to the command and run it. PMM validates the change, checks the connection to the database if the change affects it, and lists the settings that would change, so that you can check them:

    ```bash
    pmm-admin update mongodb mongodb-prod --max-collections-limit=0 --dry-run
    ```

2. To apply the change, run the command again without `--dry-run`.

## Common changes

To make a common change, find it in this list and run its command:

- remove the limit on the number of monitored collections and turn off the `collstats` collector of a MongoDB service:

    ```bash
    pmm-admin update mongodb mongodb-prod --max-collections-limit=0 --disable-collectors=collstats
    ```

    Then [read the output](#read-the-output) to check which settings changed.

- change the password of a PostgreSQL service:

    ```bash
    pmm-admin update postgresql postgresql-prod --password=new_secret_pass
    ```

- read new credentials from a file, which keeps the password out of the shell history:

    ```bash
    pmm-admin update mongodb mongodb-prod --credentials-source=/etc/pmm/mongodb-prod.json
    ```

- renew the TLS certificates of a MongoDB service:

    ```bash
    pmm-admin update mongodb mongodb-prod \
      --tls-certificate-key-file=/etc/ssl/mongodb/client.pem \
      --tls-ca-file=/etc/ssl/mongodb/ca.pem
    ```

- collect Query Analytics data from Performance Schema instead of the slow query log:

    ```bash
    pmm-admin update mysql mysql-prod --query-source=perfschema
    ```

- monitor a MySQL server that moved to a new address:

    ```bash
    pmm-admin update mysql mysql-prod 10.0.0.12:3306
    ```

- change the environment and the custom labels of a service:

    ```bash
    pmm-admin update mysql mysql-prod --environment=staging --custom-labels=team=db,tier=1
    ```

## Read the output

After an update, check the output to confirm that PMM saved the settings that you meant to change. In the output:

- the `Changed settings` list shows each setting that changed, as the flag that sets it, with its previous value
- the `Current settings` list shows everything that the service uses now, with secrets shown as `***`
- a value that you pass but that equals the stored one isn't listed, and if nothing changed, the output says `No settings changed.`
- a password or certificate file that you pass is always listed as `(new value)`, even if it equals the stored one, because the output can't compare secrets

For example, the first common change gives this output, shortened:

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

## Change the cluster of a service

To move a service to another cluster, pass `--cluster`. Scheduled backups belong to the cluster of a service, so PMM removes them when you change the cluster, and keeps the backup artifacts. To change the cluster:
{.power-number}

1. Check that no backup or restore is running in the current or the new cluster. PMM rejects the change while one is running. If this happens, run the update again after the backup or restore finishes.

2. Run `pmm-admin update` with `--cluster` set to the new cluster.

3. Create the scheduled backups of the service again.

## Use update in scripts

To check the result of an update in a script, add `--json`. `pmm-admin update` then prints the result as JSON:

```bash
pmm-admin update mysql mysql-prod --log-level=info --json
```

In the JSON:

- `changed` tells whether any setting changed
- `changes` lists each changed setting with its previous and new value
- `settings` holds the current value of every setting

A password or certificate file that you pass always counts as a change, even if it equals the stored one. When a change of `--query-source` adds or removes a Query Analytics agent, `changes` lists only `--query-source`, not the settings that come or go with the agent, such as `--max-query-length`.

## Settings that update can't change

`pmm-admin update` can't change:

- the name of the service
- the node and the pmm-agent that monitor the service (`--node-id`, `--pmm-agent-id`, and `--agent-node-id` and `--service-node-id` of an `external` service)
- the address and the node properties of an `external-serverless` service
- the monitoring user of a MySQL service (`--create-user`)

To change them, [remove the service](../../remove-services.md) and add it again with `pmm-admin add`.

PMM also refuses a few changes:

- `--query-source` can't be changed for `pmm-server-postgresql`, the internal PostgreSQL database of PMM Server.
- For a service on an Amazon RDS or Azure node, `--query-source` accepts only `perfschema` for MySQL and `pgstatements` for PostgreSQL, or `none`.
- `--disable-collectors` isn't supported for Valkey services.

## Related topics

- [Add databases with pmm-admin](add.md)
- [Manage inventory](inventory.md)
- [pmm-admin command overview](pmm-admin.md)
