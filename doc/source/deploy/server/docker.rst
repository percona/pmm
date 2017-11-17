.. _run-server-docker:

================================================================================
Running |pmm-server| Using |docker|
================================================================================

|docker| images of |pmm-server| are at the `percona/pmm-server`_ public
repository. If you intend to run |pmm-server| from a |docker| image, the host
must be able to run |docker| 1.12.6 or later, and have network access.


For more information about using |docker|, see the `Docker Docs`_.

.. This note is not necessary because the docker docs provide much better
   instructions about using package managers.  Also, this note is not
   accurate. Using apt and yum is ok: docker provides their repositories.

   .. note:: Make sure that you are using the latest version of |docker|.
      The ones provided via ``apt`` or ``yum``
      may be outdated and cause errors.

.. note:: Make sure that the firewall and routing rules of the host
   do not constrain the |docker| container.
   For more information, see :ref:`troubleshoot-connection`.

Step 1. Pull the PMM Server Image
================================================================================

To pull the latest version from Docker Hub:

.. code-block:: bash

   $ docker pull percona/pmm-server:latest

This is not required if you are running |pmm-server| for the first time.
However, it ensures that if there is an older version of the image
tagged with ``latest`` available locally,
it will be replaced by the actual latest version.

.. _data-container:

Step 2. Create a PMM Data Container
================================================================================

To create a container for persistent PMM data, run the following command:

.. code-block:: bash

   $ docker create \
      -v /opt/prometheus/data \
      -v /opt/consul-data \
      -v /var/lib/mysql \
      -v /var/lib/grafana \
      --name pmm-data \
      percona/pmm-server:latest /bin/true

.. note:: This container does not run,
   it simply exists to make sure you retain all |pmm| data
   when you upgrade to a newer |pmm-server| image.
   Do not remove or re-create this container,
   unless you intend to wipe out all PMM data and start over.

The previous command does the following:

* The ``docker create`` command instructs the Docker daemon
  to create a container from an image.

* The ``-v`` options initialize data volumes for the container.

* The ``--name`` option assigns a custom name for the container
  that you can use to reference the container within a Docker network.
  In this case: ``pmm-data``.

* ``percona/pmm-server:latest`` is the name and version tag of the image
  to derive the container from.

* ``/bin/true`` is the command that the container runs.

.. _server-container:

Step 3. Create and Run the |pmm-server| Container
================================================================================

To run |pmm-server|, use the following command:

.. include:: ../../.resources/code/sh.txt
   :start-after: docker.run.latest
   :end-before: (end-code-block)
		
The previous command does the following:

* The ``docker run`` command instructs the ``docker`` daemon
  to run a container from an image.

* The ``-d`` option starts the container in detached mode
  (that is, in the background).

* The ``-p`` option maps the port for accessing the |pmm-server| web UI.
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

* ``percona/pmm-server:latest`` is the name and version tag of the image
  to derive the container from.

.. _pmm/docker.additional_parameters:

Additional parameters
--------------------------------------------------------------------------------

When running the |pmm-server|, you may pass additional parameters to
the |docker| *run* subcommand.

.. rubric:: To enable Orchestrator

By default, Orchestrator_ is disabled. To enable it, set the
 |opt.orchestrator-enabled| option to **true**.

.. include:: ../../.resources/code/sh.txt
   :start-after: docker.run.orchestrator-enabled
   :end-before: (end-code-block)

.. rubric:: To disable telemetry

With :term:`telemetry` enabled, your |pmm-server| sends some statistics to `v.percona.com`_
every 24 hours. This statistics includes the following details:

- |pmm-server| unique ID
- |pmm| version
- The name and version of the operating system
- |mysql| version
- |perl| version

If you do not want your |pmm-server| to send this information, disable telemetry
when running your |docker| container:

.. include:: ../../.resources/code/sh.txt
   :start-after: docker.run.disable-telemetry
   :end-before: (end-code-block)

.. rubric:: To disable updates

To update your |pmm| from web interface you only need to click the
:guilabel:`Update` on the home page. The |opt.disable-updates| option is useful
if updating is not desirable. Set it to **true** when running |pmm| in
the |docker| container.

.. include: ../../.resources/code/sh.txt
   :start-after: docker.run.disable-updates
   :end-before: (end-code-block)

.. toctree::
   :hidden:

   upgrade

.. seealso::

   - :ref:`Updating PMM <deploy-pmm.updating>`

.. include:: ../../.resources/url.txt
.. include:: ../../.resources/name.txt
