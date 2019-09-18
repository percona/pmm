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
of the :option:`--server-url` parameter to the |pmm-admin.config| command, and
allow using self-signed certificates with :option:`--server-insecure-tls`.

.. note:: The :option:`--server-url` argument should include ``https://`` prefix
         and PMM Server credentials, which are **admin/admin** by default, if
         not changed at first PMM Server GUI access.

|tip.run-this.root|

.. include:: ../.res/code/pmm-admin.config.server.url.port.txt

For example, if your |pmm-server| is running on `192.168.100.1`, you have
installed |pmm-client| on a machine with IP `192.168.200.1`, and didn't change
default PMM Server credentials, run the following in the terminal of your
client. |tip.run-all.root|:

.. include:: ../.res/code/pmm-admin.config.server.url.txt

If you change the default port **443** when :ref:`running PMM Server
<deploy-pmm.server.installing>`, specify the new port number after the IP
address of |pmm-server|.

.. only:: showhidden

	.. include:: ../.res/contents/important.port.txt
	
	.. seealso::
	
	   What other options can I pass to |pmm-admin.config|?
	      Run |pmm-admin.config| |opt.help|
	
	.. _pmm-admin-config.additional-options:
	
	`pmm-admin config additional options <client-config.html#pmm-admin-config-additional-options>`_
	===============================================================================================
	
	
	Use the |pmm-admin.config| command to configure
	how |pmm-client| communicates with |pmm-server|.
	
	.. _pmm-admin.config.usage:
	
	.. rubric:: USAGE
	
	|tip.run-this.root|.
	
	.. _code.pmm-admin.config.options:
	
	.. include:: ../.res/code/pmm-admin.config.options.txt
			
	.. _pmm-admin.config.options:
	
	.. rubric:: OPTIONS
	
	The following options can be used with the |pmm-admin.config| command:
	
	|opt.bind-address|
	  Specify the bind address,
	  which is also the local (private) address
	  mapped from client-address via NAT or port forwarding
	  By default, it is set to the client-address.
	
	|opt.client-address|
	  Specify the client-address,
	which is also the remote (public) address for this system.
	  By default, it is automatically detected via request to server.
	
	|opt.client-name|
	  Specify the client name.
	  By default, it is set to the host name.
	
	|opt.force|
	  Force to set the client name on initial setup
	  after uninstall with unreachable server.
	
	|opt.server|
	  Specify the address of the |pmm-server| host.
	  If necessary, you can also specify the port after colon, for example::
	
	   pmm-admin config --server 192.168.100.6:8080
	
	  By default, port 80 is used with SSL disabled,
	  and port 443 when SSL is enabled.
	
	|opt.server-insecure-ssl|
	  Enable insecure SSL (self-signed certificate).
	
	|opt.server-password|
	  Specify the HTTP password configured on |pmm-server|.
	
	|opt.server-ssl|
	  Enable SSL encryption for connection to |pmm-server|.
	
	|opt.server-user|
	  Specify the HTTP user configured on |pmm-server| (default is ``pmm``).
	
	You can also use
	:ref:`global options that apply to any other command <pmm-admin.options>`.
	
	For more information, run |pmm-admin.config| --help.

.. include:: ../.res/replace.txt
