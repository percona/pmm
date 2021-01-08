# Updating PMM Server Using Docker

To check the version of PMM Server, run **docker ps** on the host.

Run the following commands as root or by using the **sudo** command

```
$ docker ps
CONTAINER ID   IMAGE                      COMMAND                CREATED       STATUS             PORTS                               NAMES
480696cd4187   percona/pmm-server:1.4.0   "/opt/entrypoint.sh"   4 weeks ago   Up About an hour   192.168.100.1:80->80/tcp, 443/tcp   pmm-server
```

The version number is visible in the Image column. For a Docker container created from the image tagged `latest`, the Image column contains `latest` and not the specific version number of PMM Server.

The information about the currently installed version of PMM Server is available from the `/srv/update/main.yml` file. You may extract the version number by using the **docker exec** command:

```
$ docker exec -it pmm-server head -1 /srv/update/main.yml
# v1.5.3
```

To check if there exists a newer version of PMM Server, visit [percona/pmm-server](https://hub.docker.com/r/percona/pmm-server/tags/).

## Creating a backup version of the current pmm-server Docker container

You need to create a backup version of the current `pmm-server` container if the update procedure does not complete successfully or if you decide not to upgrade your PMM Server after trying the new version.

The **docker stop** command stops the currently running `pmm-server` container:

```
$ docker stop pmm-server
```

The following command simply renames the current `pmm-server` container to avoid name conflicts during the update procedure:

```
$ docker rename pmm-server pmm-server-backup
```

## Pulling a new Docker Image

Docker images for all versions of PMM are available from [percona/pmm-server](https://hub.docker.com/r/percona/pmm-server/tags/) Docker repository.

When pulling a newer Docker image, you may either use a specific version number or the `latest` image which always matches the highest version number.

This example shows how to pull a specific version:

```
$ docker pull percona/pmm-server:1.5.0
```

This example shows how to pull the `latest` version:

```
$ docker pull percona/pmm-server:1
```

## Creating a new Docker container based on the new image

After you have pulled a new version of PMM from the Docker repository, you can use **docker run** to create a `pmm-server` container using the new image.

```
$ docker run -d \
   -p 80:80 \
   --volumes-from pmm-data \
   --name pmm-server \
   --restart always \
   percona/pmm-server:1
```

The **docker run** command refers to the pulled image as the last parameter. If you used a specific version number when running **docker pull** (see [Pulling the PMM Server Docker Image](docker.setting-up.md#pmm-server-docker-image-pulling)) replace `latest` accordingly.

Note that this command also refers to `pmm-data` as the value of `--volumes-from` option. This way, your new version will continue to use the existing data.

**WARNING**: Do not remove the `pmm-data` container when updating, if you want to keep all collected data.

Check if the new container is running using **docker ps**.

```
$ docker ps
CONTAINER ID   IMAGE                      COMMAND                CREATED         STATUS         PORTS                               NAMES
480696cd4187   percona/pmm-server:1.5.0   "/opt/entrypoint.sh"   4 minutes ago   Up 4 minutes   192.168.100.1:80->80/tcp, 443/tcp   pmm-server
```

Then, make sure that the PMM version has been updated (see [PMM Version](../../glossary.terminology.md#pmm-version)) by checking the PMM Server web interface.

## Removing the backup container

After you have tried the features of the new version, you may decide to continue using it. The backup container that you have stored (Creating a backup version of the current pmm-server Docker container) is no longer needed in this case.

To remove this backup container, you need the **docker rm** command:

```
$ docker rm pmm-server-backup
```

As the parameter to **docker rm**, supply the tag name of your backup container.

### Restoring the previous version

If, for whatever reason, you decide to keep using the old version, you just need to stop and remove the new `pmm-server` container.

```
$ docker stop pmm-server && docker rm pmm-server
```

Now, rename the `pmm-server-backup` to `pmm-server` (see Creating a backup version of the current pmm-server Docker container) and start it.

```
$ docker start pmm-server
```

**WARNING**: Do not use the **docker run** command to start the container. The **docker run** command creates and then runs a new container.

To start a new container use the **docker start** command.
