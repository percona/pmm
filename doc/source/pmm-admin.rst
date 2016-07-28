.. _pmm-admin:

===================
Managing PMM Client
===================

Use the ``pmm-admin`` tool to manage *PMM Client*.

.. note:: The ``pmm-admin`` tool requires root access
   (either logged in as a user with root privileges
   or be able to run commands with ``sudo``).

Use the ``--help`` option to view the built-in help.
For example, you can view all available commands and options
by running the following:

.. prompt:: bash

   sudo pmm-admin --help

.. contents::
   :local:
   :depth: 1

.. _pmm-admin-add:

Adding instances
================

Use the ``pmm-admin add`` command to add monitoring instances.

.. _pmm-admin-add-os:

System metrics
--------------

**To enable general system metrics monitoring:**

.. prompt:: bash

   sudo pmm-admin add os

This creates the ``pmm-os-exporter-42000`` service
that collects local system metrics for this particular OS instance.

.. note:: It should be able to detect the local PMM Client name,
   but you can also specify it explicitely as an argument.

For more information, run ``sudo pmm-admin add os --help``

.. _pmm-admin-add-queries:

Query analytics
---------------

**To enable MySQL query analytics:**

.. prompt:: bash

   sudo pmm-admin add queries

This creates the ``pmm-queries-exporter-42001`` service
that is able to collect QAN data for multiple remote MySQL server instances.

The ``pmm-admin`` tool will attempt to automatically detect
the local MySQL instance and MySQL superuser credentials.
You can use options to provide this information for ``pmm-admin``
if it is not able to auto-detect.
You can also specify the ``--create-user`` option to create a dedicated
``pmm-queries`` user on the MySQL host that you want to monitor.
This user will be given all the necessary privileges for monitoring,
and is recommended over using the MySQL superuser.

For example, to set up remote monitoring of QAN data
on a MySQL server located at 192.168.200.2,
use a command similar to the following:

.. prompt:: bash

   sudo pmm-admin add queries --user root --password root --host 192.168.200.2 --create-user

QAN can use either the slow query log or Performance Schema as the source.
By default, it chooses the slow query log for a local MySQL instance
and Performance Schema otherwise.
For more information about the differences, see :ref:`perf-schema`.

You can explicitely set the query source when adding a QAN instance
using the ``--query-source`` option.

For more information, run ``sudo pmm-admin add queries --help``

.. _pmm-admin-add-mysql:

MySQL metrics
-------------

**To enable MySQL metrics monitoring:**

.. prompt:: bash

   sudo pmm-admin add mysql

This creates the following services

* ``pmm-mysql-exporter-42002``
* ``pmm-mysql-exporter-42003``
* ``pmm-mysql-exporter-42004``

.. note:: Multiple services are required to efficiently collect metrics
   with different resolution (1 second, 5 seconds, and 60 seconds).

The ``pmm-admin`` tool will attempt to automatically detect
the local MySQL instance and MySQL superuser credentials.
You can use options to provide this information for ``pmm-admin``
if it is not able to auto-detect.
You can also specify the ``--create-user`` option to create a dedicated
``pmm-mysql`` user on the MySQL host that you want to monitor.
This user will be given all the necessary privileges for monitoring,
and is recommended over using the MySQL superuser.

For example,
to set up remote monitoring of MySQL metrics
on a server located at 192.168.200.3,
use a command similar to the following:

.. prompt:: bash

   sudo pmm-admin add mysql --user root --password root --host 192.168.200.3 --create-user

For more information, run ``sudo pmm-admin add mysql --help``.

.. _pmm-admin-add-mongodb:

MongoDB metrics
---------------

**To enable MongoDB metrics monitoring:**

.. prompt:: bash

   sudo pmm-admin add mongodb

This creates the ``pmm-mongodb-exporter-42005`` service
that collects local MongoDB metrics for this particular MongoDB instance.

.. note:: It should be able to detect the local PMM Client name,
   but you can also specify it explicitely as an argument.

You can use options to specify the MongoDB replica set, cluster name,
and node type. For example:

.. prompt:: bash

   sudo pmm-admin add mongodb --replset repl1 --cluster cluster1 --nodetype mongod 

For more information, run ``sudo pmm-admin add mongodb --help``

.. _pmm-admin-rm:

Removing instances
==================

Use the ``pmm-admin rm`` command to remove monitoring instances.
Specify the instance's type and name.
You can see the names of instances by running ``sudo pmm-admin list``.

For example, to remove a MySQL instance designated by ``ubuntu-amd4``
from monitoring, run the following:

.. prompt:: bash

   sudo pmm-admin rm mysql ubuntu-amd64

For more information, run ``sudo pmm-admin rm [command] --help``.

.. _pmm-admin-list:

Listing monitored instances
===========================

To see what is being monitored, run the following:

.. prompt:: bash

   sudo pmm-admin list

The output provides the following info:

* Version of ``pmm-admin``
* *PMM Server* host address, and local host name and address
  (this can be configured using |pmm-admin-config|_)
* System manager that ``pmm-admin`` uses to manage PMM services
* A table that lists all services currently managed by ``pmm-admin``,
  with basic information about each service

