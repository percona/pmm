#######
MongoDB
#######

.. _pmm.pmm-admin.mongodb.pass-ssl-parameter:

********************************************************
Passing SSL parameters to the mongodb monitoring service
********************************************************

SSL/TLS related parameters are passed to an SSL enabled MongoDB server as
monitoring service parameters along with the ``pmm-admin add`` command when adding
the MongoDB monitoring service.

Run this command as root or by using the ``sudo`` command

.. code-block:: bash

   pmm-admin add mongodb --tls

**Supported SSL/TLS Parameters**

``--tls``
   Enable a TLS connection with mongo server

``--tls-skip-verify``
   Skip TLS certificates validation

.. seealso::

   :ref:`pmm.ref.pmm-admin`