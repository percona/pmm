# Deploy PMM Server on VMware

Import the PMM Server OVA file into VMware products including ESXi, vSphere, Workstation, and Fusion to create a virtual machine for your monitoring environment.

## Prerequisites

- Downloaded [PMM Server OVA file](download_ova.md)
- VMware product installed (Workstation, Fusion, ESXi, or vSphere)
- At least 8GB of free RAM and 100GB of free disk space
- Network connectivity to monitored database instances

## Import OVA file

=== "OVA file downloaded using WMware UI"
    To import the OVA file using the VMware user interface:
    {.power-number}

    1. Select **File > Import**.
    2. Click **Choose file** (wording may vary depending on VMware product).
    3. Navigate to the downloaded `.ova` file and open it.
    6. In the **Save as** dialog:

        -  (Optional) Change the directory or virtual machine name.
        -  Click **Save**.
    7. Choose one of:

        - Click **Finish** to complete the import and start the virtual machine.
        - (Recommended) Click **Customize Settings** to open the VM's settings page before starting the machine.

=== "OVA file downloaded via CLI"
    To import downloaded file using the command-line interface:
    {.power-number}

    1. Install [`ovftool`][OVFTool]. (You need to register.)
    2. Import and convert the OVA file using one of these methods:

        * To download and import the OVA file directly:

            ```sh
            ovftool --name="PMM Server" --net:NAT=Wi-Fi \
            https://www.percona.com/downloads/pmm/{{release}}/ova/pmm-server-{{release}}.ova \
            pmm-server-{{release}}.vmx
            ```

        * To import a previously downloaded OVA file, replacing `Wi-Fi` with your actual network interface name. You can list available network interfaces with `ovftool --listNetworks`:

            ```sh
            ovftool --name="PMM Server" --net:NAT=Wi-Fi \
            pmm-server-{{release}}.ova \
            pmm-server.vmx
            ```

## Configure network settings

For PMM Server to be accessible, it must have proper network configuration. Bridged networking is recommended for production environments.

When using the command line, the interface is remapped during import.

### Configure networking with UI

To configure VM network settings using the UI:
{.power-number}

1. If the VM is running, shut it down first.
2. In the VMware main window, select the imported virtual machine.
3. Click **Virtual Machine** → **Settings...**.
4. Click **Network Adapter** in the hardware list.
5. Select the appropriate networking mode:
    - **Bridged Networking**: Recommended for production (direct network access)
    - **NAT**: For testing environments
6. If using bridged networking, select **Autodetect** or choose a specific network adapter.
7. Click **OK** to save changes.

### Start the VM and obtain IP address (UI method)

To start the VM and find its IP address using the VMware UI:
{.power-number}

1. In the VMware main window, select the imported PMM Server virtual machine.
2. Click the play button <i class="uil uil-caret-right"></i> or select **Virtual Machine** → **Start Up**.
3. Wait for the VM to boot completely (this may take 2-5 minutes).
4. Look for the IP address displayed in the VM console window.

### Start the VM and obtain IP address (CLI method)

To start the VM and get its IP address using the command line:
{.power-number}

1. Start the virtual machine in GUI mode to view the console:

    ```sh
    vmrun -gu root -gp percona start \
    pmm-server.vmx gui
    ```

2. Wait for the boot process to complete and note the IP address displayed in the VM console.

3. Optional: After noting the IP address, you can stop and restart the VM in headless mode:

    ```sh
    vmrun stop pmm-server.vmx
    vmrun -gu root -gp percona start \
    pmm-server.vmx nogui
    ```

## Next steps

After successfully importing and starting the PMM Server VM:

- Open a web browser and navigate to `https://<vm-ip-address>`
- [Complete initial login and setup](login_UI.md)
- [Register PMM Clients](../../../register-client-node/index.md) to begin monitoring

!!! tip "Bookmarking"
    Save the PMM Server IP address or add it to your bookmarks for easy access. For production environments, consider configuring a static IP address or DNS name.

[OVA]: https://www.percona.com/downloads/pmm/{{release}}/ova
[OVF]: https://wikipedia.org/wiki/Open_Virtualization_Format
[VirtualBox]: https://www.virtualbox.org/
[VMware]: https://www.vmware.com/products/
[OVFTool]: https://code.vmware.com/tool/ovf