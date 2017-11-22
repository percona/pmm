.. _connect-client:

===================================
Connecting PMM Client to PMM Server
===================================

After you :ref:`install PMM Client <install-client>`,
it does not automatically connect to PMM Server.

To connect the client to PMM Server,
specify the IP address using the ``pmm-admin config --server`` command.
For example, if *PMM Server* is running on ``192.168.100.1``,
and you installed *PMM Client* on a machine with IP ``192.168.200.1``:

.. code-block:: bash

   $ sudo pmm-admin config --server 192.168.100.1
   OK, PMM server is alive.

   PMM Server      | 192.168.100.1
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.200.1

.. note:: If you changed the default port 80
   when :ref:`running PMM Server <deploy-pmm.server.installing>`,
   specify it after the server's IP address. For example:

   .. code-block:: bash

      $ sudo pmm-admin config --server 192.168.100.1:8080

For more information, run ``pmm-admin config --help``.

Next Steps
==========

When the client is connected to PMM Server,
you can :ref:`start collecting data <start-collect>`
from the database instance.

