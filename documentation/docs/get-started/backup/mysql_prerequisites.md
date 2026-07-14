# MySQL backup prerequisites

Before creating MySQL backups, make sure to:

1. Check that **Backup Management** is enabled and the <i class="uil uil-history"></i> Backup option is available on the side menu. If Backup Managemt has been disabled on your instance, go to {{icon.configuration}} **Configuration > PMM Settings > Advanced Settings**, re-enable **Backup Management** then click **Apply changes**.

   !!! caution alert alert-warning "Important"
    If PMM Server runs as a Docker container, enable backup features at container creation time by adding `-e ENABLE_BACKUP_MANAGEMENT=1` to your `docker run` command.

2. Check that the [PMM Client](../../setting-up/client/index.md) is installed and running on the node where the backup will be performed.

3. To enable Xtrabackup for MySQL 8.0+, check that pmm-agent connects to MySQL with a user that has BACKUP_ADMIN privilege.

4. Check that there is only one MySQL instance running on the node.

5. Verify that MySQL is running:

    - as a service via `systemd`;

    - with the name `mysql` or `mysqld` (to confirm, use `systemctl status mysql` or `systemctl status mysqld` respectively);

    - from a `mysql` system user account.

6. Make sure that there is a `mysql` system group.

7. Check that MySQL is using the `/var/lib/mysql` directory for database storage.

8.  Make sure that `pmm-agent` has read/write permissions to the `/var/lib/mysql` directory.

9. Check that the latest versions of the following packages are installed and included in the `$PATH` environment variable:

    - [`xtrabackup`](https://www.percona.com/software/mysql-database/percona-xtrabackup), which includes:

        - [`xbcloud`](https://www.percona.com/doc/percona-xtrabackup/2.3/xbcloud/xbcloud.html)

        - [`xbstream`](https://www.percona.com/doc/percona-xtrabackup/2.3/xbstream/xbstream.html)

    - [`qpress`][PERCONA_QPRESS].

!!! caution alert alert-warning "Important"
       Make sure that the versions of xtrabackup, xbcloud, xbstream, and qpress are fully compatible with the currently installed version of MySQL on the system. 
