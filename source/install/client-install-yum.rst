.. _install-client-yum:

Installing RPM packages using yum
================================================================================

If you are running an RPM-based |linux| distribution, use the |yum| package
manager to install |pmm-client| from the official Percona software repository.

Percona provides :file:`.rpm` packages for 64-bit versions
of Red Hat Enterprise Linux 6 (Santiago) and 7 (Maipo),
including its derivatives that claim full binary compatibility,
such as, CentOS, Oracle Linux, Amazon Linux AMI, and so on.

.. note::

   |pmm-client| should work on other RPM-based distributions,
   but it is tested only on RHEL and CentOS versions 6 and 7.

To install the |pmm-client| package, complete the following procedure. |tip.run-all.root|:

1. Configure |percona| repositories using the `percona-release <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ tool. First you’ll need to download and install the official percona-release package from Percona::

     sudo yum install https://repo.percona.com/yum/percona-release-latest.noarch.rpm

   Since PMM 2 is still not GA, you’ll need to use it to enable the experimental component of the original Percona repository::

     sudo percona-release disable all
     sudo percona-release enable original experimental

   See `percona-release official documentation <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ for details.

#. Install the ``pmm2-client`` package:

   .. include:: ../.res/code/yum.install.pmm-client.txt

#. Having experimental packages enabled may affect further packages installation with versions which are not ready for production. To avoid this, disable this component with the following commands::

     sudo percona-release disable original experimental

#. Once PMM Client is installed, run the ``pmm-admin config`` command with your PMM Server IP address to register your Node within the Server::

     pmm-admin config --server-insecure-tls --server-address=<IP Address>:443

   You should see the following::

     Checking local pmm-agent status...
     pmm-agent is running.
     Registering pmm-agent on PMM Server...
     Registered.
     Configuration file /usr/local/percona/pmm-agent.yaml updated.
     Reloading pmm-agent configuration...
     Configuration reloaded.


.. include:: ../.res/replace.txt
