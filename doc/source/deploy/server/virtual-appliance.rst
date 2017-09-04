.. _run-server-ova:

=========================================
Running PMM Server as a Virtual Appliance
=========================================

Percona provides a *virtual appliance*
for running *PMM Server* in a virtual machine.
It is distributed as an *Open Virtual Appliance* (OVA) package,
which is a :command:`tar` archive with necessary files
that follow the *Open Virtualization Format* (OVF).
OVF is supported by most popular virtualization platforms, including:

* `VMware <http://www.vmware.com/>`_
* `Red Hat Virtualization <https://www.redhat.com/en/technologies/virtualization>`_
* `VirtualBox <https://www.virtualbox.org/>`_
* `XenServer <https://www.xenserver.org/>`_
* `Microsoft System Center Virtual Machine Manager <https://www.microsoft.com/en-us/cloud-platform/system-center>`_

The virtual appliance is ideal for running *PMM Server*
on an enterprise virtualization platform of your choice.
This page provides examples for running the appliance in VirtualBox
and VMware Workstation Player,
which is a good choice to experiment with PMM
at a smaller scale on a local machine.
Similar procedure should work for other platforms
(including enterprise deployments on VMware ESXi, for example),
but additional steps may be required.

.. note:: The virtual machine used for the appliance runs CentOS 7.

.. warning:: The appliance must run in a network with DHCP,
   which will automatically assign an IP address for it.
   Currently it is not possible to run it in a network without DHCP
   and manually assign a static IP for the appliance.

Running in VMware Workstation Player
====================================

The following procedure describes how to run the *PMM Server* appliance
using VMware Workstation Player:

1. Download the OVA.

   The latest version is available at
   https://www.percona.com/redir/downloads/TESTING/pmm/

#. Import the appliance.

   1. Open the **File** menu and click **Open...**.

   #. Specify the path to the OVA and click **Continue**.

      .. note:: You may get an error indicating that import failed.
         Simply click **Retry** and import should succeed.

#. Configure network settings to make the appliance accessible
   from other hosts in your network.

   If you are running the appliance on a host
   with properly configured network settings,
   select **Bridged** in the **Network connection** section
   of the appliance settings.

#. Start the *PMM Server* appliance.

   If it was assigned an IP address on the network by DHCP,
   the URL for accessing PMM will be printed in the console window.

Running in VirtualBox Using the GUI
===================================

The following procedure describes how to run the *PMM Server* appliance
using the graphical user interface of VirtualBox:

1. Download the OVA.

   The latest version is available at
   https://www.percona.com/redir/downloads/TESTING/pmm/

#. Import the appliance.

   1. Open the **File** menu and click **Import Appliance**.

   #. Specify the path to the OVA and click **Continue**.

   #. Select **Reinitialize the MAC address of all network cards**
      and click **Import**.

#. Configure network settings to make the appliance accessible
   from other hosts in your network.

   .. note:: All database hosts must be in the same network as *PMM Server*,
      so do not set the network adapter to NAT.

   If you are running the appliance on a host
   with properly configured network settings,
   select **Bridged Adapter** in the **Network** section
   of the appliance settings.

#. Start the *PMM Server* appliance.

   If it was assigned an IP address on the network by DHCP,
   the URL for accessing PMM will be printed in the console window.

Running in VirtualBox Using the Command Line
============================================

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

.. code-block:: text

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

To use this script, change the name of the image to the latest version
downloaded from https://www.percona.com/redir/downloads/TESTING/pmm/
and run it in the same directory.

Accessing the Virtual Machine
=============================

To access the VM with the *PMM Server* appliance via SSH,
provide your public key:

1. Open the URL for accessing PMM in a web browser.

   The URL is provided either in the console window or in the appliance log.

#. Submit your **public key** in the PMM web interface.

After that you can use ``ssh`` to log in as the ``admin`` user.
For example, if *PMM Server* is running at 192.168.100.1
and your **private key** is :file:`~/.ssh/pmm-admin.key`,
use the following command::

 ssh admin@192.168.100.1 -i ~/.ssh/pmm-admin.key

Next Steps
==========

:ref:`Verify that PMM Server is running <deploy-pmm.server.verifying>`
by connecting to the PMM web interface using the IP address
assigned to the virtual appliance,
then :ref:`install PMM Client <install-client>`
on all database hosts that you want to monitor.

