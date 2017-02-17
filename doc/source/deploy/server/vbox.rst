.. _run-server-vbox:

==========================================
Running PMM Server Using VirtualBox Images
==========================================

Percona provides an *Open Virtual Appliance* (OVA) of *PMM Server*,
which you can run in most popular hypervisors.
The following procedure describes how to run the appliance in VirtualBox:

1. Download the OVA.

   The latest version is available at
   https://www.percona.com/redir/downloads/TESTING/pmm/.

#. Import the appliance.

   1. Open the **File** menu and click **Import Appliance**.

   #. Specify the path to the OVA and click **Continue**.

   #. Select **Reinitialize the MAC address of all network cards**
      and click **Import**.

#. Configure network settings to make the appliance accessible
   from other hosts in your network.

   .. note:: All database hosts must be in the same network as *PMM Server*.

   If you are running the appliance on a host
   with properly configured network settings,
   select **Bridged Adapter** in the **Network** section
   of the appliance settings.

#. Start the *PMM Server* appliance.

   If it was assigned an IP address on the network,
   the URL for accessing PMM will be printed in the console window.

Running on the Command Line
===========================

Instead of using the VirtualBox GUI,
you can do everything on the command line.
Use the ``VBoxManage`` command to import, configure,
and start the appliance.

The following script imports the *PMM Server* appliance
from :file:`PMM-Server-2017-01-24.ova`
and configures it to bridge the `en0` adapter from the host.
Then the script routes console output from the appliance
to :file:`/tmp/pmm-server-console.log`.
This is done because the script then starts the appliance in headless mode
(that is, without the console).
To get the IP address for accessing PMM,
the script waits for 1 minute until the appliance boots up
and returns the lines with the IP address from the log file.

.. code-block:: none

   # Import image
   VBoxManage import PMM-Server-2017-01-24.ova

   # Modify NIC settings if needed
   VBoxManage list bridgedifs
   VBoxManage modifyvm 'PMM Server [2017-01-24]' --nic1 bridged --bridgeadapter1 'en0: Wi-Fi (AirPort)'

   # Log console output into file
   VBoxManage modifyvm 'PMM Server [2017-01-24]' --uart1 0x3F8 4 --uartmode1 file /tmp/pmm-server-console.log

   # Start instance
   VBoxManage startvm --type headless 'PMM Server [2017-01-24]'

   # Wait for 1 minute and get IP address from the log
   sleep 60
   grep cloud-init /tmp/pmm-server-console.log

Next Steps
==========

:ref:`Verify that PMM Server is running <verify-server>`
by connecting to the PMM web interface using the IP address
assigned to the virtual appliance,
then :ref:`install PMM Client <install-client>`
on all database hosts that you want to monitor.

