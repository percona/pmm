.. _install:

=========================================================
Installing the Percona Monitoring and Management Platform
=========================================================

Percona Monitoring and Management (PMM) is distributed
as a self-contained boxed solution, separated into two distinct modules:

* *PMM Server* is distributed as a Docker image
  that you can use to run a container with all the necessary components.
  For more information about Docker,
  see the `Docker Docs`_.

.. _`Docker Docs`: https://docs.docker.com/

* *PMM Client* is distributed as a tarball
  that you extract and run an install script.

For more information about the functions
and internal structure of each module, see :ref:`architecture`.

.. note:: PMM is currently in beta.

   Test it out on non-production machines
   before using it in your production environment.

Installing PMM Server
=====================

*PMM Server* is the central part of Percona Monitoring and Management.
It combines the backend API and storage for collected data
with a frontend for viewing time-based graphs
and performing thorough analysis of your MySQL and MongoDB hosts
through a web interface.

The machine where you will be hosting *PMM Server*
must be able to run Docker containers and have network access.

.. note:: Make sure that you are using the latest version of Docker.
   The ones provided via ``apt`` and ``yum``
   may be outdated and cause errors.

.. _ports:

Although you can map different ports for the components in *PMM Server*,
it is recommended that you use the following default ports:

===== ============================================
Port  Used by
===== ============================================
80    PMM landing page (configurable)
9001  QAN API
===== ============================================

.. note:: If you change the default port 80,
   you will have to specify it every time after the IP address.

*PMM Server* is distributed as a Docker image
that is hosted publically at https://hub.docker.com/r/percona/pmm-server/.

.. _version-tag:

Step 1. Check the correct version tag
-------------------------------------

You can find all available version tags listed at
https://hub.docker.com/r/percona/pmm-server/tags/.
The ``latest`` tag is an alias
that points to the latest uploaded version of the image.
If you run the *PMM Server* container with the ``latest`` tag,
keep the following in mind:

* It will be latest only at the time when you initially pull the image.
  Once a newer version is uploaded to Docker Hub,
  ``latest`` will change to point to that,
  but your local version will not update.
  To update, you will need to stop the container, remove the image,
  and then pull and run with the ``latest`` tag again.
  If you use a tag with a specific version,
  you can simply stop the container and then run with the newer tag
  (removing the old image in this case is optional).

* The ``latest`` tag may point to an experimental version of the image,
  which is not the latest recommended stable version.
  Always read the description
  to know which version is the current stable version.

.. _data-container:

Step 2. Create a PMM Data Container
-----------------------------------

To create a container for persistent PMM data, run the following command:

.. prompt:: bash

   docker create \
      -v /opt/prometheus/data \
      -v /opt/consul-data \
      -v /var/lib/mysql \
      --name pmm-data \
      percona/pmm-server:<VERSION_TAG> /bin/true

.. note:: This container does not run,
   it simply exists to make sure you retain all PMM data
   when you upgrade to a newer ``pmm-server`` image.
   Do not remove or re-create this container,
   unless you intend to wipe out all PMM data and start over.

The previous command does the following:

* The ``docker create`` command instructs the Docker daemon
  to create a container from an image.

* The ``-v`` options initialize data volumes for the container.

* The ``--name`` option assigns a custom name for the container
  that you can use to reference the container within a Docker network.
  In this case: ``pmm-data``.

* ``percona/pmm-server`` is the name of the image
  to derive the container from.

* ``<VERSION_TAG>`` should be replaced with the full version number
  of the image you want to use.
  For more information, see :ref:`version-tag`.

* ``/bin/true`` is the command that the container runs.

.. _server-container:

Step 3. Create and Run the PMM Server Container
-----------------------------------------------

To run *PMM Server*, use the following command:

.. prompt:: bash

   docker run -d \
      -p 80:80 -p 9001:9001 \
      -e ADDRESS=<SERVER_ADDR> \
      --volumes-from pmm-data \
      --name pmm-server \
      percona/pmm-server:<VERSION_TAG>

The previous command does the following:

* The ``docker run`` command instructs the ``docker`` daemon
  to run a container from an image.

* The ``-d`` option starts the container in detached mode
  (that is, in the background).

* The ``-p`` options map ports used by *PMM Server*.
  For example, if port 80 is not available,
  you can map the landing page to port 8080 using ``-p 8080:80``.
  For more information about default ports used by *PMM Server*,
  see :ref:`this table <ports>`.

* The ``-e`` option sets the ``ADDRESS`` environment variable
  to the IP address of the host where you are running the container
  (for example, ``-e ADDRESS=192.168.100.1``).
  This is necessary for QAN API to report itself on that address
  instead of the container's private IP address.

* The ``--volumes-from`` option mounts volumes
  from the ``pmm-data`` container.

* The ``--name`` option assigns a custom name for the container
  that you can use to reference the container within a Docker network.
  In this case: ``pmm-server``.

* ``percona/pmm-server`` is the name of the image
  to derive the container from.

* ``<VERSION_TAG>`` should be replaced with the full version number
  of the image you want to use.
  For more information, see :ref:`version-tag`.

Step 4. Verify Installation
---------------------------

