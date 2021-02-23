# Setting up PMM Clients

[TOC]

---

PMM Client is a package of agents and exporters installed on the host you wish to monitor.

Before installing, know your PMM Server's IP address and make sure that it is accessible.

You will need root access on the database host where you install PMM Client (either logged in as a user with root privileges or have `sudo` rights).

!!! alert alert-info "Note"

    Credentials used in communication between the exporters and the PMM Server are the following ones:

    * login is `pmm`
    * password is equal to Agent ID, which can be seen e.g. on the Inventory Dashboard.

## Supported platforms

PMM Client should run on any modern Red Hat or Debian-based 64-bit Linux distribution, but is only tested on:

- RHEL/CentOS 6, 7, 8
- Debian 8, 9, 10
- Ubuntu 16.04, 18.04, 20.04

We recommended installing PMM Client via your system's package management tool, using the software repository provided by Percona for popular Linux distributions.

If this option does not work for you, Percona provides downloadable PMM Client packages from the [Download Percona Monitoring and Management](https://www.percona.com/downloads/pmm2/) page. As well as DEB and RPM packages, you will also find:

- generic tarballs that you can extract and run the included `install` script;
- source code tarball to build the PMM client from source.

## Storage requirements

A minimum of 100 MB of storage is required for installing the PMM Client package. With a good constant connection to PMM Server, additional storage is not required. However, the client needs to store any collected data that it is not able to send over immediately, so additional storage may be required if connection is unstable or throughput is too low.

## Installing PMM Client with your Linux package manager

### Using `apt-get` (Debian/Ubuntu)

1. Configure Percona repositories using the [percona-release](https://www.percona.com/doc/percona-repo-config/percona-release.html) tool. First you’ll need to download and install the official `percona-release` package from Percona:

    ```sh
    wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
    sudo dpkg -i percona-release_latest.generic_all.deb
    ```

    !!! alert alert-info "Note"

        If you have previously enabled the experimental or testing Percona repository, don’t forget to disable them and enable the release component of the original repository as follows:

        ```sh
        sudo percona-release disable all
        sudo percona-release enable original release
        ```

2. Install the PMM client package:

    ```sh
    sudo apt-get update
    sudo apt-get install pmm2-client
    ```

3. Register your Node:

    ```sh
    pmm-admin config --server-insecure-tls --server-url=https://admin:admin@<IP Address>:443
    ```

4. You should see the following output:

    ```
    Checking local pmm-agent status...
    pmm-agent is running.
    Registering pmm-agent on PMM Server...
    Registered.
    Configuration file /usr/local/percona/pmm-agent.yaml updated.
    Reloading pmm-agent configuration...
    Configuration reloaded.
    ```

### Using `yum` (Red Hat/CentOS)

1. Configure Percona repositories using the [percona-release](https://www.percona.com/doc/percona-repo-config/percona-release.html) tool. First you’ll need to download and install the official `percona-release` package from Percona:

    ```sh
    sudo yum install https://repo.percona.com/yum/percona-release-latest.noarch.rpm
    ```

    !!! alert alert-info "Note"

        If you have previously enabled the experimental or testing Percona repository, don’t forget to disable them and enable the release component of the original repository as follows:

        ```sh
        sudo percona-release disable all
        sudo percona-release enable original release
        ```

        See [percona-release official documentation](https://www.percona.com/doc/percona-repo-config/percona-release.html) for details.


2. Install the `pmm2-client` package:

    ```sh
    yum install pmm2-client
    ```

3. Once PMM Client is installed, run the `pmm-admin config` command with your PMM Server IP address to register your Node within the Server:

    ```sh
    pmm-admin config --server-insecure-tls --server-url=https://admin:admin@<IP Address>:443
    ```

    You should see the following:

    ```
    Checking local pmm-agent status...
    pmm-agent is running.
    Registering pmm-agent on PMM Server...
    Registered.
    Configuration file /usr/local/percona/pmm-agent.yaml updated.
    Reloading pmm-agent configuration...
    Configuration reloaded.
    ```

## Connecting PMM Clients to PMM Server

With your server and clients set up, you must configure each PMM Client and
specify which PMM Server it should send its data to.

To connect a PMM Client, use this command, replacing `X.X.X.X` with the IP address of your PMM Server.

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
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

!!! alert alert-info "Notes"
    - The `--server-url` argument should include `https://` prefix and PMM Server credentials, which are `admin`/`admin` by default, if not changed at first PMM Server GUI access.
	- If you change the default port 443 when running PMM Server, specify the new port number after the IP address of PMM Server.
    - By default `pmm-admin config` refuses to add client if it already exists in the PMM Server inventory database. If you need to re-add an already existing client (e.g. after full reinstall, hostname changes, etc.), you can run `pmm-admin config` with the additional `--force` option. This will remove an existing node with the same name, if any, and all its dependent services.

By default, the node name is the host name. If you have non-unique client host names, specify the node name when adding the client:

```sh
pmm-admin add TYPE [options] NODE-NAME
```

## Removing monitoring services with `pmm-admin remove`

Use the `pmm-admin remove` command to remove monitoring services.

**USAGE**

Run this command as root or by using the `sudo` command

```sh
pmm-admin remove [OPTIONS] [SERVICE-TYPE] [SERVICE-NAME]
```

When you remove a service, collected data remains in Metrics Monitor on PMM Server for the specified [retention period](../../faq.md#how-to-control-data-retention-for-pmm).

**SERVICES**

Service type can be `mysql`, `mongodb`, `postgresql` or `proxysql`, and service
name is a monitoring service alias. To see which services are enabled,
run `pmm-admin list`.

**EXAMPLES**

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


!!! seealso
    - [pmm-admin](../../details/commands/pmm-admin.md)
    - [Percona Tools Supported Platforms](https://www.percona.com/services/policies/percona-software-support-lifecycle#pt/).
