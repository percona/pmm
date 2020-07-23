.. _pmm-admin.config:

################################################
Configuring PMM Client with ``pmm-admin config``
################################################

.. _deploy-pmm.client-server.connecting:

****************************************
Connecting PMM Clients to the PMM Server
****************************************

With your server and clients set up, you must configure each PMM Client and
specify which PMM Server it should send its data to.

To connect a PMM Client, enter the IP address of the PMM Server as the value
of the ``--server-url`` parameter to the ``pmm-admin config`` command, and
allow using self-signed certificates with ``--server-insecure-tls``.

.. note:: The ``--server-url`` argument should include ``https://`` prefix
         and PMM Server credentials, which are ``admin``/``admin`` by default, if
         not changed at first PMM Server GUI access.

Run this command as root or by using the ``sudo`` command

.. code-block:: bash

   pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.100.1:443

For example, if your PMM Server is running on `192.168.100.1`, you have
installed PMM Client on a machine with IP `192.168.200.1`, and didn't change
default PMM Server credentials, run the following in the terminal of your
client. Run the following commands as root or by using the ``sudo`` command:

.. code-block:: bash

   pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.100.1:443

.. code-block:: text

   Checking local pmm-agent status...
   pmm-agent is running.
   Registering pmm-agent on PMM Server...
   Registered.
   Configuration file /usr/local/percona/pmm-agent.yaml updated.
   Reloading pmm-agent configuration...
   Configuration reloaded.
   Checking local pmm-agent status...
   pmm-agent is running.

If you change the default port 443 when running PMM Server, specify the new port number after the IP
address of PMM Server.

.. note:: By default ``pmm-admin config`` refuses to add client if it already
   exists in the PMM Server inventory database. If you need to re-add an
   already existing client (e.g. after full reinstall, hostname changes, etc.),
   you can run ``pmm-admin config`` with the additional ``--force`` option. This
   will remove an existing node with the same name, if any, and all its
   dependent services.
