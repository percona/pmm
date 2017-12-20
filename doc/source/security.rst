.. _security:

================================================================================
Security Features in |pmm.name|
================================================================================

You can protect |pmm| from unauthorized access
using the following security features:

- HTTP password protection adds authentication
  when accessing the |pmm-server| web interface

- SSL encryption secures traffic between |pmm-client| and |pmm-server|

Enabling Password Protection
================================================================================

You can set the password for accessing the |pmm-server| web interface
by passing the :term:`SERVER_PASSWORD <SERVER_PASSWORD (Option)>` environment variable
when :ref:`creating and running the PMM Server container <server-container>`.

To set the environment variable, use the ``-e`` option.

By default, the user name is ``pmm``. You can change it by passing the
:term:`SERVER_USER <SERVER_USER (Option)>` environment variable. For example:

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

You can encrypt traffic between |pmm-client| and |pmm-server| using
SSL certificates.

1. Buy or generate SSL certificate files for |pmm|.

   For example, you can generate necessary self-signed certificate files
   into the |etc.pmm-certs| directory using the following commands:

   .. include:: .res/code/sh.org
      :start-after: +openssl.dhparam&openssl.req+
      :end-before: #+end-block

   .. note:: The |dhparam.pem| file is not required.
      It can take a lot of time to generate, so you can skip it.

      The |server.key| and |server.crt| files
      must be named exactly as shown.
      Files with other names will be ignored.

#. Mount the directory with the certificate files into |etc.nginx.ssl|
   when :ref:`running the PMM Server container <server-container>`:

   |tip.run-this.root|

   .. include:: .res/code/sh.org
      :start-after: +docker.run.example/etc.pmm-certs+
      :end-before: #+end-block
		
   .. note:: Note that the container should expose port 443
      instead of 80 to enable SSL encryption.

#. Enable SSL when :ref:`connect-client`.
   If you purchased the certificate from a certificate authority (CA):

   .. include:: .res/code/sh.org
      :start-after: +pmm-admin.config.server.server-ssl+
      :end-before: #+end-block

   .. If you generated a self-signed certificate:

   .. .. include:: .res/code/sh.org
   ..   :start-after: +pmm-admin.config.server.server-insecure-ssl+
   ..   :end-before: #+end-block

   If you have a self-signed certificate, run |docker.cp| to make the
   certificate files available to your |opt.pmm-server| container.

   .. include:: .res/code/sh.org
      :start-after: +docker.cp.certificate-crt.pmm-server+
      :end-before: #+end-block

   This example assumes that you have changed into the directory that contains the certificate files.
		
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
