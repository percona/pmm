# VirtualBox Using the Command Line

Instead of using the VirtualBox GUI, you can do everything on the command line. Use the `VBoxManage` command to import, configure, and start the appliance.

The following script imports the PMM Server appliance from `PMM-Server-1.6.0.ova` and configures it to bridge the en0 adapter from the host.  Then the script routes console output from the appliance to `/tmp/pmm-server-console.log`.  This is done because the script then starts the appliance in headless (without the console) mode.

To get the IP address for accessing PMM, the script waits for 1 minute until the appliance boots up and returns the lines with the IP address from the log file.

```
# Import image
VBoxManage import pmm-server-|VERSION NUMBER|.ova

# Modify NIC settings if needed
VBoxManage list bridgedifs
VBoxManage modifyvm 'PMM Server [VERSION NUMBER]' --nic1 bridged --bridgeadapter1 'en0: Wi-Fi (AirPort)'

# Log console output into file
VBoxManage modifyvm 'PMM Server [VERSION NUMBER]' --uart1 0x3F8 4 --uartmode1 file /tmp/pmm-server-console.log

# Start instance
VBoxManage startvm --type headless 'PMM Server [VERSION NUMBER]'

# Wait for 1 minute and get IP address from the log
sleep 60
grep cloud-init /tmp/pmm-server-console.log
```

In this script, `[VERSION NUMBER]` is the placeholder of the version of PMM Server that you are installing. By convention **OVA** files start with *pmm-server-* followed by the full version number such as 1.17.4.

To use this script, make sure to replace this placeholder with the the name of the image that you have downloaded from the [Download Percona Monitoring and Management](https://www.percona.com/downloads/pmm) site. This script also assumes that you have changed the working directory (using the **cd** command, for example) to the directory which contains the downloaded image file.
