.. _pmm/deploying/server/ova/virtualbox/gui:

VirtualBox Using the GUI
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

#. Start the *PMM Server* appliance and set the root password
   (required on the first login).

   If it was assigned an IP address on the network by DHCP,
   the URL for accessing PMM will be printed in the console window.

.. seealso::

   Using |pmm-server| as a virtual appliance
      :ref:`pmm/deploying/server/virtual-appliance`
   Setting the root password
      :ref:`pmm/deploying/server/virtual-appliance/root-password/set`


.. include:: .res/replace/name.txt
.. include:: .res/replace/url.txt
