.. _pmm.security:

Security Features in |pmm.name|
********************************************************************************

You can protect |pmm| from unauthorized access using the following security
features:

- SSL encryption secures traffic between |pmm-client| and |pmm-server|
- HTTP password protection adds authentication when accessing the |pmm-server|
  web interface
- Keep PMM Server isolated from the internet, where possible.

|chapter.toc|

.. contents::
   :local:
   :depth: 2

.. _pmm.security.ssl-encryption.enabling:

`Enabling SSL Encryption <security.html#pmm-security-ssl-encryption-enabling>`_
================================================================================

You can encrypt traffic between |pmm-client| and |pmm-server| using SSL
certificates.

.. _pmm.security.valid-certificate:

`Valid certificates <security.html#pmm-security-valid-certificate>`_
--------------------------------------------------------------------------------

To use a valid SSL certificate, mount the directory with the certificate
files to |srv.nginx| when :ref:`running the PMM Server container
<server-container>`.

.. include:: .res/code/docker.run.d.p.volumes.from.name.v.restart.txt

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

.. include:: .res/code/docker.cp.certificate-crt.pmm-server.txt

This example assumes that you have changed to the directory that contains the
certificate files.

.. _pmm.security.certificate.self-signed:

`Self-signed certificates <security.html#pmm-security-certificate-self-signed>`_
---------------------------------------------------------------------------------

The |pmm-server| images (|docker|, OVF, and AMI) already include self-signed
certificates. To be able to use them in your |docker| container, make sure to
publish the container's port *443* to the host's port *443* when running the
|docker.run| command.

.. include:: .res/code/docker.run.d.p.443.volumes-from.name.restart.txt

.. _pmm.security.pmm-client.pmm-server.ssl.enabling:

`Enabling SSL when connecting PMM Client to PMM Server <security.html#pmm-security-pmm-client-pmm-server-ssl-enabling>`_
-------------------------------------------------------------------------------------------------------------------------

Then, you need to enable SSL when :ref:`connecting a PMM Client to a PMM Server
<deploy-pmm.client_server.connecting>`.  If you purchased the certificate from a
certificate authority (CA):

.. include:: .res/code/pmm-admin.config.server.server-ssl.txt

If you generated a self-signed certificate:

.. include:: .res/code/pmm-admin.config.server.server-insecure-ssl.txt

.. _pmm.security.password-protection.enabling:

`Enabling Password Protection <security.html#pmm-security-password-protection-enabling>`_
==========================================================================================

You can set the password for accessing the |pmm-server| web interface by passing
the :option:`SERVER_PASSWORD` environment variable when
:ref:`creating and running the PMM Server container <server-container>`.

To set the environment variable, use the ``-e`` option.

By default, the user name is ``pmm``. You can change it by passing the
:option:`SERVER_USER` environment variable. Note that the
following example uses an insecure port 80 which is typically used for HTTP
connections.

|tip.run-all.root|.

.. include:: .res/code/docker.run.server-user.example.txt

|pmm-client| uses the same credentials to communicate with |pmm-server|.  If you
set the user name and password as described, specify them when :ref:`connecting
a PMM Client to a PMM Server <deploy-pmm.client_server.connecting>`:

.. include:: .res/code/pmm-admin.config.server.server-user.server-password.txt

.. _pmm.security.combining:

`Combining Security Features <security.html#pmm-security-combining>`_
================================================================================

You can enable both HTTP password protection and SSL encryption by combining the
corresponding options.

The following example shows how you might :ref:`run the PMM Server container
<server-container>`:

.. include:: .res/code/docker.run.example.txt

The following example shows how you might :ref:`connect to PMM Server
<deploy-pmm.client_server.connecting>`:

.. include:: .res/code/pmm-admin.config.example.txt

To see which security features are enabled, run either |pmm-admin.ping|,
|pmm-admin.config|, |pmm-admin.info|, or |pmm-admin.list| and look at the server
address field. For example:

.. include:: .res/code/pmm-admin.ping.txt




Enable HTTPS secure cookies in Grafana
======================================

The following assumes you are using a Docker container for PMM Server.

1. Edit ``/etc/grafana/grafana.ini``

2. Enable ``cookie_secure`` and set the value to ``true``

3. Restart Grafana: ``supervisorctl restart grafana``

.. seealso::

  https://grafana.com/docs/grafana/latest/administration/configuration/#cookie_secure


.. include:: .res/replace.txt
