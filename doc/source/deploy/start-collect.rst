.. _start-collect:

========================
Starting Data Collection
========================

After you :ref:`connect the client to PMM Server <connect-client>`,
enable data collection from the database instance
using the ``pmm-admin add`` command.

For more information about ``pmm-admin``, see :ref:`pmm-admin`.

To enable general system metrics, MySQL metrics, and query analytics, run:

.. code-block:: bash

   sudo pmm-admin add mysql

To enable general system metrics and MongoDB metrics, run:

.. code-block:: bash

   sudo pmm-admin add mongodb

To enable ProxySQL performance metrics, run:

.. code-block:: bash

   sudo pmm-admin add proxysql:metrics

To see what is being monitored, run:

.. code-block:: bash

   $ sudo pmm-admin list

For example, if you enable general OS and MongoDB metrics monitoring,
output should be similar to the following:

.. code-block:: bash

   $ sudo pmm-admin list
   pmm-admin 1.1.0

   PMM Server      | 192.168.100.1
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.200.1
   Service manager | linux-systemd

   ---------------- ------------- ------------ -------- --------------- --------
   METRIC SERVICE   NAME          CLIENT PORT  RUNNING  DATA SOURCE     OPTIONS
   ---------------- ------------- ------------ -------- --------------- --------
   linux:metrics    ubuntu-amd64  42000        YES      -
   mongodb:metrics  ubuntu-amd64  42003        YES      localhost:27017

For more information about adding instances, run ``pmm-admin add --help``.

Next Steps
==========

After you set up data collection,
you can :ref:`install PMM Client <install-client>`
on another database instance,
:ref:`connect it to PMM Server <connect-client>`,
and enable data collection in a similar way.

