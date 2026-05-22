# PMM MySQL Backup Compose

Single-container local setup: PMM Server with its **builtin pmm-agent**, plus MySQL and backup tools on the same host.

The custom server image extends `perconalab/pmm-server-fb:PR-4368-377dfd9` and adds:

- Percona Server for MySQL `8.4.6-6` (`mysqld`)
- `percona-xtrabackup-84` version `8.4.0-4` (`xtrabackup`, `xbcloud`, `xbstream`)
- `qpress`

PMM checks backup software on the node where the agent runs. The server image already runs a local `pmm-agent` (supervisord, id `pmm-server`), so MySQL is registered on `127.0.0.1:3306` inside the same container.

## Start

From this directory:

```sh
make env-up
```

Or from the repository root:

```sh
make -C dev/mysql-backup-compose env-up
```

The default tested versions are:

- PMM Server base image: `perconalab/pmm-server-fb:PR-4368-377dfd9`
- MySQL RPM: `percona-server-server-8.4.6-6.1.el9`
- XtraBackup RPM: `percona-xtrabackup-84-8.4.0-4.1.el9`

Override with `PMM_SERVER_IMAGE`, `MYSQL_SERVER_RPM_VERSION`, and `XTRABACKUP_RPM_VERSION` if needed.

PMM is available at <https://localhost/> with `admin` / `admin`.

After startup, the `mysql-backup-register` supervisord job registers the MySQL service as `mysql-backup`. Registration can take 1–2 minutes after PMM Server becomes healthy (waits for `pmm-agent` and `mysqld`).

Check status:

```sh
make env-status
```

If `mysql-backup` is missing, inspect logs and re-register:

```sh
make env-logs
make env-register
```

## Run A Backup From UI

Create an S3 backup location in the PMM UI using your own S3-compatible storage, then start a physical MySQL backup for the `mysql-backup` service.

## Stop

```sh
make env-down
```

## Notes

- MySQL datadir: `/srv/mysql-data` (inside the `pmm-data` volume).
- This is a local development setup, not a production deployment.
- MySQL backup jobs use S3 storage. Configure your S3 location manually in PMM before starting a backup.
