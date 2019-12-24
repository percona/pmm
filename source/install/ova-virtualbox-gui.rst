.. _pmm.deploying.server.ova-virtualbox-gui:

VirtualBox Using the GUI
================================================================================

The following procedure describes how to run the |pmm-server| appliance
using the graphical user interface of VirtualBox:

1. Download the OVA. The latest version is available at the `Download Percona Monitoring and Management`_ site.
#. Import the appliance. For this, open the |gui.file| menu and click
   |gui.import-appliance| and specify the path to the OVA and click
   |gui.continue|. Then, select
   |gui.reinitialize-mac-address-of-all-network-cards| and click |gui.import|.
#. Configure network settings to make the appliance accessible
   from other hosts in your network.

   .. note:: All database hosts must be in the same network as *PMM Server*,
      so do not set the network adapter to NAT.

   If you are running the appliance on a host with properly configured network
   settings, select |gui.bridged-adapter| in the |gui.network| section of the
   appliance settings.

#. Start the |pmm-server| appliance.

   If it was assigned an IP address on the network by |dhcp|, the URL for
   accessing |pmm| will be printed in the console window.

.. seealso::

   Using |pmm-server| as a virtual appliance
      :ref:`pmm.deploying.server.virtual-appliance`
   Accessing the Virtual Machine via SSH
      :ref:`pmm.deploying.server.virtual-appliance.accessing`

.. _`download percona monitoring and management`: https://www.percona.com/downloads/pmm

.. include:: ../.res/replace.txt
