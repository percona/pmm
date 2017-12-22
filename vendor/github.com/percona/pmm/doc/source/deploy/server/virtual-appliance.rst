.. _run-server-ova:

================================================================================
Running PMM Server as a Virtual Appliance
================================================================================

Percona provides a *virtual appliance* for running |pmm-server| in a virtual
machine.  It is distributed as an *Open Virtual Appliance* (OVA) package, which
is a :command:`tar` archive with necessary files that follow the *Open
Virtualization Format* (OVF).  OVF is supported by most popular virtualization
platforms, including:

* `VMware - ESXi 6.5`_
* `Red Hat Virtualization`_
* `VirtualBox`_
* `XenServer`_
* `Microsoft System Center Virtual Machine Manager`_

The virtual appliance is ideal for running |pmm-server| on an
enterprise virtualization platform of your choice.  This page provides
examples for running the appliance in |virtualbox| and VMware
Workstation Player, which is a good choice to experiment with |pmm| at
a smaller scale on a local machine.  Similar procedure should work for
other platforms (including enterprise deployments on VMware ESXi, for
example), but additional steps may be required.

.. note:: The virtual machine used for the appliance runs |centos| 7.

.. warning:: The appliance must run in a network with DHCP, which will
   automatically assign an IP address for it.  Currently it is not
   possible to run it in a network without DHCP and manually assign a
   static IP for the appliance.

Running in VMware Workstation Player
================================================================================

The following procedure describes how to run the *PMM Server* appliance
using VMware Workstation Player:

1. Download the OVA.

   The latest version is available at the `Download Percona Monitoring and Management`_ site.

#. Import the appliance.

   1. Open the |gui.file| menu and click |gui.open|.

   #. Specify the path to the OVA and click |gui.continue|.

      .. note:: You may get an error indicating that import failed.
         Simply click |gui.retry| and import should succeed.

#. Configure network settings to make the appliance accessible
   from other hosts in your network.

   If you are running the appliance on a host
   with properly configured network settings,
   select **Bridged** in the **Network connection** section
   of the appliance settings.

#. Start the |pmm-server| appliance.

   If it was assigned an IP address on the network by DHCP,
   the URL for accessing PMM will be printed in the console window.

Running in VirtualBox Using the GUI
================================================================================

The following procedure describes how to run the |pmm-server| appliance
using the graphical user interface of VirtualBox:

1. Download the OVA.

   The latest version is available at the `Download Percona Monitoring and Management`_ site.

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

Instead of using the |virtualbox| GUI,
you can do everything on the command line.
Use the ``VBoxManage`` command to import, configure,
and start the appliance.

The following script imports the |pmm-server| appliance
from :file:`PMM-Server-2017-01-24.ova`
and configures it to bridge the `en0` adapter from the host.
Then the script routes console output from the appliance
to :file:`/tmp/pmm-server-console.log`.
This is done because the script then starts the appliance in headless mode
(that is, without the console).
To get the IP address for accessing PMM,
the script waits for 1 minute until the appliance boots up
and returns the lines with the IP address from the log file.

.. include:: ../../.res/code/sh.org
   :start-after: +vboxmanage+
   :end-before: #+end-block

To use this script, change the name of the image to the latest version
downloaded from the `Download Percona Monitoring and Management` site
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

.. include:: ../../.res/replace/name.txt
.. include:: ../../.res/replace/program.txt
.. include:: ../../.res/replace/option.txt
.. include:: ../../.res/replace/url.txt
