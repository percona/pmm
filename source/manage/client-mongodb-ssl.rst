--------------------------------------------------------------------------------
MongoDB
--------------------------------------------------------------------------------

.. _pmm.pmm-admin.mongodb.pass-ssl-parameter:

`Passing SSL parameters to the mongodb monitoring service <pmm-admin.html#pmm-pmm-admin-mongodb-pass-ssl-parameter>`_
----------------------------------------------------------------------------------------------------------------------

SSL/TLS related parameters are passed to an SSL enabled |mongodb| server as
monitoring service parameters along with the |pmm-admin.add| command when adding
the |opt.mongodb-metrics| monitoring service.

|tip.run-this.root|

.. include:: ../.res/code/pmm-admin.add.mongodb-metrics.mongodb-tls.txt
   
.. list-table:: Supported SSL/TLS Parameters
   :widths: 25 75
   :header-rows: 1

   * - Parameter
     - Description
   * - |opt.mongodb-tls|
     - Enable a TLS connection with mongo server
   * - |opt.mongodb-tls-ca|  *string*
     - A path to a PEM file that contains the CAs that are trusted for server connections.
       *If provided*: MongoDB servers connecting to should present a certificate signed by one of these CAs.
       *If not provided*: System default CAs are used.
   * - |opt.mongodb-tls-cert| *string*
     - A path to a PEM file that contains the certificate and, optionally, the private key in the PEM format.
       This should include the whole certificate chain.
       *If provided*: The connection will be opened via TLS to the |mongodb| server.
   * - |opt.mongodb-tls-disable-hostname-validation|
     - Do hostname validation for the server connection.
   * - |opt.mongodb-tls-private-key| *string*
     - A path to a PEM file that contains the private key (if not contained in the :option:`mongodb.tls-cert` file).

.. include:: ../.res/contents/note.option.mongodb-queries.txt
       
.. include:: ../.res/code/mongod.dbpath.profile.slowms.ratelimit.txt

.. _pmm-admin-add-linux-metrics:

.. include:: ../.res/replace.txt
