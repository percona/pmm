.. _deploy-pmm.client_server.connecting:

`Connecting PMM Clients to the PMM Server <deploy-pmm.client_server.connecting>`_
=====================================================================================

With your server and clients set up, you must configure each |pmm-client| and
specify which |pmm-server| it should send its data to.

To connect a |pmm-client|, enter the IP address of the |pmm-server| as the value
of the |opt.server| parameter to the |pmm-admin.config| command.

|tip.run-this.root|

.. include:: ../.res/code/pmm-admin.config.server.url.port.txt

For example, if your |pmm-server| is running on `192.168.100.1`, and you have
installed |pmm-client| on a machine with IP `192.168.200.1`, run the following
in the terminal of your client. |tip.run-all.root|:

.. include:: ../.res/code/pmm-admin.config.server.url.txt

If you change the default port **80** when :ref:`running PMM Server
<deploy-pmm.server.installing>`, specify the new port number after the IP
address of |pmm-server|. For example:

.. include:: ../.res/code/pmm-admin.config.server.url.port.txt

.. include:: ../.res/contents/important.port.txt

.. seealso::

   What other options can I pass to |pmm-admin.config|?
      Run |pmm-admin.config| |opt.help|

.. include:: ../.res/replace.txt
