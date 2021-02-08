# Virtual Appliance

---

[TOC]

---

Percona provides a *virtual appliance* for running PMM Server in a virtual machine.  It is distributed as an *Open Virtual Appliance* (OVA) package, which is a `tar` archive with necessary files that follow the *Open Virtualization Format* (OVF).  OVF is supported by most popular virtualization platforms:

## Supported Platforms for Running the PMM Server Virtual Appliance

The virtual appliance is ideal for running PMM Server on an enterprise virtualization platform of your choice. This page explains how to run the appliance in VirtualBox and VMware Workstation Player. which is a good choice to experiment with PMM at a smaller scale on a local machine.  Similar procedure should work for other platforms (including enterprise deployments on VMware ESXi, for example), but additional steps may be required.

The virtual machine used for the appliance runs CentOS 7.

!!! alert alert-info "Note"
    The appliance must run in a network with DHCP, which will automatically assign an IP address for it. To assign a static IP manually, you need to acquire the root access.

## VirtualBox Using the Command Line

Instead of using the VirtualBox GUI, you can do everything on the command line. Use the `VBoxManage` command to import, configure, and start the appliance.

The following script imports the PMM Server appliance from `pmm-server-{{release}}.ova` and configures it to bridge the en0 adapter from the host.  Then the script routes console output from the appliance to `/tmp/pmm-server-console.log`.  This is done because the script then starts the appliance in headless (without the console) mode.

To get the IP address for accessing PMM, the script waits for 1 minute until the appliance boots up and returns the lines with the IP address from the log file.

```sh
# Import image
VBoxManage import pmm-server-{{release}}.ova

# Modify NIC settings if needed
VBoxManage list bridgedifs
VBoxManage modifyvm 'PMM Server {{release}}' --nic1 bridged --bridgeadapter1 'en0: Wi-Fi (AirPort)'

# Log console output into file
VBoxManage modifyvm 'PMM Server {{release}}' --uart1 0x3F8 4 --uartmode1 file /tmp/pmm-server-console.log

# Start instance
VBoxManage startvm --type headless 'PMM Server {{release}}'

# Wait for 1 minute and get IP address from the log
sleep 60
grep "IP:" /tmp/pmm-server-console.log
```

By convention **OVA** files start with `pmm-server-` followed by the full version number such as {{release}}.

To use this script, make sure to replace this placeholder with the the name of the image that you have downloaded from the [PMM download](https://www.percona.com/downloads/pmm2/{{release}}/ova) site.

## VirtualBox Using the GUI

The following procedure describes how to run the PMM Server appliance using the graphical user interface of VirtualBox:

1. Download the OVA. The latest version is available at [https://www.percona.com/downloads/pmm2/{{release}}/ova](https://www.percona.com/downloads/pmm2/{{release}}/ova).

2. Import the appliance. For this, open the *File* menu and click *Import Appliance* and specify the path to the OVA and click *Continue*. Then, select *Reinitialize the MAC address of all network cards* and click *Import*.

3. Configure network settings to make the appliance accessible from other hosts in your network.

    !!! alert alert-info
        All database hosts must be in the same network as PMM Server, so do not set the network adapter to NAT.

    If you are running the appliance on a host with properly configured network settings, select *Bridged Adapter* in the *Network* section of the
appliance settings.

4. Start the PMM Server appliance.

    If it was assigned an IP address on the network by DHCP, the URL for accessing PMM will be printed in the console window.

## VMware Workstation Player

The following procedure describes how to run the *PMM Server* appliance using VMware Workstation Player:

1. Download the OVA. The latest version is available at [https://www.percona.com/downloads/pmm2/{{release}}/ova](https://www.percona.com/downloads/pmm2/{{release}}/ova).

2. Import the appliance.

    1. Open the *File* menu and click *Open*.
    2. Specify the path to the OVA and click *Continue*.

    !!! alert alert-info
        You may get an error indicating that import failed. Click *Retry* and the import should succeed.

3. Configure network settings to make the appliance accessible from other hosts in your network.

    If you are running the appliance on a host with properly configured network settings, select **Bridged** in the **Network connection** section of the appliance settings.

4. Start the PMM Server appliance.

    Log in as `root`, password `percona` and follow the prompts to change the password.

## Identifying PMM Server IP Address

PMM Server uses DHCP for security reasons. Use this command in the PMM Server console to find out the server's IP address:

```sh
grep "IP:" /tmp/pmm-server-console.log
```

## Accessing PMM Server

1. Start the virtual machine
2. Open a web browser
3. Enter the server's IP address
4. Enter the user login and password to access the PMM Server web interface

![image](../../_images/PMM_Login.jpg)

If you run PMM Server in your browser for the first time, you are requested to supply the user login and password. The default PMM Server credentials are:

* **username:** admin
* **password:** admin

After login you will be proposed to change this default password. Enter the new password twice and click *Save*. The PMM Server is now ready and the home page opens.

![image](../../_images/PMM_Home_Dashboard.jpg)

You are creating a username and password that will be used for two purposes:

1. authentication as a user to PMM - the credentials to log in to PMM.

2. authentication between PMM Server and PMM Clients - you will re-use these credentials as a part of the server URL when configuring PMM Client for the first time on a server:

    Run this command as root or by using the `sudo` command

    ```sh
    pmm-admin config --server-insecure-tls --server-url=https://admin:admin@<IP Address>:443
    ```

## Accessing the Virtual Machine

To access the VM with the *PMM Server* appliance via SSH, you will need to provide your public key:

1. Open the URL for accessing PMM in a web browser. The URL is provided either in the console window or in the appliance log.

2. Go to *PMM > PMM Settings > SSH Key*.

3. Enter your **public key** in the *SSH Key* field and click the *Apply SSH Key* button.


After that you can use `ssh` to log in as the `admin` user. For example, if *PMM Server* is running at `192.168.100.1` and your **private key** is `~/.ssh/pmm-admin.key`, use the following command:

```sh
ssh admin@192.168.100.1 -i ~/.ssh/pmm-admin.key
```

## Next Steps

Verify that PMM Server is running by connecting to the PMM web interface using the IP address assigned to the virtual appliance, then [install PMM Client](../client/index.md) on all database hosts that you want to monitor.
