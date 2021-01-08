# Setting Up a Docker Container for PMM Server

[TOC]

A Docker image is a collection of preinstalled software which enables running a selected version of PMM Server on your computer. A Docker image is not run directly. You use it to create a Docker container for your PMM Server. When launched, the Docker container gives access to the whole functionality of PMM.

The setup begins with pulling the required Docker image. Then, you proceed by creating a special container for persistent PMM data. The last step is creating and launching the PMM Server container.

## Pulling the PMM Server Docker Image

To pull the latest version from Docker Hub:

```
$ docker pull percona/pmm-server:1
```

This step is not required if you are running PMM Server for the first time. However, it ensures that if there is an older version of the image tagged with `latest` available locally, it will be replaced by the actual latest version.

## Creating the pmm-data Container

To create a container for persistent PMM data, run the following command:

```
$ docker create \
   -v /opt/prometheus/data \
   -v /opt/consul-data \
   -v /var/lib/mysql \
   -v /var/lib/grafana \
   --name pmm-data \
   percona/pmm-server:1 /bin/true
```

**NOTE**: This container does not run, it simply exists to make sure you retain all PMM data when you upgrade to a newer PMM Server image.  Do not remove or re-create this container, unless you intend to wipe out all PMM data and start over.

The previous command does the following:

* The **docker create** command instructs the Docker daemon to create a container from an image.
* The `-v` options initialize data volumes for the container.
* The `--name` option assigns a custom name for the container that you can use to reference the container within a Docker network. In this case: `pmm-data`.
* `percona/pmm-server:1` is the name and version tag of the image to derive the container from.
* `/bin/true` is the command that the container runs.

## Creating and Launching the PMM Server Container

To create and launch PMM Server in one command, use **docker run**:

```
$ docker run -d \
   -p 80:80 \
   --volumes-from pmm-data \
   --name pmm-server \
   --restart always \
   percona/pmm-server:1
```

This command does the following:

* The **docker run** command runs a new container based on the `percona/pmm-server:1` image.
* The `-d` option starts the container in the background (detached mode).
* The `-p` option maps the port for accessing the PMM Server web UI. For example, if port **80** is not available, you can map the landing page to port 8080 using `-p 8080:80`.
* The `-v` option mounts volumes from the `pmm-data` container (see Creating the pmm-data Container).
* The `--name` option assigns a custom name to the container that you can use to reference the container within the Docker network. In this case: `pmm-server`.
* The `--restart` option defines the container’s restart policy. Setting it to `always` ensures that the Docker daemon will start the container on startup and restart it if the container exits.
* `percona/pmm-server:1` is the name and version tag of the image to derive the container from.

## Installing and using specific docker version

To install specific PMM Server version instead of the latest one, just put desired version number after the colon. Also in this scenario it may be useful to [prevent updating PMM Server via the web interface](../../glossary.option.md) with the `DISABLE_UPDATES` docker option.

For example, installing version 1.14.1 with disabled update button in the web interface would look as follows:

```
$ docker create \
   -v /opt/prometheus/data \
   -v /opt/consul-data \
   -v /var/lib/mysql \
   -v /var/lib/grafana \
   --name pmm-data \
   percona/pmm-server:1.14.1 /bin/true

$ docker run -d \
   -p 80:80 \
   --volumes-from pmm-data \
   --name pmm-server \
   -e DISABLE_UPDATES=true \
   --restart always \
   percona/pmm-server:1.14.1
```

## Additional options

When running the PMM Server, you may pass additional parameters to the **docker run** subcommand. All options that appear after the `-e` option are the additional parameters that modify the way how PMM Server operates.

The section [PMM Server Additional Options](../../glossary.option.md#pmm-glossary-pmm-server-additional-option) lists all supported additional options.
