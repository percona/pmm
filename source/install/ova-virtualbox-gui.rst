.. _pmm.deploying.server.ova-virtualbox-gui:

VirtualBox Using the GUI
================================================================================

The following procedure describes how to run the PMM Server appliance
using the graphical user interface of VirtualBox:

1. Download the OVA. The latest version is available at the `Download Percona Monitoring and Management`_ site.
#. Import the appliance. For this, open the *File* menu and click
   *Import Appliance* and specify the path to the OVA and click
   *Continue*. Then, select
   *Reinitialize the MAC address of all network cards* and click *Import*.
#. Configure network settings to make the appliance accessible
   from other hosts in your network.

   .. note:: All database hosts must be in the same network as *PMM Server*,
      so do not set the network adapter to NAT.

   If you are running the appliance on a host with properly configured network
   settings, select *Bridged Adapter* in the *Network* section of the
   appliance settings.

#. Start the PMM Server appliance.

   If it was assigned an IP address on the network by DHCP, the URL for
   accessing PMM will be printed in the console window.

.. seealso::

   Accessing the Virtual Machine via SSH
      :ref:`pmm.deploying.server.virtual-appliance.accessing`

.. _`download percona monitoring and management`: https://www.percona.com/downloads/pmm


