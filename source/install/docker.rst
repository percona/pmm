.. _backup-container-removing:
.. _container-creating:
.. _container-renaming:
.. _data-container:
.. _image-pulling:
.. _pmm-docker-backup-container-removing:
.. _pmm-docker-previous-version-restoring:
.. _pmm-server-docker-restoring:
.. _pmm.deploying.docker-container.creating:
.. _pmm.deploying.docker-image.pulling:
.. _pmm.deploying.server.docker-container.renaming:
.. _pmm.docker.specific-version:
.. _pmm.server.docker-backing-up:
.. _pmm.server.docker-image.pulling:
.. _pmm.server.docker-setting-up:
.. _run-server-docker:
.. _server-container:
.. _update-server.docker:

################################
PMM Server as a Docker container
################################

************
Introduction
************

PMM Server can run as a container with `Docker <https://docs.docker.com>`__ 1.12.6 or later. Images are available at `<https://hub.docker.com/r/percona/pmm-server>`__.

The Docker tags used here are for the latest version of PMM 2 (|release|) but you can specify any available tag to use the corresponding version of PMM Server.

Metrics collection consumes disk space. PMM needs approximately 1GB of storage for each monitored database node with data retention set to one week. (By default, data retention is 30 days.) To reduce the size of the Prometheus database, you can consider disabling table statistics.

Although the minimum amount of memory is 2 GB for one monitored database node, memory usage does not grow in proportion to the number of nodes. For example, 16GB is adequate for 20 nodes.

.. seealso::

   - :ref:`performance-issues`
   - :ref:`data-retention`

************
Run an image
************

1. Pull an image.

   .. code-block:: bash

      # Pull the latest 2.x image
      docker pull percona/pmm-server:2

2. Create a persistent data container.

   .. code-block:: bash

      docker create --volume /srv \
      --name pmm-data percona/pmm-server:2 /bin/true

   .. caution::

      PMM Server expects the data volume (specified with ``--volume``) to be ``/srv``.  Using any other value will result in data loss when upgrading.

3. Run the image to start PMM Server.

   .. code-block:: bash

      docker run --detach --restart always \
      --publish 80:80 --publish 443:443 \
      --volumes-from pmm-data --name pmm-server \
      percona/pmm-server:2

   .. note::

      You can disable manual updates via the Home Dashboard *PMM Upgrade* panel by adding ``-e DISABLE_UPDATES=true`` to the ``docker run`` command.

4. In a web browser, visit *server hostname*:80 or *server hostname*:443 to see the PMM user interface.

******************
Backup and upgrade
******************

1. Find out which version is installed.

   .. code-block:: bash

      docker exec -it pmm-server curl -u admin:admin http://localhost/v1/version

   .. note::

      Use ``jq`` to extract the quoted string value.

      .. code-block:: bash

         sudo apt install jq # Example for Debian, Ubuntu
         docker exec -it pmm-server curl -u admin:admin http://localhost/v1/version | jq .version

2. Check container mount points are the same (``/srv``).

   .. code-block:: bash

      docker inspect pmm-data | grep Destination
      docker inspect pmm-server | grep Destination

      # With jq
      docker inspect pmm-data | jq '.[].Mounts[].Destination'
      docker inspect pmm-server | jq '.[].Mounts[].Destination'

3. Stop the container and create backups.

   .. code-block:: bash

      docker stop pmm-server
      docker rename pmm-server pmm-server-backup
      mkdir pmm-data-backup && cd $_
      docker cp pmm-data:/srv .

4. Pull and run the latest image.

   .. code-block:: bash

      docker pull percona/pmm-server:2
      docker run \
      --detach \
      --restart always \
      --publish 80:80 --publish 443:443 \
      --volumes-from pmm-data \
      --name pmm-server \
      percona/pmm-server:2

5. (Optional) Repeat step 1 to confirm the version, or check the *PMM Upgrade* panel on the *Home Dashboard*.

*********************
Downgrade and restore
*********************

1. Stop and remove the running version.

   .. code-block:: bash

      docker stop pmm-server
      docker rm pmm-server

2. Restore backups.

   .. code-block:: bash

      docker rename pmm-server-backup pmm-server
      # cd to wherever you saved the backup
      docker cp srv pmm-data:/

3. Restore permissions.

   .. code-block:: bash

      docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R root:root /srv && \
      docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/alertmanager && \
      docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R root:pmm /srv/clickhouse && \
      docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R grafana:grafana /srv/grafana && \
      docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/logs && \
      docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R postgres:postgres /srv/postgres && \
      docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/prometheus && \
      docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R postgres:postgres /srv/logs/postgresql.log

4. Start (don't run) the image.

   .. code-block:: bash

      docker start pmm-server
