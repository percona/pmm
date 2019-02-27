.. _pmm-admin.config:

:ref:`Configuring PMM Client <pmm-admin.config>`
================================================================================

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
  mapped from client address via NAT or port forwarding
  By default, it is set to the client address.

|opt.client-address|
  Specify the client address,
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
