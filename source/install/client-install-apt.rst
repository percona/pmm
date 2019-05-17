.. _install-client-apt:

Installing DEB packages using apt-get
================================================================================

If you are running a DEB-based |linux| distribution, use the |apt| package
manager to install |pmm-client| from the official Percona software repository.

|percona| provides :file:`.deb` packages for 64-bit versions of the following
distributions:

.. include:: ../.res/contents/list.pmm-client.supported-apt-platform.txt

.. note::

   |pmm-client| should work on other DEB-based distributions, but it is tested
   only on the platforms listed above.

To install the |pmm-client| package, complete the following
procedure. |tip.run-all.root|:

1. Configure |percona| repositories using the `percona-release <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ tool. First you’ll need to download and install the official percona-release package from Percona::

     wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
     sudo dpkg -i percona-release_latest.generic_all.deb

   Since PMM 2 is still not GA, you’ll need to use it to enable the experimental component of the original Percona repository::

     sudo percona-release disable all
     sudo percona-release enable original experimental

   See `percona-release official documentation <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ for details.

#. Install the ``pmm2-client`` package::

     sudo apt-get update
     sudo apt-get install pmm2-client

#. Having experimental packages enabled may affect further packages installation with versions which are not ready for production. To avoid this, disable this component with the following commands::

     sudo percona-release disable original experimental
     sudo apt-get update

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
