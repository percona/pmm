.. _pmm.security.ssl-encryption.enabling:

`Enabling SSL Encryption <security.html#pmm-security-ssl-encryption-enabling>`_
================================================================================

You can encrypt traffic between |pmm-client| and |pmm-server| using SSL
certificates. TLS protocol versions compatibility is as follows:

* TLS v1.0 - deprecated,
* TLS v1.1 - deprecated,
* TLS v1.2 - supported.

.. _pmm.security.valid-certificate:

`Valid certificates <security.html#pmm-security-valid-certificate>`_
--------------------------------------------------------------------------------

To use a valid SSL certificate, mount the directory with the certificate
files to |srv.nginx| when :ref:`running the PMM Server container
<server-container>`.

.. include:: ../.res/code/docker.run.d.p.volumes.from.name.v.restart.txt

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

.. include:: ../.res/code/docker.cp.certificate-crt.pmm-server.txt

This example assumes that you have changed to the directory that contains the
certificate files.

.. _pmm.security.certificate.self-signed:

`Self-signed certificates <security.html#pmm-security-certificate-self-signed>`_
---------------------------------------------------------------------------------

The |pmm-server| images (|docker|, OVF, and AMI) already include self-signed
certificates. To be able to use them in your |docker| container, make sure to
publish the container's port *443* to the host's port *443* when running the
|docker.run| command.

.. include:: ../.res/code/docker.run.d.p.443.volumes-from.name.restart.txt

.. _pmm.security.pmm-client.pmm-server.ssl.enabling:

`Enabling SSL when connecting PMM Client to PMM Server <security.html#pmm-security-pmm-client-pmm-server-ssl-enabling>`_
-------------------------------------------------------------------------------------------------------------------------

Then, you need to enable SSL when :ref:`connecting a PMM Client to a PMM Server
<deploy-pmm.client_server.connecting>`.  If you purchased the certificate from a
certificate authority (CA):

.. include:: ../.res/code/pmm-admin.config.server.server-ssl.txt

If you generated a self-signed certificate:

.. include:: ../.res/code/pmm-admin.config.server.server-insecure-ssl.txt

.. include:: ../.res/replace.txt
