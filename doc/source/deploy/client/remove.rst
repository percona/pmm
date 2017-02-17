.. _remove-client:

===================
Removing PMM Client
===================

1. Remove all monitored instances as described in :ref:`pmm-admin-rm`.

2. Change into the directory with the extracted *PMM Client* tarball
   and run:

   .. code-block:: bash

      $ sudo ./uninstall

.. note::

   * If you installed using RPM packages:

     .. code-block:: bash

        $ rpm -e pmm-client

   * If you installed using YUM:

     .. code-block:: bash

        $ yum remove pmm-client

   * If you installed using DEB packages:

     .. code-block:: bash

        $ dpkg -r pmm-client

   * If you installed using APT:

     .. code-block:: bash

        $ apt-get remove pmm-client

