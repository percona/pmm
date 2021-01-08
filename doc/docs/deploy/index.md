# Deploying Percona Monitoring and Management

PMM is designed to be scalable for various environments.  If you have just one MySQL or MongoDB server, you can install and run both server and  clients on one database host.

It is more typical to have several MySQL and MongoDB server instances distributed over different hosts. In this case, you need to install the client package on each database host that you want to monitor. In this scenario, the  server is set up on a dedicated monitoring host.

In this chapter

[TOC]

## Installing PMM Server

To install and set up the PMM Server, use one of the following options:

* [Running PMM Server via Docker](server/docker.md)
* [PMM Server as a Virtual Appliance](server/virtual-appliance.md)
* [Running PMM Server Using AWS Marketplace](server/ami.md)

### Verifying PMM Server

In your browser, go to the server by its IP address. If you run your server as a virtual appliance or by using an Amazon machine image, you will need to setup the user name, password and your public key if you intend to connect to the server by using ssh. This step is not needed if you run PMM Server using Docker.

In the given example, you would need to direct your browser to *http://192.168.100.1*. Since you have not added any monitoring services yet, the site will not show any data.

You can also check if PMM Server is available requesting the /ping URL as in the following example:

```
$ curl http://192.168.100.1/ping
{'version': '1.8.0'}
```

## Installing Clients

PMM Client is a package of agents and exporters installed on a database host that you want to monitor. Before installing the PMM Client package on each database host that you intend to monitor, make sure that your PMM Server host is accessible.

For example, you can run the **ping** command passing the IP address of the computer that PMM Server is running on. For example:

```
$ ping 192.168.100.1
```

You will need to have root access on the database host where you will be installing PMM Client (either logged in as a user with root privileges or be able to run commands with **sudo**).

### Supported platforms

PMM Client should run on any modern Linux 64-bit distribution, however Percona provides PMM Client packages for automatic installation from software repositories only on the most popular Linux distributions:

* DEB packages for Debian based distributions such as Ubuntu
* RPM packages for Red Hat based distributions such as CentOS

