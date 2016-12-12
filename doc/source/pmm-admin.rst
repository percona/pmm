.. _pmm-admin:

===================
Managing PMM Client
===================

Use the ``pmm-admin`` tool to manage *PMM Client*.

.. note:: The ``pmm-admin`` tool requires root access
   (you should either be logged in as a user with root privileges
   or be able to run commands with ``sudo``).

Use the ``--help`` option to view the built-in help.
For example, you can view all available commands and options
by running the following:

.. code-block:: bash

   sudo pmm-admin --help

.. contents::
   :local:
   :depth: 1

.. _pmm-admin-add:

Adding monitoring services
==========================

Use the ``pmm-admin add`` command to add monitoring services.

For complete MySQL instance monitoring:

.. code-block:: bash

   sudo pmm-admin add mysql

The previous command adds the following services:

* ``linux:metrics``
* ``mysql:metrics``
* ``mysql:queries``

For complete MongoDB instance monitoring:

.. code-block:: bash

   sudo pmm-admin add mongodb

The previous command adds the following services:

* ``linux:metrics``
* ``mongodb:metrics``

.. _pmm-admin-add-linux-metrics:

linux:metrics
-------------

**To enable general system metrics monitoring:**

.. code-block:: bash

   sudo pmm-admin add linux:metrics

This creates the ``pmm-linux-metrics-42000`` service
that collects local system metrics for this particular OS instance.

.. note:: It should be able to detect the local PMM Client name,
   but you can also specify it explicitely as an argument.

For more information, run ``sudo pmm-admin add linux:metrics --help``

.. _pmm-admin-add-mysql-queries:

mysql:queries
-------------

**To enable MySQL query analytics:**

.. code-block:: bash

   sudo pmm-admin add mysql:queries

This creates the ``pmm-mysql-queries-0`` service
that is able to collect QAN data for multiple remote MySQL server instances.

The ``pmm-admin`` tool will attempt to automatically detect
the local MySQL instance and MySQL superuser credentials.
You can use options to provide this information for ``pmm-admin``
if it is not able to auto-detect.
You can also specify the ``--create-user`` option to create a dedicated
``pmm`` user on the MySQL host that you want to monitor.
This user will be given all the necessary privileges for monitoring,
and is recommended over using the MySQL superuser.

For example, to set up remote monitoring of QAN data
on a MySQL server located at 192.168.200.2,
use a command similar to the following:

.. code-block:: bash

   sudo pmm-admin add mysql:queries --user root --password root --host 192.168.200.2 --create-user

QAN can use either the slow query log or Performance Schema as the source.
By default, it chooses the slow query log for a local MySQL instance
and Performance Schema otherwise.
For more information about the differences, see :ref:`perf-schema`.

You can explicitely set the query source when adding a QAN instance
using the ``--query-source`` option.

For more information, run ``sudo pmm-admin add mysql:queries --help``

.. _pmm-admin-add-mysql-metrics:

mysql:metrics
-------------

**To enable MySQL metrics monitoring:**

.. code-block:: bash

   sudo pmm-admin add mysql:metrics

This creates the ``pmm-mysql-metrics-42002`` service
that collects MySQL instance metrics.

The ``pmm-admin`` tool will attempt to automatically detect
the local MySQL instance and MySQL superuser credentials.
You can use options to provide this information for ``pmm-admin``
if it is not able to auto-detect.
You can also specify the ``--create-user`` option to create a dedicated
``pmm`` user on the MySQL host that you want to monitor.
This user will be given all the necessary privileges for monitoring,
and is recommended over using the MySQL superuser.

For example,
to set up remote monitoring of MySQL metrics
on a server located at 192.168.200.3,
use a command similar to the following:

.. code-block:: bash

   sudo pmm-admin add mysql:metrics --user root --password root --host 192.168.200.3 --create-user

For more information, run ``sudo pmm-admin add mysql:metrics --help``.

.. _pmm-admin-add-mongodb-metrics:

mongodb:metrics
---------------

**To enable MongoDB metrics monitoring:**

.. code-block:: bash

   sudo pmm-admin add mongodb:metrics

This creates the ``pmm-mongodb-metrics-42003`` service
that collects local MongoDB metrics for this particular MongoDB instance.

