:orphan:



.. _install-client-docker:

######################################################################
Installing PMM Client via Docker
######################################################################

Docker images of PMM Client are stored at the `percona/pmm-client`_
public repository. The host must be able to run Docker 1.12.6 or later,
and have network access.

Make sure that the firewall and routing rules of the host do not constrain
the Docker container. For more information, see :ref:`troubleshoot-connection`.

For more information about using Docker, see the `Docker Docs`_.

**********************************************************************
Setting Up a Docker Container for PMM Client
**********************************************************************

A Docker image is a collection of preinstalled software which lets you
run a selected version of PMM Client.
A Docker image is not run directly.
You use it to create a Docker container for your PMM Client.
When launched, the Docker container gives access to the whole functionality
of PMM Client.

- The setup begins by pulling the required Docker image.
- Next, you create a special container for persistent PMM data.
- Finally, you create and launch the PMM Client container.

======================================================================
Pulling the PMM Client Docker Image
======================================================================

To pull the latest version from Docker Hub:

.. code-block:: bash

   docker pull percona/pmm-client:2

======================================================================
Creating a Persistent Data Store for the PMM Client Docker Container
======================================================================

To create a container for persistent data, run the following command:

.. code-block:: bash

   docker create -v /srv --name pmm-client-data percona/pmm-client:2 /bin/true

.. note:: This container does not run, but exists only to make sure you retain
      all PMM data when upgrading to a newer image.


* The ``-v`` option initializes a data volume for the container.

* The ``--name`` option assigns a name for the container
  to reference the container within a Docker network.

* ``percona/pmm-client:2`` is the name and version tag of the image
  to derive the container from.

* ``/bin/true`` is the command that the container runs.

======================================================================
Run the PMM Client Docker Container
======================================================================

.. code-block:: bash

   docker run --rm \
       -e PMM_AGENT_SERVER_ADDRESS=PMMServer:443 \
       -e PMM_AGENT_SERVER_USERNAME=admin \
       -e PMM_AGENT_SERVER_PASSWORD=admin \
       -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
       -e PMM_AGENT_SETUP=1 \
       -e PMM_AGENT_CONFIG_FILE=pmm-agent.yml \
       --volumes-from pmm-client-data \
       perconalab/pmm-client:dev-latest


.. rubric:: ENVIRONMENT VARIABLES

``PMM_AGENT_SERVER_ADDRESS``
    The PMM Server hostname and port number.

``PMM_AGENT_SERVER_USERNAME``
    The PMM Server user name.

``PMM_AGENT_SERVER_PASSWORD``
    The PMM Server user's password.

``PMM_AGENT_SERVER_INSECURE_TLS``
    If true (1), use insecure TLS. Otherwise, do not.

``PMM_AGENT_SETUP``
    If true (1), run ``pmm-agent setup``. Default: false (0).

``PMM_AGENT_CONFIG_FILE``
    The PMM Agent configuration file.


To get help:

.. code-block:: bash

   docker run --rm perconalab/pmm-client:dev-latest --help

   
     
.. _`percona/pmm-client`: https://hub.docker.com/r/percona/pmm-client/tags/
.. _`Docker Docs`: https://docs.docker.com


