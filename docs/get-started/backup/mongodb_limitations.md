# MongoDB Backup and Restore support matrix

Creating and restoring MongoDB backups in PMM currently has the following limitations and requirements:

- Restoring on different Replica set/Cluster is not supported.
- Physical backups and restores suppoted only for **Percona Server for MongoDB**
- Physical restores are not supported for deployments with arbiter nodes. For more information, see the [Percona Backup for MongoDB documentation](https://docs.percona.com/percona-backup-mongodb/usage/restore.html#physical-restore-known-limitations).
- All types of backups on sharded cluster setups are currently not supported.
- Retention policy is supported only for snapshot types of scheduled backups and for the S3-compatible storage type.
- Before restoring, make sure to prevent clients from accessing database.
  
## Support matrix

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
