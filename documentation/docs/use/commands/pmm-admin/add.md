# Add database services with pmm-admin

Use `pmm-admin add` to add database services to PMM monitoring from the command line. This command supports MySQL, PostgreSQL, MongoDB, Valkey, ProxySQL, and HAProxy.

To add services through the web interface instead, see [Connect databases in the PMM UI](../../../install-pmm/install-pmm-client/connect-database/index.md). For programmatic access, see the [PMM API](../../../api/index.md)

## Syntax

Run the `add` command in the format below. Keep in mind that `SERVICE_TYPE` is one of: `mysql`, `postgresql`, `mongodb`, `valkey`, `proxysql`, `haproxy`, `external`, `external-serverless`:

```bash
pmm-admin add <SERVICE_TYPE> [NAME] [ADDRESS] [FLAGS]
```

## Flag reference

Control connection settings, TLS, query collection, metric collectors, and service organization using the flags available for each database type:

### Connection flags

| Flag | MySQL | PG | Mongo | Valkey | Proxy | HAProxy |
|------|:-----:|:--:|:-----:|:------:|:-----:|:-------:|
| `--username`<br>(database username) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `--password`<br>(database password) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `--address`<br>(host and port) | ✓ | | | | | |
| `--socket`<br>(Unix socket path) | ✓ | | | | | |
| `--database`<br>(database name) | | ✓ | | | | |
| `--extra-dsn`<br>(additional DSN parameters) | ✓ | | | | | |
| `--listen-port`<br>(metrics listen port) | | | | | | ✓ |
| `--scheme`<br>(http/https) | | | | | | ✓ |
| `--metrics-path`<br>(metrics endpoint path) | | | | | | ✓ |

### TLS flags

| Flag | MySQL | PG | Mongo | Valkey | Proxy | HAProxy |
|------|:-----:|:--:|:-----:|:------:|:-----:|:-------:|
| `--tls`<br>(enable TLS for connection) | ✓ | ✓ | ✓ | | ✓ | ✓ |
| `--tls-skip-verify`<br>(skip certificate verification) | ✓ | ✓ | ✓ | | ✓ | ✓ |
| `--tls-ca-file`<br>(CA certificate file path) | ✓ | ✓ | ✓ | | ✓ | ✓ |
| `--tls-cert-file`<br>(client certificate file path) | ✓ | ✓ | ✓ | | ✓ | ✓ |
| `--tls-key-file`<br>(client key file path) | ✓ | ✓ | ✓ | | ✓ | ✓ |

### Query Analytics (QAN) flags

| Flag | MySQL | PostgreSQL | MongoDB | Valkey | ProxySQL | HAProxy |
|------|:-----:|:----------:|:-------:|:------:|:--------:|:-------:|
| `--query-source`<br>Query source | ✓ | ✓ | ✓ | | | |
| `--disable-queryexamples`<br>Disable query examples | ✓ | ✓ | | | | |
| `--max-query-length`<br>Max query length | ✓ | ✓ | ✓ | | | |
| `--comments-parsing`<br>Parse query comments | ✓ | ✓ | | | | |
| `--size-slow-logs`<br>Slow log rotation size | ✓ | | | | | |

### Collector flags

| Flag | MySQL | PostgreSQL | MongoDB | Valkey | ProxySQL | HAProxy |
|------|:-----:|:----------:|:-------:|:------:|:--------:|:-------:|
| `--disable-collectors`<br>Exclude collectors | ✓ | | ✓ | | ✓ | |
| `--enable-all-collectors`<br>Enable all collectors | | | ✓ | | | |
| `--max-collections-limit`<br>Max collections to monitor | | | ✓ | | | |
| `--stats-collections`<br>Collections for stats | | | ✓ | | | |
| `--disable-tablestats`<br>Disable table statistics | ✓ | | | | | |
| `--disable-tablestats-limit`<br>Table count limit | ✓ | | | | | |

### Service organization flags

| Flag | MySQL | PostgreSQL | MongoDB | Valkey | ProxySQL | HAProxy |
|------|:-----:|:----------:|:-------:|:------:|:--------:|:-------:|
| `--environment`<br>Environment name | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `--cluster`<br>Cluster name | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `--replication-set`<br>Replication set name | ✓ | ✓ | ✓ | | ✓ | ✓ |
| `--custom-labels`<br>Custom labels | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

