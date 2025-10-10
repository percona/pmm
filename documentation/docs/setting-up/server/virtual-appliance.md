# Virtual Appliance

How to run PMM Server as a virtual machine.

!!! summary alert alert-info "Summary"
    - Download and verify the [latest][OVA] OVF file.
    - Import it.
    - Reconfigure network.
    - Start the VM and get IP.
    - Log into PMM UI.
    - (Optional) Change VM root password.
    - (Optional) Set up SSH.
    - (Optional) Set up static IP.

---

Most steps can be done with either a user interface or on the command line, but some steps can only be done in one or the other. Sections are labelled **UI** for user interface or **CLI** for command line instructions.

## Terminology

- *Host* is the desktop or server machine running the hypervisor.
- *Hypervisor* is software (e.g. [VirtualBox], [VMware]) that runs the guest OS as a virtual machine.
- *Guest* is the CentOS virtual machine that runs PMM Server.

## OVA file details

| Item          | Value
|---------------|-----------------------------------------------------------
| Download page | <https://www.percona.com/downloads/pmm2/{{release}}/ova>
| File name     | `pmm-server-{{release}}.ova`
| VM name       | `PMM2-Server-{{release_date}}-N` (`N`=build number)

## VM specifications

| Component         | Value
|-------------------|-------------------------------
| OS                | Oracle Linux 9 (64-bit)
| CPU               | 1
| Base memory       | 4096 MB
| Disks             | LVM, 2 physical volumes
| Disk 1 (`sda`)    | VMDK (SCSI, 40 GB)
| Disk 2 (`sdb`)    | VMDK (SCSI, 400 GB)

## Users

| Default Username | Default password
|------------------|-----------------------
| `root`           | `percona`
| `admin`          | `admin`

## Download

### UI

