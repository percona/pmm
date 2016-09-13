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

Installing PMM Server
=====================

*PMM Server* is the central part of Percona Monitoring and Management.
It combines the backend API and storage for collected data
with a frontend for viewing time-based graphs
and performing thorough analysis of your MySQL and MongoDB hosts
through a web interface.

*PMM Server* is distributed as a Docker image
that is hosted publically at https://hub.docker.com/r/percona/pmm-server/.
The machine where you will be hosting *PMM Server*
must be able to run Docker containers and have network access.

.. note:: Make sure that you are using the latest version of Docker.
   The ones provided via ``apt`` and ``yum``
   may be outdated and cause errors.

.. note:: We encourage to use a specific version tag
   instead of the ``latest`` tag
   when using the ``pmm-server`` image.
   The current stable version is ``1.0.4``.

.. _data-container:

Step 1. Create a PMM Data Container
-----------------------------------

To create a container for persistent PMM data, run the following command:

.. code-block:: bash

   docker create \
      -v /opt/prometheus/data \
      -v /opt/consul-data \
      -v /var/lib/mysql \
      --name pmm-data \
      percona/pmm-server:1.0.4 /bin/true

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

* ``percona/pmm-server:1.0.4`` is the name and version tag of the image
  to derive the container from.

* ``/bin/true`` is the command that the container runs.

.. _server-container:

Step 2. Create and Run the PMM Server Container
-----------------------------------------------

To run *PMM Server*, use the following command:

.. code-block:: bash

   docker run -d \
      -p 80:80 \
      --volumes-from pmm-data \
      --name pmm-server \
      --restart always \
      percona/pmm-server:1.0.4

The previous command does the following:

* The ``docker run`` command instructs the ``docker`` daemon
  to run a container from an image.

* The ``-d`` option starts the container in detached mode
  (that is, in the background).

* The ``-p`` option maps the port for accessing the *PMM Server* web UI.
  For example, if port 80 is not available,
  you can map the landing page to port 8080 using ``-p 8080:80``.

* The ``--volumes-from`` option mounts volumes
  from the ``pmm-data`` container (see :ref:`data-container`).

* The ``--name`` option assigns a custom name for the container
  that you can use to reference the container within a Docker network.
  In this case: ``pmm-server``.

* The ``--restart`` option defines the container's restart policy.
  Setting it to ``always`` ensures that the Docker daemon
  will start the container on startup
  and restart it if the container exits.

* ``percona/pmm-server:1.0.4`` is the name and version tag of the image
  to derive the container from.

Step 3. Verify Installation
---------------------------

When the container starts,
you should be able to access the PMM web interfaces
using the IP address of the host where the container is running.
For example, if it is running on 192.168.100.1 with default port 80,
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

.. _client-install:

Installing PMM Client
=====================

*PMM Client* is a package of agents and exporters
installed on a MySQL or MongoDB host that you want to monitor.
The components collect various data
about general system and database performance,
and send this data to corresponding *PMM Server* components.

Before installing the *PMM Client* package on a database host,
make sure that your *PMM Server* host is accessible.
For example, you can ``ping 192.168.100.1``
or whatever IP address *PMM Server* is running on.

You will need to have root access on the database host
where you will be installing *PMM Client*
(either logged in as a user with root privileges
or be able to run commands with ``sudo``).
*PMM Client* should run on any modern Linux distribution.

The minimum requirements for Query Analytics (QAN) are:

* MySQL 5.1 or later (if using the slow query log)
* MySQL 5.6.9 or later (if using Performance Schema)

.. note:: You should not install agents on database servers
   that have the same host name,
   because host names are used by *PMM Server* to identify collected data.

RPM Packages
------------

1. Download the latest package
   from https://www.percona.com/downloads/pmm-client/LATEST/.
   For example, you can use ``wget`` as follows:

   .. code-block:: bash

      wget https://www.percona.com/downloads/pmm-client/LATEST/pmm-client-1.0.4-1.x86_64.rpm

#. Install the package:

   .. code-block:: bash

      sudo rpm -ivh pmm-client-1.0.4-1.x86_64.rpm

YUM Repository
--------------

1. If your system does not already have Percona's ``yum`` repository configured,
run the following command:

   .. code-block:: bash

      sudo yum install http://www.percona.com/downloads/percona-release/redhat/0.1-3/percona-release-0.1-3.noarch.rpm

#. Install the package:

   .. code-block:: bash

      sudo yum install pmm-client

DEB Packages
------------

