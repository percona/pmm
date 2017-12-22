.. _security:

================================================================================
Security Features in |pmm.name|
================================================================================

You can protect |pmm| from unauthorized access using the following security
features:

- HTTP password protection adds authentication
  when accessing the |pmm-server| web interface

- SSL encryption secures traffic between |pmm-client| and |pmm-server|

Enabling Password Protection
================================================================================

You can set the password for accessing the |pmm-server| web interface by passing
the :term:`SERVER_PASSWORD <SERVER_PASSWORD (Option)>` environment variable when
:ref:`creating and running the PMM Server container <server-container>`.

To set the environment variable, use the ``-e`` option.

By default, the user name is ``pmm``. You can change it by passing the
:term:`SERVER_USER <SERVER_USER (Option)>` environment variable. For example:

|tip.run-all.root|

.. include:: .res/code/sh.org
   :start-after: +docker.run.server-user.example+
   :end-before: #+end-block

|pmm-client| uses the same credentials to communicate with |pmm-server|.
If you set the user name and password as described,
specify them when :ref:`connect-client`:

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.config.server.server-user.server-password+
   :end-before: #+end-block

Enabling SSL Encryption
================================================================================

You can encrypt traffic between |pmm-client| and |pmm-server| using SSL
certificates.

Valid certificates
--------------------------------------------------------------------------------

To use a valid SSL certificate, mount the directory with the certificate
files to |srv.nginx| when :ref:`running the PMM Server container
<server-container>`.

.. include:: .res/code/sh.org
   :start-after: +docker.run.d.p.volumes.from.name.v.restart+
   :end-before: #+end-block

The directory (|etc.pmm-certs| in this example) that you intend to mount must
contain the following files:

- |certificate.crt|
- |certificate.key|
- |ca-certs.pem|
- |dhparam.pem|

.. note:: To enable SSL encryption, The container publishes port *443* instead
          of *80*.

Alternatively, you can use |docker.cp| to copy the files to an already existing |opt.pmm-server|
container.

.. include:: .res/code/sh.org
   :start-after: +docker.cp.certificate-crt.pmm-server+
   :end-before: #+end-block

This example assumes that you have changed to the directory that contains the
certificate files.

Self-signed certificates
--------------------------------------------------------------------------------

The |pmm-server| images at `percona/pmm-server` |docker| repository already
include self-signed certificates. To be able to use them make sure to publish
the container's port *443* to the host's port *443* when running the
|docker.run| command.

.. include:: .res/code/sh.org
   :start-after: +docker.run.d.p.443.volumes-from.name.restart+
   :end-before: #+end-block

Enabling SSL when connecting |pmm-client| to |pmm-server|
--------------------------------------------------------------------------------

Then, you need to enable SSL when :ref:`connect-client`.
If you purchased the certificate from a certificate authority (CA):

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.config.server.server-ssl+
   :end-before: #+end-block

If you generated a self-signed certificate:

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.config.server.server-insecure-ssl+
   :end-before: #+end-block
		
Combining Security Features
================================================================================

You can enable both HTTP password protection and SSL encryption
by combining the corresponding options.

The following example shows how you might
:ref:`run the PMM Server container <server-container>`:

.. include:: .res/code/sh.org
   :start-after: +docker.run.example+
   :end-before: #+end-block
		 
The following example shows how you might
:ref:`connect to PMM Server <connect-client>`:

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.config.example+
   :end-before: #+end-block

To see which security features are enabled,
run either ``pmm-admin ping``, ``pmm-admin config``,
``pmm-admin info``, or ``pmm-admin list``
and look at the server address field. For example:

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.ping+
   :end-before: #+end-block
	     
.. include:: .res/replace/name.txt
.. include:: .res/replace/fragment.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/option.txt