.. note:: It should be able to detect the local PMM Client name,
   but you can also specify it explicitely as an argument.

You can use options to specify the MongoDB replica set, cluster name,
and node type. For example:

.. code-block:: bash

   sudo pmm-admin add mongodb --replset repl1 --cluster cluster1 --nodetype mongod

For more information, run ``sudo pmm-admin add mongodb:metrics --help``

.. _pmm-admin-add-proxysql-metrics:

proxysql:metrics
----------------

**To enable ProxySQL performance metrics monitoring:**

.. code-block:: bash

   sudo pmm-admin add proxysql:metrics

This creates the ``pmm-proxysql-metrics-42004`` service
that collects local ProxySQL performance metrics.

.. note:: It should be able to detect the local PMM Client name,
   but you can also specify it explicitely as an argument.

For more information, run ``sudo pmm-admin add proxysql:metrics --help``

.. _pmm-admin-rm:

Removing monitoring services
============================

Use the ``pmm-admin rm`` command to remove monitoring services.
Specify the instance's type and name.
You can see the names of instances by running ``sudo pmm-admin list``.

For example, to remove a MySQL instance designated by ``ubuntu-amd4``
from monitoring, run the following:

.. code-block:: bash

   sudo pmm-admin rm mysql ubuntu-amd64

For more information, run ``sudo pmm-admin rm --help``.

.. _pmm-admin-list:

Listing monitored instances
===========================

To see what is being monitored, run the following:

.. code-block:: bash

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

   $ sudo pmm-admin list
   pmm-admin 1.0.5

   PMM Server      | 192.168.100.1
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.200.1
   Service manager | linux-systemd

   --------------- ------------- ------------ -------- ---------------- --------
   METRIC SERVICE  NAME          CLIENT PORT  RUNNING  DATA SOURCE      OPTIONS
   --------------- ------------- ------------ -------- ---------------- --------
   linux:metrics   ubuntu-amd64  42000        YES      -
   mongodb:metrics ubuntu-amd64  42003        YES      localhost:27017

.. _pmm-admin-config:

Configuring PMM Client
======================

Use the ``pmm-admin config`` command to configure
how ``pmm-admin`` communicates with *PMM Server*.

The following options are available:

--client-address string   Client host address (detected automatically)
--client-name string      Client host name (set to the current host name)
--server string           PMM Server host address
--server-insecure-ssl     Enable insecure SSL (self-signed certificate)
--server-password string  HTTP password configured on PMM Server
--server-ssl              Enable SSL to communicate with PMM Server
--server-user string      HTTP user configured on PMM Server (default "pmm")

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

   $ sudo pmm-admin info
   pmm-admin 1.0.5

   PMM Server      | 192.168.100.6
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.200.1
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

   $ sudo pmm-admin check-network --no-emoji
   PMM Network Status

   Server | 192.168.100.1
   Client | 192.168.200.1

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
   ---------------- ------------- ---------------------- -------------
   METRIC SERVICE   NAME          PROMETHEUS ENDPOINT    REMOTE STATE
   ---------------- ------------- ---------------------- -------------
   linux:metrics    ubuntu-amd64  192.168.200.1:42000    OK
   mysql:metrics    ubuntu-amd64  192.168.200.1:42002    OK
   mongodb:metrics  ubuntu-amd64  192.168.200.1:42003    PROBLEM

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

For example, to start the ``mongodb:metrics`` service on host ``ubuntu-amd64``:

.. code-block:: bash

   sudo pmm-admin start mongodb:metrics ubuntu-amd64

To stop the ``linux:metrics`` service on host ``centos-amd64``:

.. code-block:: bash

   sudo pmm-admin stop linux:metrics centos-amd64

To stop all services managed by this ``pmm-admin``:

.. code-block:: bash

   sudo pmm-admin stop --all

For more information,
run ``sudo pmm-admin start --help`` or ``sudo pmm-admin stop --help``.

.. |pmm-admin-config| replace:: ``pmm-admin config``
.. |pmm-admin-list| replace:: ``pmm-admin list``
.. |pmm-admin-add| replace:: ``pmm-admin add``

