# Backing Up PMM Data from the Docker Container

When PMM Server is run via Docker, its data are stored in the `pmm-data` container. To avoid data loss, you can extract the data and store outside of the container.

This example demonstrates how to back up PMM data on the computer where the Docker container is run and then how to restore them.

To back up the information from `pmm-data`, you need to create a local directory with essential sub folders and then run Docker commands to copy PMM related files into it.

1. Create a backup directory and make it the current working directory. In this example, we use *pmm-data-backup* as the directory name.

    ```
    $ mkdir pmm-data-backup; cd pmm-data-backup
    ```

2. Create the essential sub directories:

    ```
    $ mkdir -p opt/prometheus
    $ mkdir -p var/lib
    ```

Run the following commands as root or by using the **sudo** command

1. Stop the docker container:

    ```
    $ docker stop pmm-server
    ```

2. Copy data from the `pmm-data` container:

    ```
    $ docker cp pmm-data:/opt/prometheus/data opt/prometheus/
    $ docker cp pmm-data:/opt/consul-data opt/
    $ docker cp pmm-data:/var/lib/mysql var/lib/
    $ docker cp pmm-data:/var/lib/grafana var/lib/
    ```

Now, your PMM data are backed up and you can start PMM Server again:

```
$ docker start pmm-server
```
