# VMware - Import OVA file

=== "OVA file downloaded from UI"
    To import downloaded file from UI:
    {.power-number}

    1. Select **File** → **Import**.
    2. Click **Choose* file...*.
    3. Navigate to the downloaded `.ova` file and select it.
    4. Click **Open**.
    5. Click **Continue**.
    6. In the **Save as** dialog:
        -  (Optional) Change the directory or file name.
        -  Click **Save**.

    7. Choose one of:
        - (Optional) Click **Finish**. This starts the virtual machine.
        - (Recommended) Click **Customize Settings**. This opens the VM's settings page without starting the machine.

=== "OVA file downloaded via CLI"
    To import downloaded file from the CLI:
    {.power-number}

    1. Install [`ovftool`][OVFTool]. (You need to register.)
    2. Import and convert the OVA file. (`ovftool` can't change CPU or memory settings during import but it can set the default interface.)

        Choose one of:

        * Download and import the OVA file.

            ```sh
            ovftool --name="PMM Server" --net:NAT=Wi-Fi \
            https://www.percona.com/downloads/pmm/{{release}}/ova/pmm-server-{{release}}.ova \
            pmm-server-{{release}}.vmx
            ```

        * Import an already-downloaded OVA file.

            ```sh
            ovftool --name="PMM Server" --net:NAT=WiFi \
            pmm-server-{{release}}.ova \
            pmm-server.vmx
            ```

## Reconfigure interface

!!! note alert alert-primary "Note"
    When using the command line, the interface is remapped during import.

### Reconfigure with UI

To reconfigure the interface with the UI:
{.power-number}


1. If started, shut down the virtual machine.
2. In the VMware main window, select the imported virtual machine.
3. Click **Virtual Machine** → **Settings...**.
4. Click **Network Adapter**.
5. In the **Bridged Networking** section, select **Autodetect**.
6. Close the settings window.

### Start guest and get IP address from UI

To start the guest and get the IP address from the UI:
{.power-number}


1. In the VMware main window, select the imported virtual machine.
2. Click the play button <i class="uil uil-caret-right"></i> or select **Virtual Machine** → **Start Up**.
3. When the instance has been booted, note the IP address in the guest console.

### Start guest and get IP address from CLI

To start the guest and get the IP address from the CLI:
{.power-number}

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

[OVA]: https://www.percona.com/downloads/pmm/{{release}}/ova
[OVF]: https://wikipedia.org/wiki/Open_Virtualization_Format
[VirtualBox]: https://www.virtualbox.org/
[VMware]: https://www.vmware.com/products/
[OVFTool]: https://code.vmware.com/tool/ovf