1. Open a web browser.
2. [Visit the PMM Server download page][OVA].
3. Choose a *Version* or use the default (the latest).
4. Click the link for `pmm-server-{{release}}.ova` to download it. Note where your browser saves it.
5. Right-click the link for `pmm-server-{{release}}.sha256sum` and save it in the same place as the `.ova` file.
6. (Optional) [Verify](#verify).

### CLI

Download the latest PMM Server OVA and checksum files.

```sh
wget https://www.percona.com/downloads/pmm2/{{release}}/ova/pmm-server-{{release}}.ova
wget https://www.percona.com/downloads/pmm2/{{release}}/ova/pmm-server-{{release}}.sha256sum
```

## Verify

### CLI

Verify the checksum of the downloaded .ova file.

```sh
shasum -ca 256 pmm-server-{{release}}.sha256sum
```

## VMware

### Import

#### UI

1. Select *File* → *Import*.
2. Click *Choose file...*.
3. Navigate to the downloaded `.ova` file and select it.
4. Click *Open*.
5. Click *Continue*.
6. In the *Save as* dialog:

    a. (Optional) Change the directory or file name.

    b. Click *Save*.

7. Choose one of:

    - (Optional) Click *Finish*. This starts the virtual machine.
    - (Recommended) Click *Customize Settings*. This opens the VM's settings page without starting the machine.

#### CLI

1. Install [`ovftool`][OVFTool]. (You need to register.)
2. Import and convert the OVA file. (`ovftool` can't change CPU or memory settings during import, but it can set the default interface.)

    Choose one of:

    - Download and import the OVA file.

        ```sh
        ovftool --name="PMM Server" --net:NAT=Wi-Fi \
        https://www.percona.com/downloads/pmm2/{{release}}/ova/pmm-server-{{release}}.ova \
        pmm-server-{{release}}.vmx
        ```

    - Import an already-downloaded OVA file.

        ```sh
        ovftool --name="PMM Server" --net:NAT=WiFi \
        pmm-server-{{release}}.ova \
        pmm-server.vmx
        ```

### Reconfigure interface

!!! note alert alert-primary ""
    When using the command line, the interface is remapped during import.

#### UI

1. If started, shut down the virtual machine.
2. In the VMware main window, select the imported virtual machine.
3. Click *Virtual Machine* → *Settings...*.
4. Click *Network Adapter*.
5. In the *Bridged Networking* section, select *Autodetect*.
6. Close the settings window.

### Start guest and get IP address

#### UI

1. In the VMware main window, select the imported virtual machine.
2. Click the play button <i class="uil uil-caret-right"></i> or select *Virtual Machine* → *Start Up*.
3. When the instance has been booted, note the IP address in the guest console.

#### CLI/UI

1. Start the virtual machine in GUI mode. (There's no way to redirect a VMware VM's console to the host.)

    ```sh
    vmrun -gu root -gp percona start \
    pmm-server.vmx gui
    ```

2. When the instance has been booted, note the IP address in the guest console.

3. (Optional) Stop and restart the instance in headless mode.

    ```sh
    vmrun stop pmm-server.vmx
    vmrun -gu root -gp percona start \
    pmm-server.vmx nogui
    ```

## VirtualBox

### Import

#### UI

1. Select *File* → *Import appliance...*.
2. In the *File* field, type the path to the downloaded `.ova` file, or click the folder icon to navigate and open it.
3. Click *Continue*.
4. On the *Appliance settings* page, review the settings and click *Import*.
5. Click *Start*.
6. When the guest has booted, note the IP address in the guest console.

#### CLI

1. Open a terminal and change the directory to where the downloaded `.ova` file is.

2. (Optional) Do a 'dry run' import to see what values will be used.

    ```sh
    VBoxManage import pmm-server-{{release}}.ova --dry-run
    ```

3. Import the image.
    Choose one of:
    - With the default settings.

        ```sh
        VBoxManage import pmm-server-{{release}}.ova
        ```

    - With custom settings (in this example, Name: "PMM Server", CPUs: 2, RAM: 8192 MB).

        ```sh
        VBoxManage import --vsys 0 --vmname "PMM Server" \
        --cpus 2 --memory 8192 pmm-server-{{release}}.ova
        ```

### Interface

#### UI

1. Click *Settings*.
2. Click *Network*.
3. In the *Adapter 1* field, click *Attached to* and change to *Bridged Adapter*.
4. In the *Name* field, select your host's active network interface (e.g. `en0: Wi-Fi (Wireless)`).
5. Click *OK*.

#### CLI

1. Show the list of available bridge interfaces.

    ```sh
    VBoxManage list bridgedifs
    ```

2. Find the name of the active interface you want to bridge to (one with *Status: Up* and a valid IP address). Example: `en0: Wi-Fi (Wireless)`

3. Bridge the virtual machine's first interface (`nic1`) to the host's `en0` ethernet adapter.

    ```sh
    VBoxManage modifyvm 'PMM Server' \
    --nic1 bridged --bridgeadapter1 'en0: Wi-Fi (Wireless)'
    ```

4. Redirect the console output into a host file.

    ```sh
    VBoxManage modifyvm 'PMM Server' \
    --uart1 0x3F8 4 --uartmode1 file /tmp/pmm-server-console.log
    ```

### Get IP

#### UI

1. Select the *PMM Server* virtual machine in the list.
2. Click *Start*.
3. When the guest has booted, note the IP address in the guest console.

#### CLI

1. Start the guest.

    ```sh
    VBoxManage startvm --type headless 'PMM Server'
    ```

2. (Optional) Watch the log file.

    ```sh
    tail -f /tmp/pmm-server-console.log
    ```

3. Wait for one minute for the server to boot up.

4. Choose one of:

    - Read the IP address from the tailed log file.
    - Extract the IP address from the log file.

        ```sh
        grep -e "^IP:" /tmp/pmm-server-console.log | cut -f2 -d' '
        ```

5. (Optional) Stop the guest:

    ```sh
    VBoxManage controlvm "PMM Server" poweroff
    ```

## Log into user interface

### UI

1. Open a web browser and visit the guest IP address.

2. The PMM [login screen](../../get-started/interface.md) appears.

3. Enter the default username and password in the relevant fields and click *Log in*.

    - username: `admin`

    - password: `admin`

4. (Recommended) Follow the prompts to change the default password.

!!! note alert alert-primary ""
    You also can change the default password through SSH by using the `change-admin-password` command.

5. The PMM Home Dashboard appears.

## (Optional) Change root password

### UI

1. Start the virtual machine in GUI mode.

2. Log in with the default superuser credentials:

    - Username: `root`

    - Password: `percona`

3. Follow the prompts to change the password.

## (Optional) Set up SSH

### UI/CLI

1. Create a key pair for the `admin` user.

    ```sh
    ssh-keygen -f admin
    ```

2. Log into the PMM user interface.

3. Select *PMM → PMM Settings → SSH Key*.

4. Copy and paste the contents of the `admin.pub` file into the *SSH Key* field.

5. Click *Apply SSH Key*. (This copies the public key to `/home/admin/.ssh/authorized_keys` in the guest).

6. Log in via SSH (`N.N.N.N` is the guest IP address).

    ```sh
    ssh -i admin admin@N.N.N.N
    ```

## (Optional) Set up static IP

When the guest OS starts, it will get an IP address from the hypervisor's DHCP server. This IP can change each time the guest OS is restarted. Setting a static IP for the guest OS avoids having to check the IP address whenever the guest is restarted.

### CLI

1. Start the virtual machine in non-headless (GUI) mode.

2. Log in as `root`.

3. Edit `/etc/sysconfig/network-scripts/ifcfg-eth0`

4. Change the value of `BOOTPROTO`:

    ```ini
    BOOTPROTO=none
    ```

5. Add these values:

    ```ini
    IPADDR=192.168.1.123 # replace with the desired static IP address
    NETMASK=255.255.255.0 # replace with the netmask for your IP address
    GATEWAY=192.168.1.1 # replace with the network gateway for your IP address
    PEERDNS=no
    DNS1=192.168.1.53 # replace with your DNS server IP
    ```

6. Restart the interface.

    ```sh
    ifdown eth0 && ifup eth0
    ```

7. Check the IP.

    ```sh
    ip addr show eth0
    ```
8. Preserve the network configuration across reboots.

    ```sh
    echo "network: {config: disabled}" > /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
    ```

## Remove

### UI

1. Stop the virtual machine: select *Close* → *Power Off*.

2. Remove the virtual machine: select *Remove* → *Delete all files*.

[OVA]: https://www.percona.com/downloads/pmm2/{{release}}/ova
[OVF]: https://wikipedia.org/wiki/Open_Virtualization_Format
[VirtualBox]: https://www.virtualbox.org/
[VMware]: https://www.vmware.com/products/workstation-player/
[OVFTool]: https://code.vmware.com/tool/ovf