### Agent flags

| Flag | MySQL | PostgreSQL | MongoDB | Valkey | ProxySQL | HAProxy |
|------|:-----:|:----------:|:-------:|:------:|:--------:|:-------:|
| `--agent-password`<br>Override metrics endpoint password | ✓ | ✓ | ✓ | ✓ | ✓ | |
| `--agent-env-vars`<br>Environment variables for exporter | | | ✓ | | | |
| `--metrics-mode`<br>Metrics flow mode (auto/push/pull) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `--node-id`<br>Node ID | ✓ | ✓ | ✓ | ✓ | ✓ | |
| `--pmm-agent-id`<br>PMM Agent ID | ✓ | ✓ | ✓ | ✓ | ✓ | |
| `--service-node-id`<br>Service node ID | | | | | | ✓ |
| `--skip-connection-check`<br>Skip connection check | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

## Add MySQL

Add a MySQL instance to monitoring:

```bash
pmm-admin add mysql [NAME] [ADDRESS] [FLAGS]
```

### Connection options

Connect to MySQL using TCP or socket:

| Flag | Description | Default |
|------|-------------|---------|
| `--address` | MySQL address and port | `127.0.0.1:3306` |
| `--socket` | Path to MySQL socket | |
| `--username` | MySQL username | |
| `--password` | MySQL password | |
| `--extra-dsn` | Additional DSN parameters | |

Find the socket path:

```bash
mysql -u root -p -e "select @@socket"
```

Enable cleartext authentication for PAM or external auth:

```bash
pmm-admin add mysql --extra-dsn="allowCleartextPasswords=1" ...
```

!!! caution "Security warning"
    Cleartext authentication transmits passwords without encryption. Use only with TLS or on trusted networks.

### TLS options

Secure the connection with TLS:

| Flag | Description |
|------|-------------|
| `--tls` | Use TLS to connect |
| `--tls-skip-verify` | Skip certificate validation |
| `--tls-ca` | Path to CA certificate |
| `--tls-cert` | Path to client certificate |
| `--tls-key` | Path to client key |

### Query Analytics options

Configure query collection for QAN:

| Flag | Description | Default |
|------|-------------|---------|
| `--query-source` | Query source: `slowlog`, `perfschema`, `none` | `slowlog` |
| `--disable-queryexamples` | Disable query example collection | `false` |
| `--max-query-length` | Max query length (-1=unlimited, 0=default 2048) | `0` |
| `--size-slow-logs` | Rotate slow log at this size (e.g., `1GiB`) | Server default |
| `--comments-parsing` | Parse comments into QAN filters: `on`, `off` | `off` |

!!! note
    For `slowlog` query source, PMM needs permissions to read the slow query log file.

!!! caution
    Do not set `--max-query-length` to 1, 2, or 3. These values will cause the PMM agent to terminate.

### Table statistics options

Control table statistics collection:

| Flag | Description |
|------|-------------|
| `--disable-tablestats` | Disable table statistics collection |
| `--disable-tablestats-limit` | Disable if more than N tables (0=no limit, negative=disable) |

### Examples

- Add MySQL with slow query log:

  ```bash
  pmm-admin add mysql \
    --username=pmm \
    --password=pass \
    --query-source=slowlog \
    mysql-prod 192.168.1.10:3306
  ```

- Add MySQL with Performance Schema:

  ```bash
  pmm-admin add mysql \
    --username=pmm \
    --password=pass \
    --query-source=perfschema \
    mysql-prod 192.168.1.10:3306
  ```

- Add MySQL with TLS:

    ```bash
    pmm-admin add mysql \
      --username=pmm \
      --password=pass \
      --tls \
      --tls-ca=/path/to/ca.pem \
      --tls-cert=/path/to/client-cert.pem \
      --tls-key=/path/to/client-key.pem \
      mysql-prod 192.168.1.10:3306
    ```

- Add MySQL without query examples (for sensitive data):

  ```bash
  pmm-admin add mysql \
    --username=pmm \
    --password=pass \
    --disable-queryexamples \
    mysql-prod 192.168.1.10:3306
  ```

