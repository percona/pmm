
# Back up PMM Server Docker container

Regular backups of your PMM Server are essential for protecting your monitoring configuration and historical data, especially before migrations and upgrades.
    
## Back up Grafana plugins 
Grafana plugins have been moved to the `/srv` directory since PMM 2.23.0. So if you are upgrading PMM from a version before 2.23.0 and have installed additional plugins, you'll need to reinstall them after the upgrade.
    
To check used Grafana plugins:

```sh
docker exec -t pmm-server ls -l /var/lib/grafana/plugins
```

## Backup procedure
To back up your PMM Server container, follow the backup instructions for your deployment type.

### Step 1: Back up PMM Server container

Identify your deployment type and storage method since different PMM Server deployments store data differently:

=== "Docker with named volumes"
    Your PMM data is stored in a Docker-managed volume (like pmm-data). You'll need to copy this volume's contents to create a backup.

    **Command to check mount configuration:**
    ```sh
    docker inspect pmm-server | grep -A 10 '"Mounts"'
    ```

    **Expected output:**
    ```json
    "Mounts": [
        {
            "Type": "volume",
            "Name": "pmm-data",
            "Source": "/var/lib/docker/volumes/pmm-data/_data",
            "Destination": "/srv",
            "Driver": "local",
            "Mode": "",
            "RW": true,
            "Propagation": ""
        }
    ]
    ```

=== "Docker with host directories"
    Your PMM data is stored in a directory on your host machine that's mounted into the container. You'll back up this directory using standard file system tools.

    **Command to check mount configuration:**
    ```sh
    docker inspect pmm-server | grep -A 10 '"Mounts"'
    ```

    **Expected output:**
    ```json
    "Mounts": [
        {
            "Type": "bind",
            "Source": "/home/user/srv",
            "Destination": "/srv",
            "Mode": "",
            "RW": true,
            "Propagation": "rprivate"
        }
    ]
    ```

=== "Podman with SystemD"
    Your PMM runs as a SystemD service using Podman. You'll need to stop the service, back up the volume, and restart the service.

    **Command to check if PMM service is running:**
    ```sh
    systemctl --user is-active pmm-server
    ```

    **Expected output:**
    ```
    active
    ```

    **Alternative command to check Podman containers:**
    ```sh
    podman ps --format "table {{.Names}}\t{{.Status}}"
    ```

    **Expected output:**
    ```
    NAMES        STATUS
    pmm-server   Up 2 hours
    ```

=== "Kubernetes"
    Your PMM runs in a Kubernetes cluster. You'll use volume snapshots or persistent volume backups.

    **Command to check if PMM pods are running:**
    ```sh
    kubectl get pods -l app.kubernetes.io/name=pmm
    ```

    **Expected output:**
    ```
    NAME    READY   STATUS    RESTARTS   AGE
    pmm-0   1/1     Running   0          2d
    ```

    **Command to check persistent volume claims:**
    ```sh
    kubectl get pvc -l app.kubernetes.io/name=pmm
    ```

    **Expected output:**
    ```
    NAME               STATUS   VOLUME                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
    pmm-storage-pmm-0  Bound    pvc-abc123-def4-5678-9012  8Gi        RWO            gp2            2d
    ```

### Step 2: Choose a backup method 
Choose the appropriate backup method based on your PMM Server deployment:

=== "Docker with named volume"
    This is the most common deployment pattern and is ideal for migrations, as it preserves the entire `pmm-data` volume structure required for a successful transfer. Use this method if your PMM Server is deployed with a named Docker volume:

    **Example deployment**
    ```sh
    docker run --detach --restart always \
      --publish 443:8443 \
      --volume pmm-data:/srv \
      --name pmm-server \
      percona/pmm-server:3
    ```

    **Create volume backup**
    ```sh
    # Stop PMM Server
    docker stop pmm-server

    # Create backup volume with timestamp
    BACKUP_VOLUME="pmm-data-backup-$(date +%Y%m%d-%H%M%S)"
    docker volume create $BACKUP_VOLUME 1> /dev/null

    # Copy data from current volume to backup volume
    docker run --rm -v pmm-data:/from -v $BACKUP_VOLUME:/to alpine ash -c 'cd /from ; cp -av . /to'

    # Verify backup
    docker run --rm -v $BACKUP_VOLUME:/backup alpine ls -la /backup

    # Restart PMM Server
    docker start pmm-server

    # Note backup volume name for recovery
    echo "Backup volume created: $BACKUP_VOLUME"
    ```

    **Alternative: Export volume to archive:**
    ```sh
    # Create compressed backup archive
    mkdir -p pmm-volume-backups
    docker run --rm -v pmm-data:/volume -v $(pwd)/pmm-volume-backups:/backup alpine tar czf /backup/pmm-data-backup-$(date +%Y%m%d-%H%M%S).tar.gz -C /volume .
    ```

=== "Docker with host directory"
    Use this method if your PMM Server mounts a host directory:

    **Example deployment**
    ```sh
    docker run --detach --restart always \
      --publish 443:8443 \
      --volume /home/user/srv:/srv \
      --name pmm-server \
      percona/pmm-server:3
    ```

    **Create host directory backup**
    ```sh
    # Stop PMM Server
    docker stop pmm-server

    # Create backup directory
    BACKUP_DIR="pmm-directory-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p $BACKUP_DIR

    # Copy mounted directory (adjust path to match your deployment)
    cp -r /home/user/srv $BACKUP_DIR/

    # Or use rsync for incremental backup
    rsync -av /home/user/srv/ $BACKUP_DIR/

    # Restart PMM Server
    docker start pmm-server
    ```

