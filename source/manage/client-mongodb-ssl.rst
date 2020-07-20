--------------------------------------------------------------------------------
MongoDB
--------------------------------------------------------------------------------

.. _pmm.pmm-admin.mongodb.pass-ssl-parameter:

`Passing SSL parameters to the mongodb monitoring service <pmm-admin.html#pmm-pmm-admin-mongodb-pass-ssl-parameter>`_
----------------------------------------------------------------------------------------------------------------------

SSL/TLS related parameters are passed to an SSL enabled MongoDB server as
monitoring service parameters along with the ``pmm-admin add`` command when adding
the MongoDB monitoring service.

Run this command as root or by using the ``sudo`` command

.. include:: ../.res/code/pmm-admin.add.mongodb-metrics.mongodb-tls.txt
   
.. list-table:: Supported SSL/TLS Parameters
   :widths: 25 75
   :header-rows: 1

   * - Parameter
     - Description
   * - ``--mongodb.tls``
     - Enable a TLS connection with mongo server
   * - ``--mongodb.tls-ca``  *string*
     - A path to a PEM file that contains the CAs that are trusted for server connections.
       *If provided*: MongoDB servers connecting to should present a certificate signed by one of these CAs.
       *If not provided*: System default CAs are used.
   * - ``--mongodb.tls-cert`` *string*
     - A path to a PEM file that contains the certificate and, optionally, the private key in the PEM format.
       This should include the whole certificate chain.
       *If provided*: The connection will be opened via TLS to the MongoDB server.
   * - ``--mongodb.tls-disable-hostname-validation``
     - Do hostname validation for the server connection.
   * - ``--mongodb.tls-private-key`` *string*
     - A path to a PEM file that contains the private key (if not contained in the ``mongodb.tls-cert`` file).

.. include:: /.res/code/mongod.dbpath.profile.slowms.ratelimit.txt