For example, if you enable general OS and MongoDB metrics monitoring,
output should be similar to the following:

.. code-block:: bash
   :emphasize-lines: 1

   $ sudo pmm-admin list
   pmm-admin 1.0.2

   PMM Server      | 192.168.100.6
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.100.6
   Service manager | linux-systemd

   --------------- ------------- ------------ -------- ---------------- --------
   METRIC SERVICE  NAME          CLIENT PORT  RUNNING  DATA SOURCE      OPTIONS 
   --------------- ------------- ------------ -------- ---------------- --------
   os              ubuntu-amd64  42000        YES      -                        
   mongodb         ubuntu-amd64  42005        YES      localhost:27017 

.. _pmm-admin-config:

Configuring PMM Client
======================

Use the ``pmm-admin config`` command to configure
how ``pmm-admin`` communicates with *PMM Server*.

The following options are available:

--client-addr string   Client host address
--client-name string   Client host name (node identifier in Consul)
--server-addr string   PMM Server host address

For more information, run ``sudo pmm-admin config --help``

.. _pmm-admin-info:

Getting information about PMM Client
====================================

Use the ``pmm-admin info`` command to display basic info about ``pmm-admin``.
The output is also displayed before the table with services
when you run |pmm-admin-list|_.

The following example shows the output if both *PMM Server* and *PMM Client*
are on the same host named ``ubuntu-amd64``,
which uses ``systemd`` to manage services.

.. code-block:: bash
   :emphasize-lines: 1

   $ sudo pmm-admin info
   pmm-admin 1.0.2

   PMM Server      | 192.168.100.6
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.100.6
   Service manager | linux-systemd

This can be configured using |pmm-admin-config|_.

For more information, run ``sudo pmm-admin info --help``.

.. _pmm-admin-check-network:

Checking network connectivity
=============================

Use the ``pmm-admin check-network`` command to run tests
that verify connectivity between *PMM Client* and *PMM Server*.
The tests are performed both ways,
with results separated accordingly:

* Client > Server

  Pings Consul API, Query Analytics API, and Prometheus API
  to make sure they are alive and reachable.

  Performs a connection performance test to see the latency
  from *PMM Client* to *PMM Server*.

* Server > Client

  Checks the status of Prometheus endpoints
  and makes sure it can scrape metrics from corresponding exporters.

  Successful pings of *PMM Server* from *PMM Client*
  do not mean that Prometheus is able to scrape from exporters.
  If the output shows some endpoints in problem state,
  make sure that the corresponding service is running
  (see |pmm-admin-list|_).
  If the services that correspond to problematic endpoints are running,
  make sure that the firewall settings on *PMM Client*
  allow incoming connections for corresponding ports.

The ``pmm-admin check-network`` command has one option (``--no-emoji``),
which replaces emojis with words in the status.

The following example shows output without emojis:

.. code-block:: bash
   :emphasize-lines: 1

   $ sudo pmm-admin check-network --no-emoji
   PMM Network Status
   
   Server | 192.168.100.6
   Client | 192.168.100.6
   
   * Client > Server
   --------------- -------------
   SERVICE         CONNECTIVITY 
   --------------- -------------
   Consul API      OK           
   QAN API         OK           
   Prometheus API  OK           
   
   Connection duration | 166.689µs
   Request duration    | 364.527µs
   Full round trip     | 531.216µs
   
   * Server > Client
   -------- ------------- ---------------------- -------------
   METRIC   NAME          PROMETHEUS ENDPOINT    REMOTE STATE 
   -------- ------------- ---------------------- -------------
   os       ubuntu-amd64  192.168.100.6:42000    OK           
   mysql    ubuntu-amd64  192.168.100.6:42002    OK           
   mysql    ubuntu-amd64  192.168.100.6:42003    OK           
   mysql    ubuntu-amd64  192.168.100.6:42004    OK           
   mongodb  ubuntu-amd64  192.168.100.6:42005    PROBLEM

For more information, run ``sudo pmm-admin check-network --help``.

.. _pmm-admin-ping:

Pinging PMM Server
==================

Use the ``pmm-admin ping`` command to ping *PMM Server*.
If the ping is successful, it returns ``OK``.

For more information, run ``sudo pmm-admin ping --help``.

.. _pmm-admin-start:
.. _pmm-admin-stop:

Starting and stopping metric services
=====================================

Services that you add using |pmm-admin-add|_
can be started and stopped manually
using ``pmm-admin start`` and ``pmm-admin stop``.

For example, to start the ``mongodb`` service on host ``ubuntu-amd64``:

.. prompt:: bash

   sudo pmm-admin start mongodb ubuntu-amd64

To stop the ``mysql`` service on host ``centos-amd64``:

.. prompt:: bash

   sudo pmm-admin stop os centos-amd64

To stop all services managed by this ``pmm-admin``:

.. prompt:: bash

   sudo pmm-admin stop --all

For more information,
run ``sudo pmm-admin start --help`` or ``sudo pmm-admin stop --help``.

.. |pmm-admin-config| replace:: ``pmm-admin config``
.. |pmm-admin-list| replace:: ``pmm-admin list``
.. |pmm-admin-add| replace:: ``pmm-admin add``

