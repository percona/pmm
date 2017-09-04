.. _start-collect:

========================
Starting Data Collection
========================

After you :ref:`connect the client to PMM Server <connect-client>`,
enable data collection from the database instance
by :ref:`adding a monitoring service <pmm-admin-add>`.

To enable general system metrics, MySQL metrics,
and MySQL query analytics, run:

.. code-block:: bash

   sudo pmm-admin add mysql

To enable general system metrics, MongoDB metrics,
and MongoDB query analytics, run:

.. code-block:: bash

   sudo pmm-admin --dev-enable add mongodb

.. note:: MongoDB query analytics is experimental
   and requires the ``--dev-enable`` option when adding.
   Without this option, only general system metrics and MongoDB metrics
   will be added.

To enable ProxySQL performance metrics, run:

.. code-block:: bash

   sudo pmm-admin add proxysql:metrics

To see what is being monitored, run:

.. code-block:: bash

   $ sudo pmm-admin list

For example, if you enable general OS and MongoDB metrics monitoring,
output should be similar to the following:

.. code-block:: text

   $ sudo pmm-admin list

   ...

   PMM Server      | 192.168.100.1
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.200.1
   Service manager | linux-systemd

   ---------------- ----------- ----------- -------- ---------------- --------
   SERVICE TYPE     NAME        LOCAL PORT  RUNNING  DATA SOURCE      OPTIONS
   ---------------- ----------- ----------- -------- ---------------- --------
   linux:metrics    mongo-main  42000       YES      -
   mongodb:metrics  mongo-main  42003       YES      localhost:27017

For more information about adding instances, run ``pmm-admin add --help``.

Next Steps
==========

After you set up data collection,
you can :ref:`install PMM Client <install-client>`
on another database instance,
:ref:`connect it to PMM Server <connect-client>`,
and enable data collection in a similar way.

