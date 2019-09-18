.. _pmm.server.docker-setting-up:

================================================================================
Setting Up a |docker| Container for |pmm-server|
================================================================================

.. contents::
   :local:

A |docker| image is a collection of preinstalled software which enables running
a selected version of |pmm-server| on your computer. A |docker| image is not run
directly. You use it to create a |docker| container for your |pmm-server|. When
launched, the |docker| container gives access to the whole functionality of
|pmm|.

The setup begins with pulling the required |docker| image. Then, you proceed by
creating a special container for persistent |pmm| data. The last step is
creating and launching the |pmm-server| container.

.. _pmm.server.docker-image.pulling:

`Pulling the PMM Server Docker Image <docker-setting-up.html#image-pulling>`_
--------------------------------------------------------------------------------

To pull the latest version from Docker Hub:

.. include:: ../.res/code/docker.pull.percona-pmm-server-latest.txt

This step is not required if you are running |pmm-server| for the first time.
However, it ensures that if there is an older version of the image tagged with
``{{release}}`` available locally, it will be replaced by the actual latest
version.

.. _data-container:

`Creating the pmm-data Container <docker-setting-up.html#data-container>`_
--------------------------------------------------------------------------------

To create a container for persistent |pmm| data, run the following command:

.. code-block:: bash

   $ docker create \
      -v /srv \
      --name pmm-data \
      percona/pmm-server:2 /bin/true
	     
.. note:: This container does not run, it simply exists to make sure you retain
	  all |pmm| data when you upgrade to a newer |pmm-server| image.  Do not remove
	  or re-create this container, unless you intend to wipe out all |pmm| data and
	  start over.

The previous command does the following:

* The |docker.create| command instructs the |docker| daemon
  to create a container from an image.

* The |opt.v| options initialize data volumes for the container.

* The |opt.name| option assigns a custom name for the container
  that you can use to reference the container within a |docker| network.
  In this case: ``pmm-data``.

* ``percona/pmm-server:2`` is the name and version tag of the image
  to derive the container from.

* ``/bin/true`` is the command that the container runs.

.. important::

   Make sure that the data volumes that you initialize with the |opt.v|
   option match those given in the example. |pmm-server| expects that those
   directories are bind mounted exactly as demonstrated.

.. _server-container:

`Creating and Launching the PMM Server Container <docker-setting-up.html#server-container>`_
---------------------------------------------------------------------------------------------

To create and launch |pmm-server| in one command, use |docker.run|:

.. include:: ../.res/code/docker.run.latest.txt

This command does the following:

* The |docker.run| command runs a new container based on the
  |opt.pmm-server.latest| image.

* The |opt.p| option maps the port for accessing the |pmm-server| web UI.
  For example, if port **80** is not available,
  you can map the landing page to port 8080 using ``-p 8080:80``.

* The |opt.v| option mounts volumes
  from the ``pmm-data`` container (see :ref:`data-container`).

* The |opt.name| option assigns a custom name to the container
  that you can use to reference the container within the |docker| network.
  In this case: ``pmm-server``.

* The |opt.restart| option defines the container's restart policy.
  Setting it to ``always`` ensures that the Docker daemon
  will start the container on startup
  and restart it if the container exits.

* |opt.pmm-server.latest| is the name and version tag of the image
  to derive the container from.

.. _pmm.docker.specific-version:

`Installing and using specific docker version <docker-setting-up.html#specific-version>`_
-----------------------------------------------------------------------------------------

To install specific |pmm-server| version instead of the latest one, just put
desired version number after the colon. Also in this scenario it may be useful
to `prevent updating PMM Server via the web interface <https://www.percona.com/doc/percona-monitoring-and-management/glossary.option.html>`_ with the ``DISABLE_UPDATES`` docker option.

For example, installing version 2.0 with disabled update button in the web
interface would look as follows:

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

.. only:: showhidden

	.. _pmm.docker.additional-option:

	`Additional options <docker-setting-up.html#additional-option>`_
	--------------------------------------------------------------------------------

	When running the |pmm-server|, you may pass additional parameters to the
	|docker.run| subcommand. All options that appear after the |opt.e| option
	are the additional parameters that modify the way how |pmm-server| operates.

	The section :ref:`pmm.glossary.pmm-server.additional-option` lists all
	supported additional options.

.. seealso::

   Updating PMM
      :ref:`Updating PMM <deploy-pmm.updating>`
   Backing Up the |pmm-server| |docker| container
      :ref:`pmm.server.docker-backing-up`
   Restoring |opt.pmm-data|
      :ref:`pmm.server.docker-restoring`

.. include:: ../.res/replace.txt
