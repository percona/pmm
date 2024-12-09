# Supported setups for MySQL backups

!!! caution alert alert-warning "Important"
    MySQL backup functionality is still in Technical Preview.
    
PMM supports MySQL database server for:
    
  - Creating and restoring physical backups
  - Storing backups to Amazon S3-compatible object storage  

## Backing up MySQL databases hosted in Docker container

To ensure PMM can correctly backup and restore databases from a MySQL Docker container, make sure that the container is compatible with systemd.