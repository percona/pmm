.. _pmm.deploying.server.ova.vmware-workstation-player:

VMware Workstation Player
================================================================================

The following procedure describes how to run the *PMM Server* appliance
using VMware Workstation Player:

1. Download the OVA. The latest version is available at the `Download Percona Monitoring and Management`_ site.

#. Import the appliance.

   1. Open the |gui.file| menu and click |gui.open|.

   #. Specify the path to the OVA and click |gui.continue|.

      .. note:: You may get an error indicating that import failed.
         Simply click |gui.retry| and import should succeed.

#. Configure network settings to make the appliance accessible
   from other hosts in your network.

   If you are running the applianoce on a host
   with properly configured network settings,
   select **Bridged** in the **Network connection** section
   of the appliance settings.

#. Start the |pmm-server| appliance.

   If it was assigned an IP address on the network by DHCP,
   the URL for accessing PMM will be printed in the console window.

.. seealso::

   Accessing the Virtual Machine via SSH
      :ref:`pmm.deploying.server.virtual-appliance.accessing`

.. _`download percona monitoring and management`: https://www.percona.com/downloads/pmm

.. include:: ../.res/replace.txt

