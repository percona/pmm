# Backup and Restore

!!! caution alert alert-warning "Caution"
    Backup and restore features are a technical preview and currently only works with a MySQL database server or a MongoDB replica set cluster, backing up to an Amazon AWS S3 storage location.

!!! summary alert alert-info "Summary"
    - Enable backup features.
    - Add a storage location.
    - Satisfy preconditions:
        - For [MySQL](#mysql-backup-preconditions):
            - Confirm instance service parameters and storage location.
            - Install required packages.
        - For [MongoDB](#mongodb-backup-preconditions):
            - Install and run [Percona Backup for MongoDB] on every node in the replica set.
    - [Make a backup](#make-a-backup), or,
    - [Restore a backup](#restore-a-backup).

## Before you start

- You have an AWS S3 storage account and location details for it.

- Backup management has been enabled:

    1. Select <i class="uil uil-cog"></i> *Configuration* → <i class="uil uil-setting"></i> *Settings* → *Advanced Settings*.

    2. Activate *Backup Management*.

    3. Click *Apply changes*.

    4. Visit `<pmm-server-IP>/graph/backup` or click <i class="uil uil-history"></i> → *Backup* in the left menu bar.

        ![!](../_images/PMM_Backup_Management.jpg)

    !!! note alert alert-primary ""
        If PMM Server runs as a Docker container, enable backup features at container creation time by adding `-e ENABLE_BACKUP_MANAGEMENT=1` to your `docker run` command.

## Add a storage location

1. Select <i class="uil uil-history"></i> → *Backup* → *Storage locations*.

2. Click *Add*.

3. Fill in the form fields.

    ![!](../_images/PMM_Backup_Management_Locations_Add_Storage_Location.jpg)

    - *Name*:
    - *Description*:
    - *Type*: Choose the type of storage.
        - *S3*: Use [Amazon AWS S3]
            - *Endpoint*: The S3 backup location endpoint.
            - *Bucket Name*:
            - *Access Key*:
            - *Secret Key*:
        - *Local Client:* (Not currently implemented)
        - *Local Server:* (Not currently implemented)

4. Click *Add* to add the location or *Test* to test the connection.

## MySQL backup preconditions

- [PMM Client](../setting-up/client/index.md) is installed and running on the node.

- There is only one MySQL instance running on the node.

- MySQL is running:

    - as a service via `systemd`;

    - with the name `mysql` (to confirm, use `systemctl status mysql`);

    - from a `mysql` system user account.

- There is a `mysql` system group.

- MySQL is using the `/var/lib/mysql` directory for database storage.

- `pmm-agent` has read/write permissions to the `/var/lib/mysql` directory.

- The following packages are installed. They should be included in the `$PATH` environment variable:

    - [`xtrabackup`][PERCONA_XTRABACKUP], which includes:

        - [`xbcloud`][PERCONA_XBCLOUD];

        - [`xbstream`][PERCONA_XBSTREAM];

    - [`qpress`][PERCONA_QPRESS].

    !!! caution alert alert-warning "Important"
        The versions of each must be compatible with the installed version of MySQL.

## MongoDB backup preconditions

- [Percona Backup for MongoDB] is installed and `pbm-agent` is running on all MongoDB nodes in the replica set.

- MongoDB is a member of a replica set.

## Make a backup

1. Select <i class="uil uil-history"></i> → *Backup* → *Backup Inventory*.

2. Click <i class="uil uil-plus-square"></i> *Add*.

3. In the *Backup On Demand* dialog, enter values for:

    - *Service name*: Choose from the menu the service to back up.
    - *Vendor*: A value is automatically selected based on the service type.
    - *Backup name*: Enter a unique name for this backup.
    - *Description*: Enter a long description for this backup.
    - *Location*: Choose from the menu the [storage location](#add-a-storage-location).

4. Click *Backup*.

5. In the *Backup Inventory* pane, watch the *Status* column.

## Restore a backup

!!! note alert alert-primary ""
    For now, you can only restore a backup to the same service it was created from. I.e. a MySQL backup of service `mysql-service-1` can only be restored to a MySQL database server registered with the same service name. Future releases will give more flexibility to restore backups to other service types.

1. Select <i class="uil uil-history"></i> → *Backup* → *Backup Inventory*.

2. Find the row with the backup you want to restore.

3. In the *Actions* column for that row, click *Restore from backup*.

4. In the *Restore from backup* dialog, check the values and click *Restore*.

5. Navigate to the *Restore History* tab to check the status of the restored backup.


## Deleting a backup

1. Navigate to *Backup Management* → *Backup Inventory*.

2. Find the row with the backup you want to delete.

3. In the *Actions* column for that row, click *Delete backup*.

4. (Optional) Check "Delete from storage" to also delete the actual backup content besides just the backup register.

5. Click "Delete" to proceed.

[Amazon AWS S3]: https://aws.amazon.com/s3/
[Percona Backup for MongoDB]: https://www.percona.com/doc/percona-backup-mongodb/installation.html
[PERCONA_QPRESS]: https://www.percona.com/doc/percona-xtrabackup/LATEST/backup_scenarios/compressed_backup.html
[PERCONA_XBCLOUD]: https://www.percona.com/doc/percona-xtrabackup/2.3/xbcloud/xbcloud.html
[PERCONA_XBSTREAM]: https://www.percona.com/doc/percona-xtrabackup/2.3/xbstream/xbstream.html
[PERCONA_XTRABACKUP]: https://www.percona.com/software/mysql-database/percona-xtrabackup
