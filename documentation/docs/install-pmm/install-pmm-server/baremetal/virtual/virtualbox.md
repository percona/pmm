# VirtualBox - Import OVA file

=== "OVA file downloaded from UI"
    To import downloaded file from UI:
    {.power-number}

    1. Select **File** → **Import appliance...**.
    2. In the **File** field, type the path to the downloaded `.ova` file, or click the folder icon to navigate and open it.
    3. Click **Continue**.
    4. On the **Appliance settings** page, review the settings and click *Import*.
    5. Click **Start**.
    6. When the guest has booted, note the IP address in the guest console.

=== "OVA file downloaded via CLI"
    To import downloaded file from CLI:
    {.power-number}

    1. Open a terminal and change directory to where the downloaded `.ova` file is.

    2. (Optional) Do a 'dry run' import to see what values will be used.

        ```sh
        VBoxManage import pmm-server-{{release}}.ova --dry-run
        ```

    3. Import the image.
        
        Choose one of:
        
        * With the default settings.

            ```sh
            VBoxManage import pmm-server-{{release}}.ova
            ```

        * With custom settings (in this example, Name: "PMM Server", CPUs: 2, RAM: 8192 MB).

            ```sh
            VBoxManage import --vsys 0 --vmname "PMM Server" \
            --cpus 2 --memory 8192 pmm-server-{{release}}.ova
            ```

## Reconfigure interface
 

### Reconfigure with UI

To reconfigure the interface with the UI:
{.power-number}

1. Click **Settings**.
2. Click **Network**.
3. In the **Adapter 1** field, click **Attached to** and change to **Bridged Adapter**.
4. In the **Name** field, select your host's active network interface (e.g. `en0: Wi-Fi (Wireless)`).
5. Click **OK**.

### Reconfigure via CLI

To reconfigure via the CLI:
{.power-number}

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

### Start guest and get IP address from UI

To start the guest and get the IP address from the UI:
{.power-number}

1. Select the **PMM Server** virtual machine in the list.
2. Click **Start**.
3. When the guest has booted, note the IP address in the guest console.

### Start guest and get IP address from CLI

To start the guest and get the IP address from the CLI:
{.power-number}

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