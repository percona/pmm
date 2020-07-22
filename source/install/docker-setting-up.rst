.. _pmm.server.docker-setting-up:

================================================================================
Setting Up a Docker Container for PMM Server
================================================================================

A Docker image is a collection of preinstalled software which enables running
a selected version of PMM Server on your computer. A Docker image is not
run directly. You use it to create a Docker container for your PMM Server.
When launched, the Docker container gives access to the whole functionality
of PMM.

The setup begins with pulling the required Docker image. Then, you proceed by
creating a special container for persistent PMM data. The last step is
creating and launching the PMM Server container.

.. _pmm.server.docker-image.pulling:

`Pulling the PMM Server Docker Image <docker-setting-up.html#pmm-server-docker-image-pulling>`_
-----------------------------------------------------------------------------------------------

To pull the latest version from Docker Hub:

.. include:: ../.res/code/docker.pull.percona-pmm-server-latest.txt

This step is not required if you are running PMM Server for the first time.
However, it ensures that if there is an older version of the image tagged with
|release-code| available locally, it will be replaced by the actual latest
version.

.. _data-container:

`Creating the pmm-data Container <docker-setting-up.html#data-container>`_
--------------------------------------------------------------------------------

To create a container for persistent PMM data, run the following command:

.. code-block:: bash

   $ docker create \
      -v /srv \
      --name pmm-data \
      percona/pmm-server:2 /bin/true

.. note:: This container does not run, it simply exists to make sure you retain
      all PMM data when you upgrade to a newer PMM Server image.  Do not remove
      or re-create this container, unless you intend to wipe out all PMM data and
      start over.

The previous command does the following:

* The ``docker create`` command instructs the Docker daemon
  to create a container from an image.

* The ``-v`` options initialize data volumes for the container.

* The ``--name`` option assigns a custom name for the container
  that you can use to reference the container within a Docker network.
  In this case: ``pmm-data``.

* ``percona/pmm-server:2`` is the name and version tag of the image
  to derive the container from.

* ``/bin/true`` is the command that the container runs.

.. important::

   PMM Server expects that the data volume initialized with the ``-v`` option will be
   ``/srv``.  Using any other value will result in data loss in an upgrade.

.. _server-container:

`Creating and Launching the PMM Server Container <docker-setting-up.html#server-container>`_
---------------------------------------------------------------------------------------------

To create and launch PMM Server in one command, use ``docker run``:

.. include:: ../.res/code/docker.run.latest.txt

This command does the following:

* The ``docker run`` command runs a new container based on the
  ``percona/pmm-server:2`` image.

* The ``-p`` option maps the host port to the server port inside the docker
  container for accessing the PMM Server web UI in the format of
  ``-p <hostPort>:<containerPort>``. For example, if port **80** is not
  available on your host, you can map the landing page to port 8080 using
  ``-p 8080:80``, the same for port **443**: ``-p 8443:443``.

* The ``--volumes-from`` option mounts volumes from the ``pmm-data`` container
  created previously (see :ref:`data-container`).

* The ``--name`` option assigns a custom name to the container
  that you can use to reference the container within the Docker network.
  In this case: ``pmm-server``.

* The ``--restart`` option defines the container's restart policy.
  Setting it to ``always`` ensures that the Docker daemon
  will start the container on startup
  and restart it if the container exits.

* ``percona/pmm-server:2`` is the name and version tag of the image
  to derive the container from.

* A warning message is printed if invalid an environment variable name key is passed in via the command line option ``-e <KEY>=<VALUE>``.

.. _pmm.docker.specific-version:

`Installing and using specific PMM Server version <docker-setting-up.html#pmm-docker-specific-version>`_
----------------------------------------------------------------------------------------------------------

To install a specific PMM Server version instead of the latest one, just put
desired version number after the colon. Also in this scenario it may be useful
to `prevent updating PMM Server via the web interface <https://www.percona.com/doc/percona-monitoring-and-management/glossary.option.html>`_ with the ``DISABLE_UPDATES`` docker option.

Following docker tags are currently available to represent PMM Server versions:

* ``:latest`` currently means the latest release of the PMM 1.X

* ``:2`` is the latest released version of PMM 2

* ``:2.X`` can be used to refer any minor released version, excluding patch
  releases

* ``:2.X.Y`` tag means specific patch release of PMM


For example, installing the latest 2.x version with disabled update button in
the web interface would look as follows:

.. code-block:: bash

   $ docker create \
      -v /srv \
      --name pmm-data \
      percona/pmm-server:2 /bin/true

   $ docker run -d \
      -p 80:80 \
      -p 443:443 \
      --volumes-from pmm-data \
      --name pmm-server \
      -e DISABLE_UPDATES=true \
      --restart always \
      percona/pmm-server:2

.. seealso::

   Updating PMM
      :ref:`Updating PMM<update-server.docker>`
   Backing Up the PMM Server Docker container
      :ref:`pmm.server.docker-backing-up`
   Restoring ``pmm-data``
      :ref:`pmm.server.docker-restoring`
