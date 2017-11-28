.. _start-collect:

================================================================================
Starting Data Collection
================================================================================

After you :ref:`connect the client to PMM Server <connect-client>`,
enable data collection from the database instance
by :ref:`adding a monitoring service <pmm-admin.add>`.

To enable general system metrics, |mysql| metrics,
and |mysql| query analytics, run:

.. code-block:: bash

   $ pmm-admin add mysql

To enable general system metrics, |mongodb| metrics,
and |mongodb| query analytics, run as root:

.. code-block:: bash

   $ pmm-admin add mongodb

To enable |proxysql| performance metrics, run as root:

.. code-block:: bash

   $ pmm-admin add proxysql:metrics

To see what is being monitored, run as root:

.. code-block:: bash

   $ pmm-admin list

For example, if you enable general OS and |mongodb| metrics monitoring,
output should be similar to the following:

.. code-block:: text

   $ pmm-admin list

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

For more information about adding instances, run
|pmm-admin.add|
|opt.help|.

Next Steps
================================================================================

After you set up data collection,
you can :ref:`install PMM Client <install-client>`
on another database instance,
:ref:`connect it to PMM Server <connect-client>`,
and enable data collection in a similar way.

.. include:: ../.res/replace/name.txt
.. include:: ../.res/replace/program.txt
.. include:: ../.res/replace/option.txt
