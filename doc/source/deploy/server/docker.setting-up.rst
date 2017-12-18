.. _pmm/server/docker.setting-up:

Setting Up a |docker| Container for |pmm-server|
================================================================================

A |docker| image is a collection of preinstalled software which enables running
a selected version of |pmm-server| on your computer. A |docker| image is not run
directly. You use it to create a |docker| container for your |pmm-server|. When
launched, the |docker| container gives access to the whole functionality of
|pmm|.

The setup begins with pulling the required |docker| image. Then, you proceed by
creating a special container for persistent |pmm| data. The last step is
creating and launching the |pmm-server| container.

Pulling the |pmm-server| Image
--------------------------------------------------------------------------------

To pull the latest version from Docker Hub:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.pull.percona-pmm-server-latest+
   :end-before: #+end-block

This step is not required if you are running |pmm-server| for the first time.
However, it ensures that if there is an older version of the image tagged with
|opt.latest| available locally, it will be replaced by the actual latest
version.

.. _data-container:

Creating the |opt.pmm-data| Container
--------------------------------------------------------------------------------

To create a container for persistent |pmm| data, run the following command:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.create.percona-pmm-server-latest+
   :end-before: #+end-block
	     
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

* ``percona/pmm-server:latest`` is the name and version tag of the image
  to derive the container from.

* ``/bin/true`` is the command that the container runs.

.. important::

   Make sure that the data volumes that you initialize with the |opt.v|
   option match those given in the example. |pmm-server| expects that those
   directories are bind mounted exactly as demonstrated.

.. _server-container:

Creating and Launching the |pmm-server| Container
--------------------------------------------------------------------------------

To create and launch |pmm-server| in one command, use |docker.run|:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.run.latest+
   :end-before: #+end-block
		
This command does the following:

* The |docker.run| command runs a new container based on the
  |opt.pmm-server.latest| image.

* The |opt.d| option starts the container in the background (detached mode).

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

.. _pmm/docker.additional_parameters:

Additional parameters
--------------------------------------------------------------------------------

When running the |pmm-server|, you may pass additional parameters to the
|docker.run| subcommand. All options that appear after the |opt.e| option
are the additional parameters that modify the way how |pmm-server| operates.

.. rubric:: **To enable Orchestrator**

By default, Orchestrator_ is disabled. To enable it, set the
|opt.orchestrator-enabled| option to **true**.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.run.orchestrator-enabled+
   :end-before: #+end-block

.. rubric:: **To disable telemetry**

With :term:`telemetry` enabled, your |pmm-server| sends some statistics to `v.percona.com`_
every 24 hours. This statistics includes the following details:

- |pmm-server| unique ID
- |pmm| version
- The name and version of the operating system
- |mysql| version
- |perl| version

If you do not want your |pmm-server| to send this information, disable telemetry
when running your |docker| container:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.run.disable-telemetry+
   :end-before: #+end-block

.. rubric:: **To disable updates**

To update your |pmm| from web interface you only need to click the
|gui.update| on the home page. The |opt.disable-updates| option is useful
if updating is not desirable. Set it to **true** when running |pmm| in
the |docker| container.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.run.disable-updates+
   :end-before: #+end-block

.. seealso::

   Default ports
      :term:`Ports` in :ref:`pmm/glossary/terminology-reference`
   Updating PMM
     :ref:`Updating PMM <deploy-pmm.updating>`
   Backing Up the |pmm-server| |docker| container
      :ref:`pmm/server/docker/backing-up`
   Restoring |opt.pmm-data|
      :ref:`pmm/server/docker.restoring`


.. include:: ../../.res/replace/name.txt
.. include:: ../../.res/replace/option.txt
.. include:: ../../.res/replace/program.txt
.. include:: ../../.res/replace/url.txt