When the container starts,
you should be able to access the PMM web interfaces
using the IP address of the host where the container is running.
For example, if it is running on 192.168.100.1 with default ports,
you should be able to access the following:

==================================== ================================
Component                            URL
==================================== ================================
PMM landing page                     http://192.168.100.1
Query Analytics (QAN web app)        http://192.168.100.1/qan/
Metrics Monitor (Grafana)            | http://192.168.100.1/graph/
                                     | user name: ``admin``
                                     | password: ``admin``
==================================== ================================

Installing PMM Client
=====================

*PMM Client* is a package of agents and exporters
installed on a MySQL or MongoDB host that you want to monitor.
The components collect various data
about general system and database performance,
and send this data to corresponding *PMM Server* components.

Before installing the *PMM Client* package on a database host,
make sure that your *PMM Server* host is accessible.
You will need to have root access on the database host
where you will be installing *PMM Client*
(either logged in as a user with root privileges
or be able to run commands with ``sudo``).
*PMM Client* should run on any modern Linux distribution.

Query Analytics (QAN) requires:

* MySQL 5.1 or later (if using the slow query log)
* MySQL 5.6.9 or later (if using Performance Schema)

.. _client-install:

**To install PMM Client:**

1. Download the latest package
   from https://www.percona.com/redir/downloads/TESTING/pmm/.
   For example, you can use ``wget`` as follows:

   .. prompt:: bash

      wget https://www.percona.com/redir/downloads/TESTING/pmm/pmm-client.tar.gz

2. Extract the downloaded tarball:

   .. prompt:: bash

      tar -xzf pmm-client.tar.gz

3. Change into the extracted directory and run the install script.
   Specify the IP address of the *PMM Server* host as the argument.
   For example:

   .. prompt:: bash

      sudo ./install 192.168.100.1

   .. note:: If you changed the default port 80
      when `creating the PMM Server container <server-container>`_,
      specify it after the IP address. For example:

      .. prompt:: bash

         sudo ./install 192.168.100.1:8080

Starting Data Collection
------------------------

After you install *PMM Client*,
enable data collection using the ``pmm-admin`` tool:

To enable general system metrics monitoring,
run ``pmm-admin add os`` followed by the IP address
of the *PMM Client* host. For example:

.. prompt:: bash

   sudo pmm-admin add os 192.168.100.2

MySQL Data
**********

To enable MySQL metrics monitoring and query analytics,
run ``pmm-admin add mysql``.

.. note:: Query analytics must be able to detect
   the local MySQL instance and MySQL superuser credentials.
   Make sure that the necessary options are specified
   in :file:`~/.my.cnf`. For example:

   .. code-block:: none

      user=root
      password=pass
      socket=/var/run/mysqld/mysqld.sock

   Alternatively, you can specify MySQL superuser credentials
   as command-line options for the ``pmm-admin`` tool:

   .. prompt:: bash

      pmm-admin -user root -password pass add mysql

For a complete list of command-line options, run ``pmm-admin -help``.

MongoDB Data
************

To enable MongoDB metrics monitoring, run ``pmm-admin add mongodb``.

You can use options to specify the MongoDB replica set, cluster name,
and node type. For example:

.. prompt:: bash

   pmm-admin -mongodb-replset repl1 -mongodb-cluster cluster1 -mongodb-nodetype mongod add mongodb

Verifying
*********

To see what is being monitored, run ``pmm-admin list``.
If everything is enabled, output should be similar to the following:

.. code-block:: bash

   $ pmm-admin list
         TYPE NAME                                            OPTIONS
   ---------- ----------------------------------------------- -------
        mysql ubuntu-amd64
           os ubuntu-amd64

Removing PMM Server
===================

1. Stop and remove the ``pmm-server`` container:

   .. prompt:: bash

      docker stop pmm-server && docker rm pmm-server

2. If you also want to remove all collected data,
   remove the ``pmm-data`` container:

   .. prompt:: bash

      docker rm pmm-data

Upgrading PMM Server
====================

When a newer version of *PMM Server* image becomes available:

1. Stop and remove the ``pmm-server`` container:

   .. prompt:: bash

      docker stop pmm-server && docker rm pmm-server

2. Create and run from the image with the new version tag,
   as described in :ref:`server-container`.

.. warning:: Do not remove the ``pmm-data`` container when upgrading,
   if you want to keep all collected data.

.. _remove-client:

Removing PMM Client
===================

1. Stop the *PMM Client* services:

   .. prompt:: bash

      sudo /etc/init.d/percona-qan-agent stop && /etc/init.d/percona-prom-pm stop

2. Clear out the *PMM Client* installation directory, binaries, and services:

   .. prompt:: bash

      rm -rf /usr/local/percona /usr/local/bin/pmm-admin /etc/init.d/percona-prom-pm /etc/init.d/percona-qan-agent

.. _upgrade-client:

Upgrading PMM Client
====================

When a newer version of *PMM Client* becomes available:

1. :ref:`Remove PMM Client <remove-client>`.

2. Download and install the *PMM Client* package
   as described :ref:`here <client-install>`.

.. rubric:: References

.. target-notes::

