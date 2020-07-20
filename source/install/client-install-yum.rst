.. _install-client-yum:

Installing RPM packages using ``yum``
================================================================================

If you are running an RPM-based Linux distribution, use the ``yum`` package
manager to install PMM Client from the official Percona software repository.

Percona provides :file:`.rpm` packages for 64-bit versions
of Red Hat Enterprise Linux 6 (Santiago) and 7 (Maipo),
including its derivatives that claim full binary compatibility,
such as, CentOS, Oracle Linux, Amazon Linux AMI, and so on.

.. note::

   PMM Client should work on other RPM-based distributions,
   but it is tested only on RHEL and CentOS versions 6 and 7.

To install the PMM Client package, complete the following procedure. Run the following commands as root or by using the ``sudo`` command:

1. Configure Percona repositories using the `percona-release <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ tool. First you’ll need to download and install the official percona-release package from Percona::

     sudo yum install https://repo.percona.com/yum/percona-release-latest.noarch.rpm

   .. note:: If you have previously enabled the experimental or testing
      Percona repository, don't forget to disable them and enable the release
      component of the original repository as follows::

         sudo percona-release disable all
         sudo percona-release enable original release

   See `percona-release official documentation <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ for details.

#. Install the ``pmm2-client`` package:

   .. include:: ../.res/code/yum.install.pmm-client.txt

#. Once PMM Client is installed, run the ``pmm-admin config`` command with your PMM Server IP address to register your Node within the Server:

   .. include:: ../.res/code/pmm-admin.config.server.url.dummy.txt

   You should see the following::

     Checking local pmm-agent status...
     pmm-agent is running.
     Registering pmm-agent on PMM Server...
     Registered.
     Configuration file /usr/local/percona/pmm-agent.yaml updated.
     Reloading pmm-agent configuration...
     Configuration reloaded.



