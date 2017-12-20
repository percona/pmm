.. _pmm/server/docker/backing-up:

================================================================================
Backing Up PMM Data from the |docker| Container
================================================================================

When |pmm-server| is run via |docker|, its data are stored in the |opt.pmm-data|
container. To avoid data loss, you can extract the data and store outside of the
container.

This example demonstrates how to back up |pmm| data on the computer
where the |docker| container is run and then how to restore them.

To back up the information from |opt.pmm-data|, you need to create a local
directory with essential sub folders and then run |docker| commands to
copy |pmm| related files into it.

#. Create a backup directory and make it the current working
   directory. In this example, we use *pmm-data-backup* as the
   directory name.

   .. code-block:: bash

      $ mkdir pmm-data-backup; cd pmm-data-backup

#. Create the essential sub directories:

   .. code-block:: bash

      $ mkdir -p opt/prometheus
      # mkdir -p var/lib

|tip.run-all.root|

#. Stop the docker container:

   .. code-block:: bash

      $  docker stop pmm-server

#. Copy data from the |opt.pmm-data| container:

   .. code-block:: bash

      $ docker cp pmm-data:/opt/prometheus/data opt/prometheus/
      $ docker cp pmm-data:/opt/consul-data opt/
      $ docker cp pmm-data:/var/lib/mysql var/lib/
      $ docker cp pmm-data:/var/lib/grafana var/lib/

Now, your |pmm| data are backed up and you can start |pmm-server| again:

.. code-block:: bash

   $ docker start pmm-server

.. seealso::

   Restoring |opt.pmm-data|
      :ref:`pmm/server/docker.restoring`

   Updating |pmm-server| run via |docker|
      :ref:`update-server.docker`


.. include:: ../../.res/replace/name.txt
.. include:: ../../.res/replace/option.txt
.. include:: ../../.res/replace/fragment.txt
