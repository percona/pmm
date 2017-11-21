.. _security:

======================================================
Security Features in Percona Monitoring and Management
======================================================

You can protect PMM from unauthorized access
using the following security features:

- HTTP password protection adds authentication
  when accessing the *PMM Server* web interface

- SSL encryption secures traffic between *PMM Client* and *PMM Server*

Enabling Password Protection
============================

You can set the password for accessing the *PMM Server* web interface
by passing the ``SERVER_PASSWORD`` environment variable
when :ref:`creating and running the PMM Server container <server-container>`.
To set the environment variable, use the ``-e`` option.
For example, to set the password to ``pass1234``::

 -e SERVER_PASSWORD=pass1234

By default, the user name is ``pmm``.
You can change it by passing the ``SERVER_USER`` variable.

For example:

.. code-block:: bash

   $ docker run -d -p 80:80 \
     --volumes-from pmm-data \
     --name pmm-server \
     -e SERVER_USER=jsmith \
     -e SERVER_PASSWORD=pass1234 \
     --restart always \
     percona/pmm-server:latest

*PMM Client* uses the same credentials to communicate with *PMM Server*.
If you set the user name and password as described,
specify them when :ref:`connect-client`:

.. code-block:: bash

   $ sudo pmm-admin config --server 192.168.100.1 --server-user jsmith --server-password pass1234

Enabling SSL Encryption
=======================

You can encrypt traffic between *PMM Client* and *PMM Server*
using SSL certificates.

1. Buy or generate SSL certificate files for PMM.

   For example, you can generate necessary self-signed certificate files
   into the :file:`/etc/pmm-certs` directory using the following commands:

   .. code-block:: text

      # openssl dhparam -out /etc/pmm-certs/dhparam.pem 4096
      # openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /etc/pmm-certs/server.key -out /etc/pmm-certs/server.crt
      Generating a 2048 bit RSA private key
      ....................................+++
      ....+++
      writing new private key to '/etc/pmm-certs/server.key'
      -----
      You are about to be asked to enter information that will be incorporated
      into your certificate request.
      What you are about to enter is what is called a Distinguished Name or a DN.
      There are quite a few fields but you can leave some blank
      For some fields there will be a default value,
      If you enter '.', the field will be left blank.
      -----
      Country Name (2 letter code) [XX]:US
      State or Province Name (full name) []:North Carolina
      Locality Name (eg, city) [Default City]:Raleigh
      Organization Name (eg, company) [Default Company Ltd]:Percona
      Organizational Unit Name (eg, section) []:PMM
      Common Name (eg, your name or your server's hostname) []:centos7.vm
      Email Address []:jsmith@example.com

   .. note:: The :file:`dhparam.pem` file is not required.
      It can take a lot of time to generate, so you can skip it.

   .. note:: The :file:`server.key` and :file:`server.crt` files
      must be named exactly as shown.
      Files with other names will be ignored.

#. Mount the directory with the certificate files into :file:`/etc/nginx/ssl`
   when :ref:`running the PMM Server container <server-container>`:

   .. code-block:: bash

      $ docker run -d -p 443:443 \
        --volumes-from pmm-data \
        --name pmm-server \
        -v /etc/pmm-certs:/etc/nginx/ssl \
        --restart always \
        percona/pmm-server:latest

   .. note:: Note that the container should expose port 443
      instead of 80 to enable SSL encryption.

#. Enable SSL when :ref:`connect-client`.
   If you purchased the certificate from a certificate authority (CA):

   .. code-block:: bash

      $ sudo pmm-admin config --server 192.168.100.1 --server-ssl

   If you generated a self-signed certificate:

   .. code-block:: bash

      $ sudo pmm-admin config --server 192.168.100.1 --server-insecure-ssl

Combining Security Features
===========================

You can enable both HTTP password protection and SSL encryption
by combining the corresponding options.

The following example shows how you might
:ref:`run the PMM Server container <server-container>`:

.. code-block:: bash

   $ docker run -d -p 443:443 \
     --volumes-from pmm-data \
     --name pmm-server \
     -e SERVER_USER=jsmith \
     -e SERVER_PASSWORD=pass1234 \
     -v /etc/pmm-certs:/etc/nginx/ssl \
     --restart always \
     percona/pmm-server:latest

The following example shows how you might
:ref:`connect to PMM Server <connect-client>`:

.. code-block:: bash

   $ sudo pmm-admin config --server 192.168.100.1 --server-user jsmith --server-password pass1234 --server-insecure-ssl

To see which security features are enabled,
run either ``pmm-admin ping``, ``pmm-admin config``,
``pmm-admin info``, or ``pmm-admin list``
and look at the server address field. For example:

.. code-block:: text

   $ sudo pmm-admin ping
   OK, PMM server is alive.

   PMM Server      | 192.168.100.1 (insecure SSL, password-protected)
   Client Name     | centos7.vm
   Client Address  | 192.168.200.1

