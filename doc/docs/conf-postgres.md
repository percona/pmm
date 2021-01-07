# Configuring PostgreSQL for Monitoring

Monitoring PostgreSQL metrics with the [postgres_exporter](https://github.com/wrouesnel/postgres_exporter) is enabled by `pmm-admin add postgresql` command. The `postgresql` alias will set up `postgresql:metrics` and also `linux:metrics` on a host (for more information, see [Adding monitoring services](pmm-admin.md#pmm-admin-add)).

`pmm-admin` supports passing PostgreSQL connection information via following flags:

| Flag         | Description         |
| ------------ | ------------------- |
| `--host`     | PostgreSQL host     |
| `--password` | PostgreSQL password |
| `--port`     | PostgreSQL port     |
| `--user`     | PostgreSQL user     |

An example command line would look like this:

```
pmm-admin add postgresql --host=localhost --password='secret' --port=5432 --user=pmm_user
```

**NOTE**: Capturing read and write time statistics is possible only if `track_io_timing` setting is enabled. This can be done either in configuration file or with the following query executed on the running system:

```
ALTER SYSTEM SET track_io_timing=ON;
SELECT pg_reload_conf();
```

## Supported versions of PostgreSQL

PMM follows [postgresql.org EOL policy](https://www.postgresql.org/support/versioning/), and thus supports monitoring PostgreSQL version 9.4 and up.  Older versions may work, but will not be supported.  For additional assistance, visit [Percona PMM Forums](https://www.percona.com/forums/questions-discussions/percona-monitoring-and-management/).

### Setting Up the Required Permissions

Percona recommends that a PostgreSQL user be configured for `SUPERUSER` level access, in order to gather the maximum amount of data with a minimum amount of complexity. This can be done with the following command for the standalone PostgreSQL installation:

```
CREATE USER pmm_user WITH SUPERUSER ENCRYPTED PASSWORD 'secret';
```

**NOTE**: In case of monitoring a PostgreSQL database running on an Amazon RDS instance, the command should look as follows:

```
CREATE USER pmm_user WITH rds_superuser ENCRYPTED PASSWORD 'secret';
```
