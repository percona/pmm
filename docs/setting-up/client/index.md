# Setting up PMM Clients

PMM Client is a package of agents and exporters installed on a database host
that you want to monitor. Before installing the PMM Client package on each
database host that you intend to monitor, make sure that your PMM Server host
is accessible.

For example, you can run the `ping` command passing the IP address of the
computer that PMM Server is running on. For example:

```sh
ping 192.168.100.1
```

You will need to have root access on the database host where you will be
installing PMM Client (either logged in as a user with root privileges or be
able to run commands with `sudo`).

**Supported platforms**

PMM Client should run on any modern Linux 64-bit distribution, however
Percona provides PMM Client packages for automatic installation from
software repositories only on the most popular [Linux](linux/) distributions.

It is recommended that you install your PMM (Percona Monitoring and Management) client by using the
software repository for your system. If this option does not work for you,
Percona provides downloadable PMM Client packages
from the [Download Percona Monitoring and Management](https://www.percona.com/downloads/pmm2/) page.

In addition to DEB and RPM packages, this site also offers:

* Generic tarballs that you can extract and run the included `install` script.
* Source code tarball to build your PMM (Percona Monitoring and Management) client from source.

!!! warning

    You should not install agents on database servers that have the same host name, because host names are used by PMM Server to identify collected data.

**Storage requirements**

Minimum 100 MB of storage is required for installing the PMM Client package. With a good constant connection to PMM Server, additional storage is not required. However, the client needs to store any collected data that it is not able to send over immediately, so additional storage may be required if connection is unstable or throughput is too low.



## Connecting PMM Clients to the PMM Server

With your server and clients set up, you must configure each PMM Client and
specify which PMM Server it should send its data to.

To connect a PMM Client, enter the IP address of the PMM Server as the value
of the `--server-url` parameter to the `pmm-admin config` command, and
allow using self-signed certificates with `--server-insecure-tls`.

!!! note

    The `--server-url` argument should include `https://` prefix and PMM Server credentials, which are `admin`/`admin` by default, if not changed at first PMM Server GUI access.

Run this command as root or by using the `sudo` command

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.100.1:443
```

For example, if your PMM Server is running on 192.168.100.1, you have installed PMM Client on a machine with IP 192.168.200.1, and didn’t change default PMM Server credentials, run the following in the terminal of your client. Run the following commands as root or by using the `sudo` command:

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.100.1:443
```

```
Checking local pmm-agent status...
pmm-agent is running.
Registering pmm-agent on PMM Server...
Registered.
Configuration file /usr/local/percona/pmm-agent.yaml updated.
Reloading pmm-agent configuration...
Configuration reloaded.
Checking local pmm-agent status...
pmm-agent is running.
```

If you change the default port 443 when running PMM Server, specify the new port number after the IP address of PMM Server.

!!! note

    By default `pmm-admin config` refuses to add client if it already exists in the PMM Server inventory database. If you need to re-add an already existing client (e.g. after full reinstall, hostname changes, etc.), you can run `pmm-admin config` with the additional `--force` option. This will remove an existing node with the same name, if any, and all its dependent services.

## Removing monitoring services with `pmm-admin remove`

Use the `pmm-admin remove` command to remove monitoring services.

### USAGE

Run this command as root or by using the `sudo` command

```sh
pmm-admin remove [OPTIONS] [SERVICE-TYPE] [SERVICE-NAME]
```

When you remove a service,
collected data remains in Metrics Monitor on PMM Server for the specified [retention period](https://www.percona.com/doc/percona-monitoring-and-management/2.x/faq.html#how-to-control-data-retention-for-pmm).

### SERVICES

Service type can be mysql, mongodb, postgresql or proxysql, and service
name is a monitoring service alias. To see which services are enabled,
run `pmm-admin list`.

### EXAMPLES

```sh
# Removing MySQL service named mysql-sl
pmm-admin remove mysql mysql-sl

# remove MongoDB service named mongo
pmm-admin remove mongodb mongo

# remove PostgreSQL service named postgres
pmm-admin remove postgresql postgres

# remove ProxySQL service named ubuntu-proxysql
pmm-admin remove proxysql ubuntu-proxysql
```

For more information, run `pmm-admin remove --help`.