1. Download the latest package
   from https://www.percona.com/downloads/pmm-client/LATEST/.
   For example, you can use ``wget`` as follows:

   .. code-block:: bash

      wget https://www.percona.com/downloads/pmm-client/LATEST/pmm-client_1.0.4-1_amd64.deb

#. Install the package:

   .. code-block:: bash

      sudo dpkg -i pmm-client_1.0.4-1_amd64.deb

APT Repository
--------------

1. If your system does not already have Percona's ``apt`` repository configured,
fetch the repository package:

   .. code-block:: bash

      wget https://repo.percona.com/apt/percona-release_0.1-3.$(lsb_release -sc)_all.deb

#. Install the repository package:

   .. code-block:: bash

      sudo dpkg -i percona-release_0.1-3.$(lsb_release -sc)_all.deb

#. Update the local ``apt`` cache:

   .. code-block:: bash

      sudo apt-get update

#. Install the ``pmm-client`` package:

   .. code-block:: bash

      sudo apt-get install pmm-client

Tarball Packages
----------------

1. Download the latest package
   from https://www.percona.com/downloads/pmm-client/LATEST/.
   For example, you can use ``wget`` as follows:

   .. code-block:: bash

      wget https://www.percona.com/downloads/pmm-client/LATEST/pmm-client-1.0.4-x86_64.tar.gz

2. Extract the downloaded tarball:

   .. code-block:: bash

      tar -xzf pmm-client-1.0.4-x86_64.tar.gz

3. Change into the extracted directory and run the install script:

   .. code-block:: bash

      sudo ./install

Connecting to PMM Server
------------------------

To connect the client to PMM Server,
specify the IP address using the ``pmm-admin config --server`` command.
For example, if *PMM Server* is running on ``192.168.100.1``,
and you installed *PMM Client* on a machine with IP ``192.168.200.1``:

   .. code-block:: bash
      :emphasize-lines: 1

      $ sudo pmm-admin config --server 192.168.100.1
      OK, PMM server is alive.

      PMM Server      | 192.168.100.1
      Client Name     | ubuntu-amd64
      Client Address  | 192.168.200.1

.. note:: If you changed the default port 80
   when `creating the PMM Server container <server-container>`_,
   specify it after the server's IP address. For example:

   .. code-block:: bash

      sudo pmm-admin config --server 192.168.100.1:8080

For more information, run ``pmm-admin config --help``

Starting Data Collection
------------------------

To enable data collection, use the ``pmm-admin add`` command.

For general system metrics, MySQL metrics, and query analytics:

.. code-block:: bash

   sudo pmm-admin add mysql

For general system metrics and MongoDB metrics:

.. code-block:: bash

   sudo pmm-admin add mongodb

To see what is being monitored:

.. code-block:: bash

   sudo pmm-admin list

For example, if you enable general OS and MongoDB metrics monitoring,
output should be similar to the following:

.. code-block:: bash
   :emphasize-lines: 1

   $ sudo pmm-admin list
   pmm-admin 1.0.4

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

For more information about managing *PMM Client* with the ``pmm-admin`` tool,
see :ref:`pmm-admin`.

.. _remove-server:

Removing PMM Server
===================

1. Stop and remove the ``pmm-server`` container:

   .. code-block:: bash

      docker stop pmm-server && docker rm pmm-server

2. If you also want to discard all collected data,
   remove the ``pmm-data`` container:

   .. code-block:: bash

      docker rm pmm-data

.. note:: Before removing the data container,
   you should remove all instances on all *PMM Clients*
   using :ref:`pmm-admin rm <pmm-admin-rm>`.

.. _upgrade-server:

Upgrading PMM Server
====================

When a newer version of *PMM Server* image becomes available:

1. Stop and remove the ``pmm-server`` container:

   .. code-block:: bash

      docker stop pmm-server && docker rm pmm-server

2. Create and run from the image with the new version tag,
   as described in :ref:`server-container`.

.. warning:: Do not remove the ``pmm-data`` container when upgrading,
   if you want to keep all collected data.

.. _remove-client:

Removing PMM Client
===================

1. Remove all monitored instances as described in :ref:`pmm-admin-rm`.

2. Change into the directory with the extracted *PMM Client* tarball
   and run:

   .. code-block:: bash

      sudo ./uninstall

.. _upgrade-client:

Upgrading PMM Client
====================

When a newer version of *PMM Client* becomes available:

1. :ref:`Remove PMM Client <remove-client>`.

2. Download and install the *PMM Client* package
   as described :ref:`here <client-install>`.

.. rubric:: References

.. target-notes::

