# Install PMM Server on the virtual machine

Deploy PMM Server as a virtual appliance using the PMM OVA file. Compatible with VirtualBox, VMware, and other hypervisors.  

??? "Summary"
    - Download and verify the [latest](https://www.percona.com/downloads) OVF file.
    - Import the OVA into your hypervisor.
    - Reconfigure network settings.
    - Start the VM and get its IP address.
    - Log into the PMM web interface.
    - (Optional) Change VM root password.
    - (Optional) Enable SSH access.
     - (Optional) Set a static IP address

Most steps can be done from either the UI or on the command line, but some steps can only be done in one or the other. Sections are labelled **UI** for user interface or **CLI** for command line instructions.

## Terminology

- **Host** is the desktop or server machine running the hypervisor.
- **Hypervisor**:  software (e.g. VirtualBox, VMware) that runs the guest OS as a virtual machine.
- **Guest VM** - Virtual machine running PMM Server (Oracle Linux 9.3).  


## OVA file details

| Item          | Value
|---------------|-----------------------------------------------------------
| Download page | <https://www.percona.com/downloads/pmm/{{release}}/ova>
| File name     | `pmm-server-{{release}}.ova`
| VM name       | `pmm-Server-{{release_date}}-N` (`N`=build number)

## VM specifications

| Component         | Value
|-------------------|-------------------------------
| OS                | Oracle Linux 9.3
| CPU               | 1
| Base memory       | 4096 MB
| Disks             | LVM, 2 physical volumes
| Disk 1 (`sda`)    | VMDK (SCSI, 40 GB)
| Disk 2 (`sdb`)    | VMDK (SCSI, 400 GB)

## Default Users

| Default username | Default password
|------------------|-----------------------
| `root`           | `percona`
| `admin`          | `admin`



