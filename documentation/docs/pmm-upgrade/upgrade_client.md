# Upgrade PMM Client

There are two primary methods to update PMM Clients, depending on your initial installation method:
{.power-number}

1. Using your operating system's package manager
2. Updating from a tarball

### 1. Package Manager method

The package manager method is generally more convenient and efficient. Percona provides the [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) package, which helps you install Percona software, including PMM Client. PMM Client is available from the `pmm-client` repository.

To deploy a new version of the Client via package manager, simply replace the currently installed package with the latest version of the PMM Client or with a specific version.

#### Install the latest PMM Client version

Run the commands below to install the latest PMM Client version via package manager and keep your existing Client configuration during the update process.

For example, to install the latest version of the PMM Client on Red Hat or its derivatives:

=== "Debian-based"

    ```sh
    percona-release enable pmm3-client
    apt update
    apt install pmm-client
    ```
=== "Red Hat-based"

    ```sh
    percona-release enable pmm3-client
    yum update pmm-client
    ```

#### Deploy a specific version

To deploy a specific version of the PMM Client via package manager, check the available versions and then provide the full name of the package. For example:

=== "Red Hat-based"
    ```sh
    yum --showduplicates search pmm-client
    pmm-client-3.0.0-6.el9.x86_64 : Percona Monitoring and Management Client (pmm-agent)
    pmm-client-3.0.1-6.el9.x86_64 : Percona Monitoring and Management Client (pmm-agent)
    yum update pmm-client-3.0.1-6.el9.x86_64
    ```

=== "Debian-based"
    ```sh
    apt-cache madison pmm-client
    pmm-client | 3.1.0-6.jammy | http://repo.percona.com/pmm-client/apt jammy/main amd64 Packages
    pmm-client | 3.0.0-6.jammy | http://repo.percona.com/pmm-client/apt jammy/main amd64 Packages
    apt install pmm-client=3.0.1-6.jammy
    ```

### 2. Tarball method

If you initially installed the PMM Client from a tarball, you can update it by replacing the currently installed package with the latest version:
{.power-number}

 1. [Download](https://www.percona.com/downloads) `tar.gz` with `pmm-client`.
 2. Extract the tarball.
 3. Run `./install_tarball` script with the `-u` flag.

!!! caution alert alert-warning "Important"
    The configuration file will be overwritten if you do not provide the `-u` flag while the `pmm-agent` is updated.
