.. _install-client-apt:

=========================================
Installing PMM Client on Debian or Ubuntu
=========================================

If you are running a DEB-based Linux distribution,
use the :command:`apt` package manager
to install *PMM Client* from the official Percona software repository.

Percona provides :file:`.deb` packages for 64-bit versions
of the following distributions:

.. include:: ../../.res/contents/list.pmm-client.supported-apt-platform.txt

.. note:: *PMM Client* should work on other DEB-based distributions,
   but it is tested only on the platforms listed above.

To install *PMM Client*:

1. Configure Percona repositories as described in
   `Percona Software Repositories Documentation
   <https://www.percona.com/doc/percona-repo-config/index.html>`_.

#. Install the ``pmm-client`` package:

   .. code-block:: bash

      sudo apt-get install pmm-client

.. _apt-testing-repo:

Next Steps
==========

After you install *PMM Client*,
:ref:`connect it to PMM Server <connect-client>`.

