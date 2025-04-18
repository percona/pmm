# Deploy PMM Server on VirtualBox

Import the PMM Server OVA file into Oracle VirtualBox to create a virtual machine for your monitoring environment.

## Prerequisites

- Downloaded [PMM Server OVA file](download_ova.md)
- Oracle VirtualBox 6.0 or later installed
- At least 8GB of free RAM and 100GB of free disk space

## Import OVA file

=== "Using VirtualBox UI"
    To import the OVA file using the VirtualBox user interface:
    {.power-number}

    1. Open Oracle VirtualBox.
    2. Go to **File > Import Appliance**.
    3. Click on the folder icon and browse to select the downloaded PMM Server OVA file, then click **Next**.
    5. Review the appliance settings:
        - You can customize the name of the VM
        - Adjust CPU and memory settings if needed
        - Review network settings
    6. Click **Import**.
    7. Wait for the import process to complete (this may take several minutes).

=== "Using VBoxManage CLI"
    To import the OVA file using the command-line interface:
    {.power-number}

    1. Open a terminal or command prompt.
    2. Use the VBoxManage command to import the OVA:

        ```sh
        VBoxManage import pmm-server-{{release}}.ova --vsys 0 --vmname "PMM Server"
        ```

    3. To customize VM settings during import (optional):

        ```sh
        VBoxManage import pmm-server-{{release}}.ova --vsys 0 --vmname "PMM Server" \
          --cpus 4 --memory 8192 --unit 9 --disk pmm-data.vmdk
        ```

## Configure network settings

For the VM to be accessible on your network, configure the network settings appropriately.

=== "Using VirtualBox UI"
    To configure network settings using the VirtualBox UI:
    {.power-number}

    1. Select the imported PMM Server VM.
    2. Go to **Settings > Network**.
    3. Ensure **Adapter 1** is enabled and attached to:
        - **Bridged Adapter** for direct network access (recommended)
        - **NAT** if you prefer to use port forwarding
    4. If using **Bridged Adapter**, select the physical network interface to bridge to.
    5. Click **OK**.

=== "Using VBoxManage CLI"
    To configure network settings using the command line:
    {.power-number}

    1. For bridged networking (recommended for production):

        ```sh
        VBoxManage modifyvm "PMM Server" --nic1 bridged --bridgeadapter1 eth0
        ```
        Replace `eth0` with your actual network interface name.

    2. For NAT networking (easier for testing):

        ```sh
        VBoxManage modifyvm "PMM Server" --nic1 nat
        ```

    3. To set up port forwarding with NAT (optional):

        ```sh
        VBoxManage modifyvm "PMM Server" --nic1 nat
        VBoxManage modifyvm "PMM Server" --natpf1 "https,tcp,,8443,,443"
        VBoxManage modifyvm "PMM Server" --natpf1 "http,tcp,,8080,,80"
        ```
        This forwards host ports 8443 and 8080 to guest ports 443 and 80.

## Start the VM and obtain IP address

=== "Using VirtualBox UI"
    To start the VM and get its IP address using the UI:
    {.power-number}

    1. Select the PMM Server VM in the VirtualBox Manager.
    2. Click **Start**.
    3. A console window will open showing the boot process.
    4. Wait for the boot process to complete (2-5 minutes).
    5. The console will display the IP address once booting is complete.

=== "Using VBoxManage CLI"
    To start the VM and get its IP address using the command line:
    {.power-number}

    1. Start the VM in headless mode (no UI):

        ```sh
        VBoxManage startvm "PMM Server" --type headless
        ```

    2. Wait for the VM to fully boot (approximately 2-5 minutes).

    3. Get the VM's IP address (for bridged networking):

        ```sh
        VBoxManage guestproperty get "PMM Server" "/VirtualBox/GuestInfo/Net/0/V4/IP"
        ```

    4. If the above command doesn't show the IP address, you can check the VM's console:

        ```sh
        VBoxManage startvm "PMM Server" --type separate
        ```
        This opens just the console window.


## Troubleshooting network issues
If you cannot connect to the VM:
    
- For bridged networking, ensure your host's firewall allows traffic to the VM
- For NAT with port forwarding, connect to your host's IP address with the forwarded port (e.g., https://localhost:8443)
- Verify VirtualBox network settings are correctly configured

## Next steps

After successfully importing and starting the PMM Server VM:

- Open a web browser and navigate to `https://<vm-ip-address>`
- [Complete initial login and setup](login_UI.md)
- [Register PMM Clients](../../../register-client-node/index.md) to begin monitoring