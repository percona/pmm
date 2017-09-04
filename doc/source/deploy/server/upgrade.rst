.. _update-server.docker:

====================================================================================================
Updating PMM Server Using Docker
====================================================================================================

To check the version of the *PMM Server* container, run ``docker ps`` on the
host.  For example:

.. code-block:: bash

   $ docker ps
   CONTAINER ID   IMAGE                      COMMAND                CREATED       STATUS             PORTS                               NAMES
   480696cd4187   percona/pmm-server:1.1.5   "/opt/entrypoint.sh"   4 weeks ago   Up About an hour   192.168.100.1:80->80/tcp, 443/tcp   pmm-server

If there is a newer version
available at https://hub.docker.com/r/percona/pmm-server/tags/:

1. Stop and remove the ``pmm-server`` container:

   .. code-block:: bash

      $ docker stop pmm-server && docker rm pmm-server

   .. warning:: Do not remove the ``pmm-data`` container when updating,
      if you want to keep all collected data.

#. Run new version of *PMM Server*:

   .. code-block:: bash

      $ docker run -d \
        -p 80:80 \
        --volumes-from pmm-data \
        --name pmm-server \
        --restart always \
        percona/pmm-server:latest

#. Confirm that the new version is running using ``docker ps`` again

.. code-block:: bash

   $ docker ps
   CONTAINER ID   IMAGE                      COMMAND                CREATED         STATUS         PORTS                               NAMES
   480696cd4187   percona/pmm-server:1.2.0   "/opt/entrypoint.sh"   4 minutes ago   Up 4 minutes   192.168.100.1:80->80/tcp, 443/tcp   pmm-server

.. seealso:: References

   For information about updating the |product-abbrev| server and the
   |product-abbrev| client, see the :ref:`deploy-pmm.updating` section.

.. include:: ../../replace.txt
