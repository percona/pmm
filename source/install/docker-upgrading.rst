.. _update-server.docker:
.. _pmm.deploying.server.docker-container.renaming:
.. _container-renaming:
.. _pmm.deploying.docker-image.pulling:
.. _image-pulling:
.. _pmm.deploying.docker-container.creating:
.. _container-creating:

################################
Updating PMM Server Using Docker
################################

1. Check the installed version of PMM Server. There are two methods.

   1. Use ``docker ps``:

      .. code-block:: bash

         docker ps

      This will show the version tag appended to the image name (e.g. ``percona/pmm-server:2``).

   2. Use ``docker exec``:

      .. code-block:: bash

         docker exec -it pmm-server curl -u admin:admin http://localhost/v1/version

      This will print a JSON string containing version fields.

2. Check if there is a newer version of PMM Server.

   Visit `<https://hub.docker.com/r/percona/pmm-server/tags/>`_.

3. Stop the container and create backups.

   Back-up the current container and its data so that
   you can revert back to using them, and as a safeguard in case
   the update procedure fails.

   .. code-block:: bash

      docker stop pmm-server
      docker rename pmm-server pmm-server-backup
      docker cp pmm-data pmm-data-backup

4. Pull the new PMM Server image.

   You may specify an exact version number, or the latest.

   To pull a specific version (|release| in this example):

   .. parsed-literal::

      docker pull percona/pmm-server:|release|

   To pull the latest version of PMM 2:

   .. code-block:: bash

      docker pull percona/pmm-server:2

5. Run the image.

   .. code-block:: bash

      docker run -d -p 80:80 -p 443:443 --volumes-from pmm-data --name pmm-server --restart always percona/pmm-server:<VERS>

   - ``<VERS>`` is the image version pulled in the previous step.
   - ``pmm-data`` is your existing data image.

6. Check the new version.

   Repeat step 1. You can also check the PMM Server web interface.

.. _pmm/docker/previous-version.restoring:

******************************
Restoring the previous version
******************************

1. Stop and remove the running version.

   .. code-block:: bash

      docker stop pmm-server
      docker rm pmm-server
      docker rm pmm-data

2. Restore (rename) the backups.

   .. code-block:: bash

      docker rename pmm-server-backup pmm-server
      docker rename pmm-data-backup pmm-data

3. Start (don't ``run``) the image.

   .. code-block:: bash

      $ docker start pmm-server

.. _pmm/docker/backup-container.removing:
.. _backup-container-removing:

*****************************
Removing the backup container
*****************************

If you stay with the new version and are sure you no longer need your backup containers, you can remove them.

.. code-block:: bash

   docker rm pmm-server-backup
   docker rm pmm-data-backup
