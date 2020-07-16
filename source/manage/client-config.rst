.. _pmm-admin.config:

--------------------------------------------------------------------------------
`Configuring PMM Client with pmm-admin config <pmm-admin.config>`_
--------------------------------------------------------------------------------

.. _deploy-pmm.client-server.connecting:

`Connecting PMM Clients to the PMM Server <client-config.html#deploy-pmm-client-server-connecting>`_
====================================================================================================

With your server and clients set up, you must configure each |pmm-client| and
specify which |pmm-server| it should send its data to.

To connect a |pmm-client|, enter the IP address of the |pmm-server| as the value
of the ``--server-url`` parameter to the |pmm-admin.config| command, and
allow using self-signed certificates with ``--server-insecure-tls``.

.. note:: The ``--server-url`` argument should include ``https://`` prefix
         and PMM Server credentials, which are **admin/admin** by default, if
         not changed at first PMM Server GUI access.

|tip.run-this.root|

.. include:: ../.res/code/pmm-admin.config.server.url.port.txt

For example, if your |pmm-server| is running on `192.168.100.1`, you have
installed |pmm-client| on a machine with IP `192.168.200.1`, and didn't change
default PMM Server credentials, run the following in the terminal of your
client. |tip.run-all.root|:

.. include:: ../.res/code/pmm-admin.config.server.url.txt

If you change the default port **443** when running PMM Server, specify the new port number after the IP
address of |pmm-server|.

.. note:: By default ``pmm-admin config`` refuses to add client if it already
   exists in the PMM Server inventory database. If you need to re-add an
   already existing client (e.g. after full reinstall, hostname changes, etc.),
   you can run ``pmm-admin config`` with the additional ``--force`` option. This
   will remove an existing node with the same name, if any, and all its
   dependent services.

.. include:: ../.res/replace.txt
