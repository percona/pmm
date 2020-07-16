.. _pmm.server.docker-restoring:

#######################################################
Restoring Backed-up Information to a PMM Data Container
#######################################################

You can restore a backup copy of your ``pmm-data`` container with these steps.

1. Stop the container:

   .. code-block:: bash

      $ docker stop pmm-server

2. Rename the container:

   .. code-block:: bash

      $ docker rename pmm-server pmm-server-backup

3. Rename the data container:

   .. code-block:: bash

      $ docker rename pmm-data pmm-data-backup

4. Create a new data container:

   .. code-block:: bash

      $ docker create -v /srv --name pmm-data percona/pmm-server:2 /bin/true


.. note::

   This step creates a new data container based on the latest
   ``percona/pmm-server:2`` image. All available versions of ``pmm-server`` images are listed at
   `<https://hub.docker.com/r/percona/pmm-server/tags/>`_.

Assuming that you have a backup copy of your ``pmm-data`` (see :ref:`pmm.server.docker-backing-up`), restore your data as follows:

1. Change to the directory where your ``pmm-data`` backup files are:

   .. code-block:: bash

      $ cd <path to>/pmm-data-backup

2. Copy data from your backup directory to the ``pmm-data`` container:

   .. code-block:: bash

      $ docker cp srv pmm-data:/srv

3. Apply correct ownership to ``pmm-data`` files:

   .. code-block:: bash

      $ docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv

4. Run (create and launch) a new PMM server container:

   .. code-block:: bash

      $ docker run -d -p 80:80 -p 443:443 --volumes-from pmm-data \
      --name pmm-server --restart always percona/pmm-server:2
