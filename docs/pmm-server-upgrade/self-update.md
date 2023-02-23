# Self-update of pmm-server-upgrade

`pmm-server-upgrade` supports self-updating.  
The process consists of the following steps:

1. In regular intervals `pmm-server-upgrade` triggers a self-update
2. The latest Docker image of `pmm-server-upgrade` is downloaded
3. If it's different than the currently running version, self-update starts
4. The current process stops the API server to stop listening on the unix socket file
5. New container is started
6. It is expected the new container starts listening on the unix socket file
7. Once the new container is healthy, the current container shuts down and disables restart policy on itself
8. In case of an error, the new container is stopped and a unix socket file is restored by the current process
