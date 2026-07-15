# MongoDB Backup and Restore support matrix

## PBM version compatibility

| PBM version | PMM support |
| --- | --- |
| Earlier than 2.0.1 | No |
| 2.0.1 to 2.9.x | Full |
| 2.10.0 and later | Full |

PMM requires Percona Backup for MongoDB (PBM) 2.0.1 or later. PBM 2.10.0 and newer are supported by PMM versions that include the PMM-14576 backup and restore polling improvements.

Check the [PBM compatibility matrix](https://docs.percona.com/percona-backup-mongodb/details/versions.html) to select a PBM version that supports your MongoDB or Percona Server for MongoDB version.

Creating and restoring MongoDB backups in PMM currently has the following limitations and requirements:

- Physical backups and restores are supported only for **Percona Server for MongoDB**.
- Physical restores are not supported for deployments with arbiter nodes. For more information, see the [Percona Backup for MongoDB documentation](https://docs.percona.com/percona-backup-mongodb/usage/restore.html#physical-restore-known-limitations).
- Creating backups for sharded clusters is available straight from the UI. However, restoring these backup artifacts is only possible via the CLI, using Percona Backup for MongoDB. For information on restoring sharded backups, check the [PBM documentation](https://docs.percona.com/percona-backup-mongodb/usage/restore.html).
- Retention policy is supported only for snapshot types of scheduled backups and for the S3-compatible storage type.
- PMM can restore a backup only when the target service runs the same MongoDB version recorded in the backup artifact.
- Before restoring, make sure to prevent clients from accessing the database.

## Backup and restore support matrix

## Backup: Logical

| Full or PITR | Storage type (S3 or Local) | Support level |                                                                    
| ---- | -------- | ------------- |
| PITR  | S3       | <b style="color:#5794f2;"><b style="color:#5794f2;">Full</b></b>                                  |                   
| PITR  | Local    | <b style="color:#5794f2;">Full</b>                                    |
| Full   | S3      | <b style="color:#5794f2;">Full</b>                                    |                                               
| Full   | Local   | <b style="color:#5794f2;">Full</b>                                    |


## Backup: Physical
| Full or PITR | Storage type (S3 or Local) | Support level |                                                                    
| ---- | -------- | ------------- |
| PITR  | S3       | <b style="color:#e36526;">No</b>                                       
| PITR  | Local    | <b style="color:#e36526;">No</b>                                       
| Full   | S3      | <b style="color:#5794f2;">Full</b>                                   
| Full   | Local   | <b style="color:#5794f2;">Full</b>                                    


## Restore: Logical
| Full or PITR | Storage type (S3 or Local) | Support level |                                                                    
| ---- | -------- | ------------- |
| PITR  | S3       | <b style="color:#5794f2;">Full</b>                                    |                                               
| PITR  | Local    | <b style="color:#e36526;">No</b>                                      |
| Full   | S3       | <b style="color:#5794f2;">Full</b>                                    |                                               
| Full   | Local    | <b style="color:#5794f2;">Full</b>                                    |                                               

## Restore: Physical
| Full or PITR | Storage type (S3 or Local) | Support level|                                                                    
| ---- | -------- | ------------- |
| PITR  | S3       | <b style="color:#e36526;">No</b>                        |            
| PITR  | Local    | <b style="color:#e36526;">No</b>                        |             
| Full   | S3       | <b style="color:#e36526;">Partial*</b> |                                    
| Full   | Local    | <b style="color:#e36526;">Partial*</b> |         

\* Partial support for non-containerized deployments and NO support for containerized deployments.                             