=== "Podman with systemD"
    Use this method if your PMM Server runs with Podman and SystemD:

    **Example deployment via SystemD service**
    ```sh
    # Podman with named volume via SystemD
    systemctl --user status pmm-server
    ```

    **Create Podman volume backup**
    ```sh
    # Stop PMM Server service
    systemctl --user stop pmm-server

    # Create backup volume with timestamp
    BACKUP_VOLUME="pmm-data-backup-$(date +%Y%m%d-%H%M%S)"
    podman volume create $BACKUP_VOLUME

    # Copy data from current volume to backup volume
    podman run --rm -v pmm-server:/from -v $BACKUP_VOLUME:/to alpine ash -c 'cp -av /from/. /to'

    # Verify backup
    podman run --rm -v $BACKUP_VOLUME:/backup alpine ls -la /backup

    # Restart PMM Server service
    systemctl --user start pmm-server

    # Note backup volume name for restoration
    echo "Backup volume created: $BACKUP_VOLUME"
    ```

=== "Kubernetes (Helm)"
    Use this method if your PMM Server runs on Kubernetes via Helm. Requires `StorageClass` and `VolumeSnapshotClass` that support snapshots. Check with your Kubernetes provider for availability.

    **Example deployment**
    ```sh
    helm install pmm percona/pmm
    ```

    **Create Kubernetes volume snapshot**
    ```sh
    # Check available storage classes and snapshot classes
    kubectl get storageclass
    kubectl get volumesnapshotclass

    # Create volume snapshot
    cat <<EOF | kubectl apply -f -
    apiVersion: snapshot.storage.k8s.io/v1
    kind: VolumeSnapshot
    metadata:
      name: pmm-backup-$(date +%Y%m%d-%H%M%S)
      labels:
        app.kubernetes.io/name: pmm
    spec:
      source:
        persistentVolumeClaimName: pmm-storage-pmm-0
      volumeSnapshotClassName: your-snapshot-class
    EOF

    # Verify snapshot creation
    kubectl get volumesnapshot -l app.kubernetes.io/name=pmm
    ```

=== "Universal container copy"

    While this method works universally, the deployment-specific methods above are more efficient and preserve storage structures better. This method works for all deployment types as a fallback option when you're unsure about your deployment type or you need a quick backup without determining volume setup.
    {.power-number}

    1. Stop the running PMM Server container:

        ```sh
        docker stop pmm-server
        ```

    2. Rename the container to preserve it as a backup source:

        ```sh
        docker rename pmm-server pmm-server-backup
        ```

    3. Create a backup subdirectory and navigate to it:

        ```sh
        mkdir pmm-data-backup-$(date +%Y%m%d-%H%M%S) && cd pmm-data-backup-$(date +%Y%m%d-%H%M%S)
        ```

    4. Back up the data:

        ```sh
        docker cp pmm-server-backup:/srv/. .
        ```

    5. Verify the backup was created successfully:

        ```sh
        ls -la .
        ```

    6. Create new container from original image:

        ```sh
        docker run -d -v pmm-data:/srv -p 443:8443 --name pmm-server --restart always percona/pmm-server:3
        ```

### Step 3: Verify the integrity of the backup

=== "Volume backups" 
    For backups created using the Named Volume method:

    ```sh
    # Check backup volume contents
    docker run --rm -v $BACKUP_VOLUME:/backup alpine ls -la /backup

    # Verify critical directories
    docker run --rm -v $BACKUP_VOLUME:/backup alpine \
      ash -c 'ls -la /backup/grafana /backup/prometheus /backup/clickhouse 2>/dev/null || echo "Some directories may not exist in older versions"'
    ```

=== "Directory backups" 
    For directory-based backups:

    ```sh
    # Check backup directory
    ls -la $BACKUP_DIR/

    # Verify critical subdirectories
    ls -la $BACKUP_DIR/{grafana,prometheus,clickhouse} 2>/dev/null || echo "Some directories may not exist in older versions"
    ```

=== "Podman volumes"

    For backups created using Podman volumes:

    ```sh
    # Check backup volume contents
    podman run --rm -v $BACKUP_VOLUME:/backup alpine ls -la /backup

    # Verify critical directories exist
    podman run --rm -v $BACKUP_VOLUME:/backup alpine ls -la /backup/grafana /backup/prometheus /backup/clickhouse 2>/dev/null || echo "Some directories may not exist in older versions"
    ```

=== "Kubernetes snapshots"

    For backups created using Kubernetes volume snapshots:

    ```sh
    # Check snapshot status
    kubectl get volumesnapshot -l app.kubernetes.io/name=pmm

    # Verify snapshot is ready
    kubectl get volumesnapshot pmm-backup-YYYYMMDD-HHMMSS -o jsonpath='{.status.readyToUse}'

    # Check snapshot size and source
    kubectl describe volumesnapshot pmm-backup-YYYYMMDD-HHMMSS
    ```

## Next steps after backup  

After creating your backup, you have two options:
{.power-number}

1. Resume normal operations if you were creating a routine backup, restart your original container.
2. [Upgrade](../docker/upgrade_container.md) or [restore the container](../docker/restore_container.md) if you were backing up before an upgrade or restoration.

## Backup storage recommendations

- Store backups in a location separate from the PMM Server host
- Implement automated rotation of backups to manage disk space
- Consider encrypting backups containing sensitive monitoring data
- Test restores periodically to verify backup integrity