It is recommended that you install your  client by using the software repository for your system. If this option does not work for you, Percona provides downloadable PMM Client packages from the [Download Percona Monitoring and Management](https://www.percona.com/downloads/pmm-client) page.

In addition to DEB and RPM packages, this site also offers:

* Generic tarballs that you can extract and run the included `install` script.
* Source code tarball to build your  client from source.

**WARNING**: You should not install agents on database servers that have the same host name, because host names are used by PMM Server to identify collected data.

### Storage requirements

Minimum **100** MB of storage is required for installing the PMM Client package. With a good constant connection to PMM Server, additional storage is not required. However, the client needs to store any collected data that it is not able to send over immediately, so additional storage may be required if connection is unstable or throughput is too low.

### Installing PMM Client on Debian or Ubuntu

If you are running a DEB-based Linux distribution, use the **apt** package manager to install PMM Client from the official Percona software repository.

Percona provides `.deb` packages for 64-bit versions of the following distributions:

* Debian 8 (jessie)
* Debian 9 (stretch)
* Ubuntu 14.04 LTS (Trusty Tahr)
* Ubuntu 16.04 LTS (Xenial Xerus)
* Ubuntu 16.10 (Yakkety Yak)
* Ubuntu 17.10 (Artful Aardvark)
* Ubuntu 18.04 (Bionic Beaver)

**NOTE**: PMM Client should work on other DEB-based distributions, but it is tested only on the platforms listed above.

To install the PMM Client package, complete the following procedure. Run the following commands as root or by using the **sudo** command:

1. Configure Percona repositories as described in [Percona Software Repositories Documentation](https://www.percona.com/doc/percona-repo-config/index.html).

2. Install the PMM Client package:

    ```
    $ apt-get install pmm-client
    ```

    **NOTE**: You can also download PMM Client packages from the [PMM download page](https://www.percona.com/downloads/pmm/). Choose the appropriate PMM version and your GNU/Linux distribution in two pop-up menus to get the download link (e.g. *Percona Monitoring and Management 1.17.2* and *Ubuntu 18.04 (Bionic Beaver*).

### Installing the PMM Client Package on Red Hat and CentOS

If you are running an RPM-based Linux distribution, use the **yum** package manager to install PMM Client from the official Percona software repository.

Percona provides `.rpm` packages for 64-bit versions of Red Hat Enterprise Linux 6 (Santiago) and 7 (Maipo), including its derivatives that claim full binary compatibility, such as, CentOS, Oracle Linux, Amazon Linux AMI, and so on.

**NOTE**: PMM Client should work on other RPM-based distributions, but it is tested only on RHEL and CentOS versions 6 and 7.

To install the PMM Client package, complete the following procedure. Run the following commands as root or by using the **sudo** command:

1. Configure Percona repositories as described in [Percona Software Repositories Documentation](https://www.percona.com/doc/percona-repo-config/index.html).

2. Install the `pmm-client` package:

    ```
    yum install pmm-client
    ```

    **NOTE**: You can also download PMM Client packages from the [PMM download page](https://www.percona.com/downloads/pmm/). Choose the appropriate PMM version and your GNU/Linux distribution in two pop-up menus to get the download link (e.g. *Percona Monitoring and Management 1.17.2* and *Red Hat Enterprise Linux / CentOS / Oracle Linux 7*).

## Connecting PMM Clients to the PMM Server

With your server and clients set up, you must configure each PMM Client and specify which PMM Server it should send its data to.

To connect a PMM Client, enter the IP address of the PMM Server as the value of the `--server` parameter to the **pmm-admin config** command.

Run this command as root or by using the **sudo** command

```
$ pmm-admin config --server 192.168.100.1:8080
```

For example, if your PMM Server is running on 192.168.100.1, and you have installed PMM Client on a machine with IP 192.168.200.1, run the following in the terminal of your client. Run the following commands as root or by using the **sudo** command:

```
$ pmm-admin config --server 192.168.100.1
OK, PMM server is alive.

PMM Server      | 192.168.100.1
Client Name     | ubuntu-amd641
Client Address  | 192.168.200.1
```

If you change the default port **80** when running PMM Server, specify the new port number after the IP address of PMM Server. For example:

```
$ pmm-admin config --server 192.168.100.1:8080
```

## Collecting Data from PMM Clients on PMM Server

To start collecting data on each PMM Client connected to a server, run the **pmm-admin add** command along with the name of the selected monitoring service.

Run the following commands as root or by using the **sudo** command.

Enable general system metrics, MySQL metrics, MySQL query analytics:

```
$ pmm-admin add mysql
```

Enable general system metrics, MongoDB metrics, and MongoDB query analytics:

```
$ pmm-admin add mongodb
```

Enable ProxySQL performance metrics:

```
$ pmm-admin add proxysql:metrics [NAME] [OPTIONS]
```

To see what is being monitored, run **pmm-admin list**. For example, if you enable
general OS and MongoDB metrics monitoring, the output should be similar to the
following:

```
$ pmm-admin list

...

PMM Server      | 192.168.100.1
Client Name     | ubuntu-amd64
Client Address  | 192.168.200.1
Service manager | linux-systemd

---------------- ----------- ----------- -------- ---------------- --------
SERVICE TYPE     NAME        LOCAL PORT  RUNNING  DATA SOURCE      OPTIONS
---------------- ----------- ----------- -------- ---------------- --------
linux:metrics    mongo-main  42000       YES      -
mongodb:metrics  mongo-main  42003       YES      localhost:27017
```

## Obtaining Diagnostics Data for Support

PMM Server is able to generate a set of files for enhanced diagnostics, which can be examined and/or shared with Percona Support to solve an issue faster.

Collected data are provided by the `logs.zip` service, and cover the following subjects:

* Prometheus targets
* Consul nodes, QAN API instances
* Amazon RDS and Aurora instances
* Version
* Server configuration
* Percona Toolkit commands

You can retrieve collected data from your PMM Server in a single zip archive using this URL:

```
https://<address-of-your-pmm-server>/managed/logs.zip
```

## Updating

When changing to a new version of PMM, you update the PMM Server and each PMM Client separately.

### Updating the PMM Server

**WARNING**: Currently PMM Server doesn’t support updates from 1.x to 2.0. Just install the new PMM 2 following its [official installation instructions](https://www.percona.com/doc/percona-monitoring-and-management/2.x/setting-up/server/docker.html).

The updating procedure of your PMM Server depends on the option that you selected for installing it.

If you are running PMM Server as a virtual appliance or using an Amazon Machine Image, use the Update button on the Home dashboard (see PMM Home Page) in case of available updates.

### Updating a PMM Client

**WARNING**: Currently PMM Client has no compatibility between 1.x to 2.0. Coexistence of 1.x and 2.x clients is not supported as well. If you need PMM 2.x, remove the old pmm-client package and install the new pmm2-client one following its [installation instructions](https://www.percona.com/doc/percona-monitoring-and-management/2.x/setting-up/client/index.html).

When a newer version of PMM Client becomes available, you can update to it from  the Percona software repositories:

Debian or Ubuntu

```
$ sudo apt-get update && sudo apt-get install pmm-client
```

Red Hat or CentOS

```
$ yum update pmm-client
```

If you installed your  client manually, remove it and then download and install a newer version.

## Uninstalling PMM Components

Each PMM Client and the PMM Server are removed separately. First, remove all monitored services by using the **pmm-admin remove** command (see Removing monitoring services). Then you can remove each PMM Client and the PMM Server.

### Removing the PMM Client

Remove all monitored instances as described in Removing monitoring services. Then, uninstall the **pmm-admin** package. The exact procedure of removing the PMM Client depends on the method of installation.

Run the following commands as root or by using the **sudo** command

Using YUM

```
$ yum remove pmm-client
```

Using APT

```
$ apt-get remove pmm-client
```

Manually installed RPM package

```
$ rpm -e pmm-client
```

Manually installed DEB package

```
$ dpkg -r pmm-client
```

Using the generic PMM Client tarball.

    **cd** into the directory where you extracted the tarball
    contents. Then, run the `unistall` script:

```
$ ./uninstall
```

### Removing the PMM Server

If you run your PMM Server using Docker, stop the container as follows:

```
$ docker stop pmm-server && docker rm pmm-server
```

To discard all collected data (if you do not plan to use PMM Server in the future), remove the `pmm-data` container:

```
$ docker rm pmm-data
```

If you run your PMM Server using a virtual appliance, just stop and remove it.

To terminate the PMM Server running from an Amazon machine image, run the following command in your terminal:

```
$ aws ec2 terminate-instances --instance-ids -i-XXXX-INSTANCE-ID-XXXX
```
