# MySQL backup prerequisites

Before creating MySQL backups:
{.power-number}

1. Enable **Backup Management** and confirm the <i class="uil uil-history"></i> **Backup** option is available on the side menu. If **Backup Management** is disabled, go to :material-cog: **Configuration > Settings > Advanced settings**, enable **Backup Management**, and click **Apply changes**.

    !!! caution alert alert-warning "Important"
        If PMM Server runs as a Docker container, enable backup features at container creation time by adding `-e ENABLE_BACKUP_MANAGEMENT=1` to your `docker run` command.

2. Install and run the [PMM Client](../../install-pmm/install-pmm-client/index.md) on the node.

3. To enable XtraBackup for MySQL 8.0+, connect `pmm-agent` to MySQL using a user with the `BACKUP_ADMIN` privilege.

4. Run only one MySQL instance on the node.

5. Verify that MySQL is running:

    - as a service via `systemd`
    - with the name `mysql` or `mysqld` (run `systemctl status mysql` or `systemctl status mysqld` to confirm)
    - from a `mysql` system user account

6. Verify that a `mysql` system group exists.

7. Verify that MySQL uses `/var/lib/mysql` for database storage.

8. Grant `pmm-agent` read/write permissions to `/var/lib/mysql`.

9. Install the latest versions of the following packages and add them to `$PATH`. For XtraBackup version requirements, see [XtraBackup and MySQL version compatibility](#xtrabackup-and-mysql-version-compatibility):

    - [`xtrabackup`](https://docs.percona.com/percona-xtrabackup/) (includes [`xbcloud`](https://www.percona.com/doc/percona-xtrabackup/2.3/xbcloud/xbcloud.html) and [`xbstream`](https://www.percona.com/doc/percona-xtrabackup/2.3/xbstream/xbstream.html))
    - [`qpress`][PERCONA_QPRESS]

## XtraBackup and MySQL version compatibility

When installing `xtrabackup`, use a version of [Percona XtraBackup (PXB)](https://docs.percona.com/percona-xtrabackup/) that matches your MySQL version:

- MySQL 5.5, 5.6, 5.7 — PXB 2.4.x
- MySQL 8.0.0–8.0.33 — PXB 8.0.x (same version or newer)
- MySQL 8.0.34+ — PXB 8.0.34+
- MySQL 8.1.x, 8.2.x, 8.3.x — matching PXB version (supports only the matching MySQL Innovation release)
- MySQL 8.4.x — any PXB 8.4.x release (supports MySQL 8.4 LTS, including future patch releases; does not support MySQL 8.0 or 9.x)