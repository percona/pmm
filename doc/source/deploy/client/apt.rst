.. _install-client-apt:

=========================================
Installing PMM Client on Debian or Ubuntu
=========================================

If you are running a DEB-based Linux distribution,
use the :command:`apt` package manager
to install *PMM Client* from the official Percona software repository.

Percona provides :file:`.deb` packages for 64-bit versions
of the following distributions:

* Debian 7 (wheezy)
* Debian 8 (jessie)
* Ubuntu 12.04 LTS (Precise Pangolin)
* Ubuntu 14.04 LTS (Trusty Tahr)
* Ubuntu 16.04 LTS (Xenial Xerus)
* Ubuntu 16.10 (Yakkety Yak)

.. note:: *PMM Client* should work on other DEB-based distributions,
   but it is tested only on platforms listed above.

To install *PMM Client*:

1. If your system does not already have
   Percona's ``apt`` repository configured,
   fetch the repository package:

   .. code-block:: bash

      wget https://repo.percona.com/apt/percona-release_0.1-4.$(lsb_release -sc)_all.deb

#. Install the repository package:

   .. code-block:: bash

      sudo dpkg -i percona-release_0.1-4.$(lsb_release -sc)_all.deb

#. Update the local ``apt`` cache:

   .. code-block:: bash

      sudo apt-get update

#. Install the ``pmm-client`` package:

   .. code-block:: bash

      sudo apt-get install pmm-client

.. _apt-testing-repo:

Testing and Experimental Repositories
=====================================

Percona offers pre-release builds from the testing repo,
and early-stage development builds from the experimental repo.
To enable them, add either ``testing`` or ``experimental`` at the end
of the Percona repository definition in your repository file
(by default, :file:`/etc/apt/sources.list.d/percona-release.list`).

For example, if you are running Debian 8 ("jessie")
and want to install the latest testing builds,
the definitions should look like this::

  deb http://repo.percona.com/apt jessie main testing
  deb-src http://repo.percona.com/apt jessie main testing

If you are running Ubuntu 14.04 LTS (Trusty Tahr)
and want to install the latest experimental builds,
the definitions should look like this::

  deb http://repo.percona.com/apt trusty main experimental
  deb-src http://repo.percona.com/apt trusty main experimental

Next Steps
==========

After you install *PMM Client*,
:ref:`connect it to PMM Server <connect-client>`.

