# Set up PMM Client

There are different ways to install PMM Client on a node and register it with PMM Server. Choose from:

- [Docker](#docker): Run PMM Client as a Docker container.

- [Package manager](#package-manager):
    - On Debian or Red Hat Linux, install `percona-release` and use a Linux package manager (`apt`/`dnf`) to install PMM Client.
    - On Debian or Red Hat, download `.deb`/`.rpm` PMM Client packages and manually install them.

!!! hint alert "Binary is only way to install PMM client without root permissions"
    - [Binary package](#binary-package): For other Linux distributions, download and unpack generic PMM Client Linux binaries.

When you have installed PMM Client, you must:

- [Register the node with PMM Server](#register).
- [Configure and add services according to type](#add-services).

If you need to, you can [unregister](#unregister), [remove services](#remove-services) or [remove PMM Client](#remove).

---

Here's an overview of the choices.

![!image](../../images/PMM_Client_Setup.png)

## Before you start

- [Set up PMM Server](../server/index.md) with a known IP address accessible from the client node.
- You have superuser (root) access on the client host.
- You have superuser access to any database servers that you want to monitor.
- These Linux packages are installed: `curl`, `gnupg`, `sudo`, `wget`.
- If using it, install [Docker].
- System requirements:
    - Operating system -- PMM Client runs on any modern 64-bit Linux distribution. It is tested on supported versions of Debian, Ubuntu, CentOS, and Red Hat Enterprise Linux. (See [Percona software support life cycle]).
    - Disk -- A minimum of 100 MB of storage is required for installing the PMM Client package. 
    
    With a good connection to PMM Server, additional storage is not required. However, the client needs to store any collected data that it cannot dispatch immediately, so additional storage may be required if the connection is unstable or the throughput is low. VMagent uses 1 GB of disk space for cache during a network outage. QAN, on the other hand, uses RAM to store cache.

## Install

### Docker

The [PMM Client Docker image] is a convenient way to run PMM Client as a preconfigured [Docker] container.

1. Pull the PMM Client docker image:

    ```sh
    docker pull \
    percona/pmm-client:2
    ```

2. Use the image as a template to create a persistent data store that preserves local data when the image is updated:

    ```sh
    docker create \
    --volume /srv \
    --name pmm-client-data \
    percona/pmm-client:2 /bin/true
    ```

3. Run the container to start [PMM Agent](../../details/commands/pmm-agent.md) in setup mode. Set `X.X.X.X` to the IP address of your PMM Server. Do not use the `--detach` flag as PMM Agent only outputs logs to the console:

    ```sh
    PMM_SERVER=X.X.X.X:443
    docker run \
    --rm \
    --name pmm-client \
    -e PMM_AGENT_SERVER_ADDRESS=${PMM_SERVER} \
    -e PMM_AGENT_SERVER_USERNAME=admin \
    -e PMM_AGENT_SERVER_PASSWORD=admin \
    -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
    -e PMM_AGENT_SETUP=1 \
    -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
    --volumes-from pmm-client-data \
    percona/pmm-client:2
    ```
    
    !!! hint alert-success "Tips"
        You can find a complete list of compatible environment variables [here](../../details/commands/pmm-agent.md).

4. Check status:

    ```sh
    docker exec pmm-client \
    pmm-admin status
    ```

    In the PMM user interface you will also see an increase in the number of monitored nodes.

You can now add services with [`pmm-admin`](../../details/commands/pmm-admin.md) by prefixing commands with `docker exec pmm-client`.

!!! hint alert alert-success "Tips"
    - Adjust host firewall and routing rules to allow Docker communications. ([Read more](../../how-to/troubleshoot.md))
    - For help: `docker run --rm percona/pmm-client:2 --help`


    In the GUI.

    - Select {{icon.dashboards}} *PMM Dashboards* → {{icon.node}} *System (Node)* → {{icon.node}} *Node Overview*.
    - In the *Node Names* menu, select the new node.
    - Change the time range to see data.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.

### Package manager

!!! hint alert alert-success "Tip"
    If you have used `percona-release` before, disable and re-enable the repository:

    ```sh
    percona-release disable all
    percona-release enable original release
    ```

=== "Debian-based"

    1. Configure repositories: 
       ```sh
       wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
       dpkg -i percona-release_latest.generic_all.deb
       ```

    2. Install the PMM Client package:
       !!! hint "Root permissions"
           ```sh
           apt update
           apt install -y pmm2-client
           ```

    3. Check:
       ```sh
       pmm-admin --version
       ```

    4. [Register the node](#register).

=== "Red Hat-based"

    1. Configure repositories:
       ```sh
       yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
       ```

    2. Install the PMM Client package:
       ```sh
       yum install -y pmm2-client
       ```

    3. Check:
       ```sh
       pmm-admin --version
       ```

    4. [Register the node](#register).

### Package manager -- manual download

1. Visit the [Percona Monitoring and Management 2 download] page.
2. Under *Version:*, select the one you want (usually the latest).
3. Under *Software:*, select the item matching your software platform.
4. Click to download the package file:

    - For Debian, Ubuntu: `.deb`
    - For Red Hat, CentOS, Oracle Linux: `.rpm`

(Alternatively, copy the link and use `wget` to download it.)

Here are the download page links for each supported platform.

- [Debian 9 (Stretch)]
- [Debian 10 (Buster)]
- [Debian 11 (Bullseye)]
- [Red Hat/CentOS/Oracle 7]
- [Red Hat/CentOS/Oracle 8]
- [Ubuntu 18.04 (Bionic Beaver)]
- [Ubuntu 20.04 (Focal Fossa)]
- [Ubuntu 22.04 (Jammy Jellyfish)]

=== "Debian-based"

    ```sh
    dpkg -i *.deb
    ```

=== "Red Hat-based"

    ```sh
    dnf localinstall *.rpm
    ```

### Binary package

1. Download the PMM Client package:

    ```sh
    wget https://downloads.percona.com/downloads/pmm2/{{release}}/binary/tarball/pmm2-client-{{release}}-x86_64.tar.gz
    ```

2. Download the PMM Client package checksum file:

    ```sh
    wget https://downloads.percona.com/downloads/pmm2/{{release}}/binary/tarball/pmm2-client-{{release}}-x86_64.tar.gz.sha256sum
    ```

3. Verify the download:

    ```sh
    sha256sum -c pmm2-client-{{release}}-x86_64.tar.gz.sha256sum
    ```

4. Unpack the package and move into the directory:

    ```sh
    tar xfz pmm2-client-{{release}}-x86_64.tar.gz && cd pmm2-client-{{release}}
    ```

5. Choose one of these two commands (depends on your permissions):

    !!! caution alert alert-warning "Without root permissions"
        ```sh
        export PMM_DIR=YOURPATH
        ```
        where YOURPATH replace with you real path, where you have required access.

    !!! caution alert alert-warning "With root permissions"
        ```sh
        export PMM_DIR=/usr/local/percona/pmm2
        ```

6. Run the installer:

    !!! hint "Root permissions (if you skipped step 5 for non root users)"
        ```sh
        ./install_tarball
        ```

7. Change the path:

    ```sh
    PATH=$PATH:$PMM_DIR/bin
    ```

8. Set up the agent (pick the command for you depending on permissions):

    !!! hint "Root permissions"
    ```sh
    pmm-agent setup --config-file=/usr/local/percona/pmm2/config/pmm-agent.yaml --server-address=192.168.1.123 --server-insecure-tls --server-username=admin --server-password=admin
    ```

    !!! caution alert alert-warning "Non root users"
    ```sh
    pmm-agent setup --config-file=${PMM_DIR}/config/pmm-agent.yaml --server-address=192.168.1.123 --server-insecure-tls --server-username=admin --server-password=admin --paths-tempdir=${PMM_DIR}/tmp --paths-base=${PMM_DIR}
    ```

9. Run the agent:

    ```sh
    pmm-agent --config-file=${PMM_DIR}/config/pmm-agent.yaml
    ```

10. Open a new terminal and check:

    ```sh
    pmm-admin status
    ```
    
    !!! hint "PMM-Agent can be updated from tarball"

        1. Download tar.gz with pmm2-client.
        2. Extract it.
        3. Run ./install_tarball script with the "-u" flag.

        The configuration file will be overwritten if you do not provide the "-u" flag while the pmm-agent is updated.

## Register

Register your client node with PMM Server:

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
```

- `X.X.X.X` is the address of your PMM Server.
- `443` is the default port number.
- `admin`/`admin` is the default PMM username and password. This is the same account you use to log into the PMM user interface, which you had the option to change when first logging in.

!!! caution alert alert-warning "Important"
    Clients *must* be registered with the PMM Server using a secure channel. If you use http as your server URL, PMM will try to connect via https on port 443. If a TLS connection can't be established you will get an error and you must use https along with the appropriate secure port.

### Examples

Register on PMM Server with IP address `192.168.33.14` using the default `admin/admin` username and password, a node with IP address `192.168.33.23`, type `generic`, and name `mynode`.

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.33.14:443 192.168.33.23 generic mynode
```

Register your client node running on Docker with PMM Server.

```sh
docker exec pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
```

## Add services

You must configure and adding services according to the service type.

- [MySQL](mysql.md) (and variants Percona Server for MySQL, Percona XtraDB Cluster, MariaDB)
- [MongoDB](mongodb.md)
- [PostgreSQL](postgresql.md)
- [ProxySQL](proxysql.md)
- [Amazon RDS](aws.md)
- [Microsoft Azure](azure.md)
- [Google Cloud Platform](google) (MySQL and PostgreSQL)
- [Linux](linux.md)
- [External services](external.md)
- [HAProxy](haproxy.md)
- [Remote instances](remote.md)

!!! hint alert alert-success "Tip"
    To change the parameters of a previously-added service, remove the service and re-add it with new parameters.

## Remove

How to remove (uninstall) PMM Client.

### Docker

!!! caution alert alert-warning "Caution"
    These steps delete the PMM Client Docker image and client services configuration data.

1. Stop pmm-client container:

    ```sh
    docker stop pmm-client
    ```

2. Remove containers:

    ```sh
    docker rm pmm-client
    ```

3. Remove the image:

    ```sh
    docker rmi $(docker images | grep "percona/pmm-client" | awk {'print $3'})
    ```

4. Remove the volume:

    ```sh
    docker volume rm pmm-client-data
    ```

### Package manager

=== "Debian-based distributions"

    1. Uninstall the PMM Client package:
       ```sh
       apt remove -y pmm2-client
       ```

    2. Remove the Percona repository:
       ```sh
       dpkg -r percona-release
       ```

=== "Red Hat-based distributions"

    1. Uninstall the PMM Client package:
       ```sh
       yum remove -y pmm2-client
       ```

    2. Remove the Percona repository:
       ```sh
       yum remove -y percona-release
       ```

## Unregister

Unregister PMM Client from PMM Server:

```sh
pmm-admin unregister --force
```

All services monitored by this node will be removed from monitoring.

## Remove services

Specify the service type and service name to remove services from monitoring:

```sh
pmm-admin remove <service-type> <service-name>
```

`service-type`
: One of `mysql`, `mongodb`, `postgresql`, `proxysql`, `haproxy`, `external`.

!!! seealso alert alert-info "See also"
    - [Percona release]
    - [PMM Client architecture](../../details/architecture.md#pmm-client)

[Debian 12(Bookworm)]: https://www.percona.com/downloads/pmm2/{{release}}/binary/debian/bookworm/
[Debian 11 (Bullseye)]: https://www.percona.com/downloads/pmm2/{{release}}/binary/debian/bullseye/
[Debian 10 (Buster)]: https://www.percona.com/downloads/pmm2/{{release}}/binary/debian/buster/
[Docker]: https://docs.docker.com/get-docker/
[Percona Monitoring and Management 2 download]: https://www.percona.com/downloads/pmm2/
[Percona release]: https://www.percona.com/doc/percona-repo-config/percona-release.html
[Percona software support life cycle]: https://www.percona.com/services/policies/percona-software-support-lifecycle#pt
[PMM Client Docker image]: https://hub.docker.com/r/percona/pmm-client/tags/
[Red Hat/CentOS/Oracle 7]: https://www.percona.com/downloads/pmm2/{{release}}/binary/redhat/7/
[Red Hat/CentOS/Oracle 8]: https://www.percona.com/downloads/pmm2/{{release}}/binary/redhat/8/
[Ubuntu 20.04 (Focal Fossa)]: https://www.percona.com/downloads/pmm2/{{release}}/binary/debian/focal/
[Ubuntu 22.04 (Jammy Jellyfish)]: https://www.percona.com/downloads/pmm2/{{release}}/binary/debian/jammy/
