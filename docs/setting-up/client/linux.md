# Linux

## Adding general system metrics service

PMM collects Linux metrics automatically starting from the moment when you have configured your node for monitoring with `pmm-admin config`.


## Installing DEB packages using `apt-get`

If you are running a DEB-based Linux distribution, you can use the `apt` package manager to install PMM client from the official Percona software repository.

Percona provides `.deb` packages for 64-bit versions of popular Linux distributions.

The list can be found on [Percona’s Software Platform Lifecycle page](https://www.percona.com/services/policies/percona-software-platform-lifecycle/).

!!! note

    Although PMM client should work on other DEB-based distributions, it is tested only on the platforms listed above.

To install the PMM client package, follow these steps.

1. Configure Percona repositories using the [percona-release](https://www.percona.com/doc/percona-repo-config/percona-release.html) tool. First you’ll need to download and install the official `percona-release` package from Percona:

    ```sh
    wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
    sudo dpkg -i percona-release_latest.generic_all.deb
    ```

    !!! note

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

## Installing RPM packages using `yum`

If you are running an RPM-based Linux distribution, use the `yum` package manager to install PMM Client from the official Percona software repository.

Percona provides `.rpm` packages for 64-bit versions of Red Hat Enterprise Linux 6 (Santiago) and 7 (Maipo), including its derivatives that claim full binary compatibility, such as, CentOS, Oracle Linux, Amazon Linux AMI, and so on.

!!! note

    PMM Client should work on other RPM-based distributions, but it is tested only on RHEL and CentOS versions 6 and 7.

To install the PMM Client package, complete the following procedure. Run the following commands as root or by using the `sudo` command:

1. Configure Percona repositories using the [percona-release](https://www.percona.com/doc/percona-repo-config/percona-release.html) tool. First you’ll need to download and install the official percona-release package from Percona:

    ```sh
    sudo yum install https://repo.percona.com/yum/percona-release-latest.noarch.rpm
    ```

    !!! note

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
