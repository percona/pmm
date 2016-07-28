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

*PMM Server* is distributed as a Docker image
that is hosted publically at https://hub.docker.com/r/percona/pmm-server/.
The machine where you will be hosting *PMM Server*
must be able to run Docker containers and have network access.

.. note:: Make sure that you are using the latest version of Docker.
   The ones provided via ``apt`` and ``yum``
   may be outdated and cause errors.

.. note:: We encourage to use a specific version tag
   instead of the ``latest`` tag
   when using the ``pmm-server`` image,
   The current stable version is ``1.0.2``.

.. _data-container:

Step 1. Create a PMM Data Container
-----------------------------------

To create a container for persistent PMM data, run the following command:

.. prompt:: bash

   docker create \
      -v /opt/prometheus/data \
      -v /opt/consul-data \
      -v /var/lib/mysql \
      --name pmm-data \
      percona/pmm-server:1.0.2 /bin/true

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

* ``percona/pmm-server:1.0.2`` is the name and version tag of the image
  to derive the container from.

* ``/bin/true`` is the command that the container runs.

.. _server-container:

Step 2. Create and Run the PMM Server Container
-----------------------------------------------

To run *PMM Server*, use the following command:

.. prompt:: bash

   docker run -d \
      -p 80:80 \
      --volumes-from pmm-data \
      --name pmm-server \
      --restart always \
      percona/pmm-server:1.0.2

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

* ``percona/pmm-server:1.0.2`` is the name and version tag of the image
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
   Specify the IP address of the *PMM Server* host
   followed by the client's IP address as the arguments.

   .. code-block:: none

      sudo ./install <PMM server address[:port]> <client address>

   For example, if *PMM Server* is running on ``192.168.100.1``
   and you are installing *PMM Client* on a machine with IP ``192.168.200.1``:

   .. prompt:: bash

      sudo ./install 192.168.100.1 192.168.200.1

   .. note:: If you changed the default port 80
      when `creating the PMM Server container <server-container>`_,
      specify it after the server's IP address. For example:

      .. prompt:: bash

         sudo ./install 192.168.100.1:8080 192.168.200.1

Starting Data Collection
------------------------

After you install *PMM Client*,
enable data collection using the ``pmm-admin`` tool.

To enable general system metrics monitoring:

.. prompt:: bash

   sudo pmm-admin add os

To enable MySQL query analytics:

.. prompt:: bash

   sudo pmm-admin add queries

To enable MySQL metrics monitoring:

.. prompt:: bash

   sudo pmm-admin add mysql

To enable MongoDB metrics monitoring:

.. prompt:: bash

   sudo pmm-admin add mongodb

To see what is being monitored:

.. prompt:: bash

   sudo pmm-admin list

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

The ``pmm-admin`` tool has built-in help that can be viewed
using the ``--help`` option.
For more information about managing *PMM Client* with the ``pmm-admin`` tool,
see :ref:`pmm-admin`.

.. _remove-server:

Removing PMM Server
===================

1. Stop and remove the ``pmm-server`` container:

   .. prompt:: bash

      docker stop pmm-server && docker rm pmm-server

2. If you also want to discard all collected data,
   remove the ``pmm-data`` container:

   .. prompt:: bash

      docker rm pmm-data

.. note:: Before removing the data container,
   you should remove all instances on all *PMM Clients*
   using :ref:`pmm-admin rm <pmm-admin-rm>`.

.. _upgrade-server:

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

1. Remove all monitored instances as described in :ref:`pmm-admin-rm`.

2. Change into the directory with the extracted *PMM Client* tarball
   and run:

   .. prompt:: bash

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

