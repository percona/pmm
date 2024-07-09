# Update of PMM Server

## Update process

Update of PMM Server which includes `managed` and other components is triggered by sending a [StartUpdate](https://github.com/percona/pmm/blob/6761010b8b30042936c58c022752f6b57581afee/api/serverpb/server.proto#L325) message.
This performs the following actions:
1. Runs [pmm-update](https://github.com/percona/pmm/tree/main/update) command to initiate an update
2. `pmm-update` first updates itself to the latest version and restarts
3. `pmm-update` then runs a set of Ansible tasks to update all other components

**Notes**
- `pmm-update` does not handle rollbacks in case of errors
- `pmm-update` requires root priveleges to run

## Testing a custom pmm-update build

When making changes to `pmm-update`, you can test if they work in the following way:

1. Install an older version of PMM Server to trigger an option to upgrade. This can be achieved either by:
    1. Installing an older version or
    2. Enabling experimental repo which already has an RC build available. Run these commands in the docker container:
        ```sh
        sed -i -e 's^/release/^/experimental/^' /etc/yum.repos.d/pmm2-server.repo
        percona-release enable pmm2-client experimental
        yum makecache
        ```
2. Create a new rpm package with the updated `pmm-update`. Refer to [Building RPM package](#building-rpm-package) section below.
3. Copy the rpm package and enable the `local` repo. The `local` repo is available in the container by default and points to `/tmp/RPMS`.
    ```sh
    mkdir -p /tmp/RPMS
    cp </path/to/pmm-update.rpm> /tmp/RPMS
    createrepo /tmp/RPMS/
    yum-config-manager --enable local
    ```
    
    The rpm file is usually in `/root/rpmbuild/RPMS/pmm-update/noarch/`
4. You can now trigger an update in the UI and it will install the latest `pmm-update` package

## Building RPM package

All steps are performed in the docker container.

1. Install dependencies
    ```sh
    yum install -y \
        make gcc wget curl \
        rpmdevtools createrepo rpm-build yum-utils
    ```
2. Install go https://go.dev/doc/install
3. From the pmm repo, copy `build/packages/rpm/server/SPECS/pmm-update.spec` file to the container
4. Change the `pmm-update.spec` file:
    1. Update version to some high number, eg
        ```
        %define full_pmm_version 150.0.0
        ```
    2. Update commit hash to the hash of your `pmm-update` version. For this you need to have the commit already available in https://github.com/percona/pmm
        ```
        %global commit 592eddf656bce32a11bd958af0a32c62bd5ea34c
        ```
5. Build the rpm package
    ```sh
    mkdir -p /root/rpmbuild/SOURCES
    spectool -C /root/rpmbuild/SOURCES/ -g pmm-update.spec
    rpmbuild --define '_rpmdir %{_topdir}/RPMS/pmm-update' --define 'dist .el7' -ba pmm-update.spec
    ```
