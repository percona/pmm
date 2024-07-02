# VM artifacts for PMM

## Summary

Containers are a lightweight solution to run on any platform. One such platform is Virtual Machine. There is no additional benefit to having a custom Virtual Machine image for PMM.

Modern Operation Systems adopted a new pattern of minimal VMs that are designed to run containers and are Cloud Native:

- Fedora CoreOS (FCOS)
- openSUSE MicroOS
- bottlerocket-os
- etc

Those OSes provide additional capabilities compared to the custom image:

- transactional updates
- auto-update
- init/bootstrap container or application in the image

Instead of a custom VM image, we recommend using a more advanced base VM of your choice and running PMM as a container inside. There is additional "How To" documentation with examples and Migration documentation.

## Motivation

[Currently](https://github.com/percona/pmm/blob/pmm-3.35.0/build/packer/pmm.json), we build several VM artifacts with CentOS 7 as a base. There is also work to migrate that base to EL 9 base. 
Migration to EL9 will not solve the problem but will further postpone it to a later time.

VM was designed like a container - base image (CentOS 7/EL9) + ansible roles/playbooks/tasks to provision PMM inside the image. After provisioning, there is another custom image with PMM and all the needed software that could be used to bootstrap.

As it is an additional artifact, it needs the following:

- maintenance
- support
- testing

There is an initiative and PoC that does half of the job to move from the custom VM:

- https://jira.percona.com/browse/PMM-8306
- https://github.com/percona/pmm-server/pull/343/files

It shows the possibility of running PMM in a container and gets us closer to the end Goal.

### Goals

Deprecate custom VM to:

- reduce maintenance: ansible roles, packer, pipelines
- reduce support: support one artifact instead of two
- reduce testing: update/upgrade, image validations, additional tests
- reduce cloud resources: building images, storing images
- increase speed: release testing cycle, ansible development, and maintenance

Educate users on best practices to run PMM as a container in mature and modern VM environments.

## Proposal

Deprecate PMM VM artifact as part of the product.

Current users could migrate to the container that could be run on various platforms (ECS, k8s, VMs, bare-metal).

Provide "How To" run PMM in VM documentation and migration from custom images to upstream technologies documentation.

## Design Details

Container-oriented, Cloud native VM images support init/bootstrapping technologies, such as:

- ignition
- cloud-init
- cfn-init
- etc

Documentation should include: 
- list of available VM image options and links to the upstream documentation.
- recommend using FCOS with Ignition and provide examples of how to run it for VirtualBox, AWS, GCP, and DigitalOcean:
https://docs.fedoraproject.org/en-US/fedora-coreos/stream-metadata/

Ignition config could be:

- provided in the documentation
- sourced from the Percona-owned remote endpoint (URL), for example, Portal

### Risks and Mitigations

#### Value

There will be only one artifact - containers. Remove confusion and extend the choice of the base image.

The separate base image brings:

- security
- capabilities
  - auto-update
  - transactions
- variety

Separation of concerns, base image versus functionality, enables more capabilities for both. Image updates could happen more often and be more robust. For example, transactional updates allow automatic rollbacks if [health check](https://github.com/openSUSE/health-checker) wouldn't pass.

Another value is reducing the cost of PMM products.

#### Usability

Removing additional artifacts reduces the users' scope and transfers base OS image maintenance to them.

**Mitigation**

Upstream documentation for the base VM images is rolling much faster and up to date.

Produce additional documentation for usage and migration.

`pmm-cli` could be adopted to support VM bootstrap for chosen platforms for better UX.

#### Feasibility

PoCs are done and show the feasibility of using such technology:

- https://jira.percona.com/browse/PMM-8306
- [Butane config](#poc)

#### Business viability

Telemetry shows that PMM on VM adoption is not that big, but the cost reduction could be much more significant with deprecating custom VM images.

As security first company, we need to separate concerns and not carry additional (OS) layers that don't bring new value.

**Mitigation**

For essential users, we could handhold them to the new approach.

### PoC

I have validated that approach works running it locally as well as in the Cloud.

Content of the `pmm-server-butane.yaml`:

```yaml
variant: fcos
version: 1.4.0
passwd:
  users:
    - name: core
      ssh_authorized_keys:
        - ssh-rsa AAAA...
systemd:
  units:
    - name: serial-getty@ttyS0.service
      dropins:
      - name: autologin-core.conf
        contents: |
          [Service]
          # Override Execstart in main unit
          ExecStart=
          # Add new Execstart with `-` prefix to ignore failure
          ExecStart=-/usr/sbin/agetty --autologin core --noclear %I $TERM
          TTYVTDisallocate=no
    - name: failure.service
      enabled: true
      contents: |
        [Service]
        Type=oneshot
        ExecStart=/usr/bin/false
        RemainAfterExit=yes

        [Install]
        WantedBy=multi-user.target
    - name: pmm-server.service
      enabled: true
      contents: |
        [Unit]
        Description=pmm-server
        Wants=network-online.target
        After=network-online.target

        [Service]
        Type=simple

        # set environment for this unit
        Environment=PMM_VOLUME_PATH=/var/lib/pmm-data/
        Environment=PMM_TAG=2.35.0
        Environment=PMM_IMAGE=docker.io/percona/pmm-server

        # optional env file that could override previous env settings for this unit
        EnvironmentFile=-/var/lib/pmm-data/env

        ExecStart=/usr/bin/podman run --rm --replace=true --name=%N \
                      --network=host --ulimit=host \
                      --mount=type=bind,src=${PMM_VOLUME_PATH},dst=/srv,relabel=shared \
                      --health-cmd=none --health-interval=disable \
                      ${PMM_IMAGE}:${PMM_TAG}
        ExecStop=/usr/bin/podman stop -t 10 %N
        Restart=always
        RestartSec=20

        [Install]
        Alias=%N
        WantedBy=multi-user.target
storage:
  disks:
  - # pmm-data volume
    device: /dev/disk/by-diskseq/2
    # We do not want to wipe the partition table since this is a persistent storage
    wipe_table: false
    partitions:
    - number: 1
      label: pmm-data
      # as large as possible
      size_mib: 0
      resize: true
  filesystems:
    - path: /var/lib/pmm-data
      device: /dev/disk/by-partlabel/pmm-data
      format: xfs
      # Ask Butane to generate a mount unit for us so that this filesystem
      # gets mounted in the real root.
      with_mount_unit: true

```

Convert Butane to Ignition:
```sh
podman run --interactive --rm --security-opt label=disable \
       --volume ${PWD}:/pwd --workdir /pwd quay.io/coreos/butane:release \
       --pretty --strict pmm-server-butane.yaml > pmm-server.ign
```

Resulting [Ignition config](pmm-server.ign).

#### Public Cloud

GCP:

```sh
STREAM="stable"
IGNITION_CONFIG="/absolute/path/pmm-server.ign"
VM_NAME="den-test-pmm-ignition"
DISK_NAME="den-test-pmm-data"
ZONE="us-central1-a"

gcloud compute instances create --zone=${ZONE} --tags https-server \
--metadata-from-file "user-data=${IGNITION_CONFIG}" \
--image-project "fedora-coreos-cloud" --image-family "fedora-coreos-${STREAM}" \
--create-disk "name=${DISK_NAME},size=20GB,device-name=pmm-server-data,auto-delete=no" \
"${VM_NAME}"
```
GCP will return External `IP` as outcome the command. `ssh` to that `IP`, go to `https://IP`.

#### Libvirt

**Install**

Local demo on kvm: https://docs.fedoraproject.org/en-US/fedora-coreos/provisioning-qemu/. 

I will spin up local VM and check that it works.

PMM data volume:

```sh
qemu-img create /home/dkondratenko/.local/share/libvirt/images/sdb.qcow2 20G
```

Start VM:

```sh
STREAM="stable"
IGNITION_CONFIG="/absolute/path/pmm-server.ign"
IMAGE="/home/user/.local/share/libvirt/images/fedora-coreos-37.20230205.3.0-qemu.x86_64.qcow2"
PMM_DATA="/home/user/.local/share/libvirt/images/sdb.qcow2"
VM_NAME="pmm-test-01"
VCPUS="2"
RAM_MB="2048"
DISK_GB="10"
# For x86 / aarch64,
IGNITION_DEVICE_ARG=(--qemu-commandline="-fw_cfg name=opt/com.coreos/config,file=${IGNITION_CONFIG}")

# Setup the correct SELinux label to allow access to the config
chcon --verbose --type svirt_home_t ${IGNITION_CONFIG}

virt-install --connect="qemu:///system" --name="${VM_NAME}" --vcpus="${VCPUS}" --memory="${RAM_MB}" \
        --os-variant="fedora-coreos-$STREAM" --import --graphics=none \
        --disk="size=${DISK_GB},backing_store=${IMAGE},boot.order=1" \
        --disk="serial=pmm-server-data,path=${PMM_DATA},boot.order=2" \
        --network bridge=virbr0 "${IGNITION_DEVICE_ARG[@]}"
```

Check `IP` address on a VM and go to the PMM UI in the browser using it.

**Update**

Let me demonstrate image replacement.

Stop old VM and detach persistent storage:

```sh
virsh --connect qemu:///system shutdown pmm-test-02
virsh --connect qemu:///system detach-disk pmm-test-02 --persistent vdb
```

Spin new VM with new image and old storage:

```sh
STREAM="testing"
IGNITION_CONFIG="/absolute/path/pmm-server.ign"
IMAGE="/home/user/.local/share/libvirt/images/fedora-coreos-37.20230218.2.0-qemu.x86_64.qcow2"
PMM_DATA="/home/user/.local/share/libvirt/images/sdb.qcow2"
VM_NAME="pmm-test-02"
VCPUS="2"
RAM_MB="2048"
DISK_GB="10"
# For x86 / aarch64,
IGNITION_DEVICE_ARG=(--qemu-commandline="-fw_cfg name=opt/com.coreos/config,file=${IGNITION_CONFIG}")

# Setup the correct SELinux label to allow access to the config
chcon --verbose --type svirt_home_t ${IGNITION_CONFIG}

virt-install --connect="qemu:///system" --name="${VM_NAME}" --vcpus="${VCPUS}" --memory="${RAM_MB}" \
        --os-variant="fedora-coreos-$STREAM" --import --graphics=none \
        --disk="size=${DISK_GB},backing_store=${IMAGE},boot.order=1" \
        --disk="serial=pmm-server-data,path=${PMM_DATA},boot.order=2" \
        --network bridge=virbr0 "${IGNITION_DEVICE_ARG[@]}"
```

Check `IP` address on a VM and got to the PMM UI in the browser using it. Validate that old and new data is present.

#### Going beyond the PoC

Podman was used just for the demo and could also be docker.

There is no need to bind volume. Persistent volume should be mounted to the correct path so docker state and volumes would be saved there automatically.

Ignition has a lot of features, so there could be:

- additional service to re-size volume if it is expanded
- configuration files that change the behavior of the container (envs)
- auto-rollback
- etc

## Drawbacks

### AWS Marketplace

Custom PMM image could be easily uploaded to the AWS Marketplace.

But updates happen less often than it is required. For example, CVE in a base image would need to wait for the next PMM release.

**Mitigation**

AWS Marketplace supports [CloudFormation](https://docs.aws.amazon.com/marketplace/latest/userguide/cloudformation.html), with a similar init mechanism (`cfn-init`). So there could be an AWS Marketplace presence if we change the instrument from a custom image to the CloudFormation Template.

Deprecating AWS Marketplace presence or offloading it to the partners is another way to mitigate this problem.

### Migration

It should be reasonably straightforward, as PMM data should be stored in a separate volume. There could be some issues with different users/groups, which could be documented or automated with Ignition.

Documentation about migration from the custom PMM image to the cloud-native VM with a container should be developed.

## Alternatives

TBD

