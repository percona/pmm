.. _install-client-apt:

#########################################
Installing DEB packages using ``apt-get``
#########################################

If you are running a DEB-based Linux distribution, you can use the ``apt`` package
manager to install PMM client from the official Percona software repository.

Percona provides ``.deb`` packages for 64-bit versions of popular Linux distributions.

The list can be found on `Percona's Software Platform Lifecycle page <https://www.percona.com/services/policies/percona-software-platform-lifecycle/>`__.

.. note::

   Although PMM client should work on other DEB-based distributions, it is tested
   only on the platforms listed above.

To install the PMM client package, follow these steps.


1. Configure Percona repositories using the `percona-release <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ tool. First you’ll need to download and install the official ``percona-release`` package from Percona:

   .. code-block:: bash

      wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
      sudo dpkg -i percona-release_latest.generic_all.deb

   .. note::
   
      If you have previously enabled the experimental or testing
      Percona repository, don't forget to disable them and enable the release
      component of the original repository as follows:

      .. code-block:: bash

         sudo percona-release disable all
         sudo percona-release enable original release

2. Install the PMM client package:

   .. code-block:: bash

      sudo apt-get update
      sudo apt-get install pmm2-client

3. Register your Node:

   .. code-block:: bash

      pmm-admin config --server-insecure-tls --server-url=https://admin:admin@<IP Address>:443

4. You should see the following output:

   .. code-block:: text

     Checking local pmm-agent status...
     pmm-agent is running.
     Registering pmm-agent on PMM Server...
     Registered.
     Configuration file /usr/local/percona/pmm-agent.yaml updated.
     Reloading pmm-agent configuration...
     Configuration reloaded.
