# Log into user interface

To log in to the PMM user interface:
{.power-number}

1. Open a web browser and visit the guest IP address.

2. The PMM [login screen](../../../../reference/ui/log_in.md) appears.

3. Enter the default username and password in the relevant fields and click *Log in*.

    - username: `admin`

    - password: `admin`

4. (Recommended) Follow the prompts to change the default password.

    !!! note alert alert-primary "Note"
        You also can change the default password through SSH by using the `change-admin-password` command.

5. The PMM Home Dashboard appears.


??? info "(optional) Change root password from UI"

    1. Start the virtual machine in GUI mode.

    2. Log in with the default superuser credentials:

        - Username: `root`

        - Password: `percona`

    3. Follow the prompts to change the password.


??? info "(optional) Set up SSH from UI/CLI"

    To set up SSH from UI/CLI:
    {.power-number}

    1. Create a key pair for the `admin` user.

        ```sh
        ssh-keygen -f admin
        ```

    2. Log into the PMM user interface.

    3. Select **PMM Configuration > Settings > SSH Key**.

    4. Copy and paste the contents of the `admin.pub` file into the **SSH Key** field.

    5. Click **Apply SSH Key**. (This copies the public key to `/home/admin/.ssh/authorized_keys` in the guest).

    6. Log in via SSH (`N.N.N.N` is the guest IP address).

        ```sh
        ssh -i admin admin@N.N.N.N
        ```

??? info "(optional) Set up static IP via CLI"

    When the guest OS starts, it will get an IP address from the hypervisor's DHCP server. This IP can change each time the guest OS is restarted. Setting a static IP for the guest OS avoids having to check the IP address whenever the guest is restarted.
    {.power-number}

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