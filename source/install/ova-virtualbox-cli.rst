.. _pmm.deploying.server.ova-virtualbox-cli:

VirtualBox Using the Command Line
================================================================================

Instead of using the |virtualbox| GUI, you can do everything on the command
line. Use the ``VBoxManage`` command to import, configure, and start the
appliance.

The following script imports the |pmm-server| appliance from
|pmm-server-1.6.0.ova| and configures it to bridge the `en0` adapter from the
host.  Then the script routes console output from the appliance to
|tmp.pmm-server-console.log|.  This is done because the script then starts the
appliance in headless (without the console) mode.

To get the IP address for accessing PMM, the script waits for 1 minute until the
appliance boots up and returns the lines with the IP address from the log file.

.. include:: ../.res/code/vboxmanage.txt

In this script, :code:`[VERSION NUMBER]` is the placeholder of the version of
|pmm-server| that you are installing. By convention **OVA** files start with
*pmm-server-* followed by the full version number such as |release|.

To use this script, make sure to replace this placeholder with the the name of
the image that you have downloaded from the `Download Percona Monitoring and
Management`_ site. This script also assumes that you have changed the working
directory (using the |cd| command, for example) to the directory which contains
the downloaded image file.

.. seealso::

   Accessing the Virtual Machine via SSH
      :ref:`pmm.deploying.server.virtual-appliance.accessing`


.. rubric:: Downloading the latest development version

.. _`download percona monitoring and management`: https://www.percona.com/downloads/pmm

.. include:: ../.res/replace.txt