- Add MySQL via socket:

  ```bash
  pmm-admin add mysql \
    --username=pmm \
    --password=pass \
    --socket=/var/run/mysqld/mysqld.sock \
    mysql-local
  ```

### Disable collectors

Exclude specific collectors from monitoring:

```bash
pmm-admin add mysql \
  --disable-collectors='heartbeat,global_status,info_schema.innodb_cmp' \
  --username=pmm \
  --password=pass \
  mysql-prod 192.168.1.10:3306
```

For available collectors, see the [mysqld_exporter repository](https://github.com/percona/mysqld_exporter).


## Add PostgreSQL

Add a PostgreSQL instance to monitoring:
```bash
pmm-admin add postgresql [NAME] [ADDRESS] [FLAGS]
```

### Connection options

Connect using `--username` and `--password`. The `--database` flag specifies which database to connect to (defaults to `postgres`).

### TLS options

Secure the connection between PMM and your PostgreSQL instance with TLS:

- `--tls`
:   Use TLS to connect.

- `--tls-skip-verify`
:   Skip certificate validation.

- `--tls-ca-file`
:   Path to the CA certificate file.

- `--tls-cert-file`
:   Path to the client certificate file.

- `--tls-key-file`
:   Path to the client key file.

### Query Analytics options

Control how PMM collects query data for Query Analytics (QAN):

- `--query-source`
:   Source for collecting queries: `pgstatements` (default), `pgstatmonitor`, or `none`.

- `--disable-queryexamples`
:   Disable collection of query examples. Only applies when `--query-source` is set to `pgstatmonitor`. With `pgstatements`, query examples are never collected.

- `--max-query-length`
:   Maximum query length in QAN. Set to `-1` for unlimited, `0` for the default (2048 characters), or a specific number to truncate after that many characters. Do not set to 1, 2, or 3 as these values will cause the PMM agent to terminate.

- `--comments-parsing`
:   Enable or disable parsing comments from queries into QAN filter groups: `on` or `off` (default).

!!! note
    Query examples are only available with `pgstatmonitor`. With `pgstatements`, query examples are never collected.

    Do not set `--max-query-length` to 1, 2, or 3. These values will cause the PMM agent to terminate.

### Examples

- Add PostgreSQL with `pg_stat_statements`:

  ```bash
  pmm-admin add postgresql \
    --username=pmm \
    --password=pass \
    --query-source=pgstatements \
    postgres-prod 192.168.1.30:5432
  ```

- Add PostgreSQL with `pg_stat_monitor`:

  ```bash
  pmm-admin add postgresql \
    --username=pmm \
    --password=pass \
    --query-source=pgstatmonitor \
    postgres-prod 192.168.1.30:5432
  ```

- Add PostgreSQL without query examples:

  ```bash
  pmm-admin add postgresql \
    --username=pmm \
    --password=pass \
    --query-source=pgstatmonitor \
    --disable-queryexamples \
    postgres-prod 192.168.1.30:5432
  ```

- Add PostgreSQL with TLS:

  ```bash
  pmm-admin add postgresql \
    --username=pmm \
    --password=pass \
    --tls \
    --tls-ca-file=/path/to/ca.pem \
    postgres-prod 192.168.1.30:5432
  ```

## Add MongoDB

Add a MongoDB instance to monitoring:
```bash
pmm-admin add mongodb [NAME] [ADDRESS] [FLAGS]
```

### Connection options

Connect using `--username` and `--password`.

### TLS options

Secure the connection between PMM and your MongoDB instance with TLS:

- `--tls`
:   Use TLS to connect.

- `--tls-skip-verify`
:   Skip certificate validation.

- `--tls-ca-file`
:   Path to the CA certificate file.

- `--tls-certificate-key-file`
:   Path to the combined certificate/key file.

- `--tls-certificate-key-file-password`
:   Password for the certificate/key file.

### Query Analytics options

Control how PMM collects query data for Query Analytics (QAN):

- `--query-source`
:   Source for collecting queries: `profiler` (default), `mongolog`, or `none`.

- `--max-query-length`
:   Maximum query length in QAN. Set to `-1` for unlimited, `0` for the default (4096 characters), or a specific number to truncate after that many characters. Do not set to 1, 2, or 3 as these values will cause the PMM agent to terminate.

### Collector options

Control which metrics PMM collects:

| Flag | Description | Default |
|------|-------------|---------|
| `--enable-all-collectors` | Enable all collectors | `false` |
| `--disable-collectors` | Comma-separated list of collectors to exclude | |
| `--max-collections-limit` | Max collections (-1=PMM decides, 0=unlimited) | `-1` |
| `--stats-collections` | Limit stats to specific databases/collections | |

By default, PMM enables only `diagnosticdata` and `replicasetstatus` collectors. Use `--enable-all-collectors` to enable `collstats`, `dbstats`, `indexstats`, and `topmetrics`.

!!! caution
    A very high `--max-collections-limit` value can impact CPU and memory usage. Use `--stats-collections` to limit the scope of databases and collections being monitored.

### Collector resolution

PMM collects metrics at different intervals based on collector performance:

**High resolution** (fast collectors):

- `diagnosticdata`
- `replicasetstatus`
- `topmetrics`

**Low resolution** (slower collectors):

- `dbstats`
- `indexstats`
- `collstats`

### Environment variables

Pass environment variables to the MongoDB exporter using `--agent-env-vars`. Only variables already set in the `pmm-agent` environment will be passed:

```bash
pmm-admin add mongodb \
  --username=pmm \
  --password=pass \
  --agent-env-vars="LOG_LEVEL,OTHER_VAR" \
  mongodb-prod 192.168.1.20:27017
```

### Examples

- Add MongoDB with default collectors:

  ```bash
  pmm-admin add mongodb \
    --username=pmm \
    --password=pass \
    mongodb-prod 192.168.1.20:27017
  ```

- Add MongoDB with all collectors:

  ```bash
  pmm-admin add mongodb \
    --username=pmm \
    --password=pass \
    --enable-all-collectors \
    mongodb-prod 192.168.1.20:27017
  ```

- Add MongoDB with all collectors except topmetrics:

  ```bash
  pmm-admin add mongodb \
    --username=pmm \
    --password=pass \
    --enable-all-collectors \
    --disable-collectors=topmetrics \
    mongodb-prod 192.168.1.20:27017
  ```

- Add MongoDB with collection limit:

  ```bash
  pmm-admin add mongodb \
    --username=pmm \
    --password=pass \
    --enable-all-collectors \
    --max-collections-limit=500 \
    mongodb-prod 192.168.1.20:27017
  ```

- Add MongoDB with stats for specific databases:

  ```bash
  pmm-admin add mongodb \
    --username=pmm \
    --password=pass \
    --enable-all-collectors \
    --stats-collections=db1,db2.collection1 \
    mongodb-prod 192.168.1.20:27017
  ```

  This collects stats for all collections in `db1` and only `collection1` in `db2`.

- Add MongoDB with cluster name:

  ```bash
  pmm-admin add mongodb \
    --username=pmm \
    --password=pass \
    --cluster=my-replica-set \
    mongodb-prod 192.168.1.20:27017
  ```

- Add MongoDB with unlimited collections:

  ```bash
  pmm-admin add mongodb \
    --username=pmm \
    --password=pass \
    --enable-all-collectors \
    --max-collections-limit=0 \
    mongodb-prod 192.168.1.20:27017
  ```

## Add Valkey/Redis

Add a Valkey or Redis instance to monitoring:
```bash
pmm-admin add valkey [NAME] [ADDRESS] [FLAGS]
```

### Connection options

Connect using `--username` and `--password`. Use `--tls` and `--tls-skip-verify` to secure the connection with TLS.

### Examples

- Add Valkey:
  ```bash
    pmm-admin add valkey \
      --username=pmm \
      --password=pass \
      valkey-prod 192.168.1.40:6379
  ```

- Add Valkey with TLS:
  ```bash
    pmm-admin add valkey \
      --username=pmm \
      --password=pass \
      --tls \
      --tls-skip-verify \
      valkey-prod 192.168.1.40:6379
  ```

- Add Valkey with environment labels:
  ```bash
    pmm-admin add valkey \
      --username=pmm \
      --password=pass \
      --environment=production \
      --cluster=cache-cluster \
      valkey-prod 192.168.1.40:6379
  ```

## Add ProxySQL

Add a ProxySQL instance to monitoring:
```bash
pmm-admin add proxysql [NAME] [ADDRESS] [FLAGS]
```

### Connection options

Connect using `--username` and `--password` for the ProxySQL admin interface. Use `--tls` and `--tls-skip-verify` to secure the connection.

Use `--disable-collectors` with a comma-separated list to exclude specific collectors from monitoring.

### Examples

- Add ProxySQL:
  ```bash
    pmm-admin add proxysql \
      --username=admin \
      --password=admin \
      proxysql-prod 192.168.1.50:6032
  ```

- Add ProxySQL with TLS:
  ```bash
    pmm-admin add proxysql \
      --username=admin \
      --password=admin \
      --tls \
      proxysql-prod 192.168.1.50:6032
  ```

## Add HAProxy

Add an HAProxy instance to monitoring. Unlike other services, HAProxy requires `--listen-port` to specify where metrics are exposed:

```bash
pmm-admin add haproxy [NAME] [FLAGS]
```

### Connection options

- `--listen-port` *(required)*
:   Port where HAProxy exposes metrics.

- `--scheme`
:   URI scheme, `http` or `https`.

- `--metrics-path`
:   Path to the metrics endpoint. Defaults to `/metrics`.

Optionally use `--username` and `--password` if your HAProxy metrics endpoint requires authentication.

### Examples

- Add HAProxy:
  ```bash
    pmm-admin add haproxy \
      --listen-port=8404 \
      haproxy-prod
  ```

- Add HAProxy with HTTPS:
  ```bash
    pmm-admin add haproxy \
      --listen-port=8404 \
      --scheme=https \
      --tls-skip-verify \
      haproxy-prod
  ```

- Add HAProxy with authentication:
  ```bash
    pmm-admin add haproxy \
      --listen-port=8404 \
      --username=admin \
      --password=pass \
      haproxy-prod
  ```

## Add external services

Add custom Prometheus exporters to PMM.

### External service

Add an external Prometheus exporter running on a known port:
```bash
pmm-admin add external [NAME] [ADDRESS] [FLAGS]
```

Example:

```bash
  pmm-admin add external \
    --listen-port=9104 \
    --scheme=https \
    --tls-skip-verify \
    custom-exporter
```

### External serverless

Add an external serverless service with a full URL:
```bash
pmm-admin add external-serverless [NAME] [ADDRESS] [FLAGS]
```

Example:

```bash
  pmm-admin add external-serverless \
    --url=https://host.docker.internal:9218 \
    --tls-skip-verify \
    serverless-exporter
```

## Common patterns

Use these flags with any database type to organize services, control authentication, and configure agent behavior.

### Organize services with labels

Group services by environment, cluster, and team:

```bash
pmm-admin add mysql \
  --username=pmm \
  --password=pass \
  --environment=production \
  --cluster=mysql-main \
  --replication-set=replica-set-1 \
  --custom-labels="team=backend,app=orders" \
  mysql-prod 192.168.1.10:3306
```

### Override the agent password

Set a custom password for the metrics endpoint:

```bash
pmm-admin add mysql \
  --username=pmm \
  --password=pass \
  --agent-password=custom-metrics-pass \
  mysql-prod 192.168.1.10:3306
```

!!! note
    Avoid special characters like `\`, `;`, and `$` in the agent password.

### Control metrics flow

Choose how metrics travel between agent and server:

```bash
pmm-admin add mysql \
  --username=pmm \
  --password=pass \
  --metrics-mode=push \
  mysql-prod 192.168.1.10:3306
```

#### Available modes

- `auto`: Server chooses (default)
- `push`: Agent pushes metrics to server
- `pull`: Server scrapes metrics from agent

## See also

- [Manage inventory to modify agent configurations](../pmm-admin/inventory.md)
- [Remove services from monitoring](../../remove-services.md)
- [Connect databases to PMM](../../../install-pmm/install-pmm-client/connect-database/index.md)