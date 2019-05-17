.. _pmm.server.docker-backing-up:

Backing Up |pmm| Data from the |docker| Container
================================================================================

When |pmm-server| is run via |docker|, its data are stored in the |opt.pmm-data|
container. To avoid data loss, you can extract the data and store outside of the
container.

This example demonstrates how to back up |pmm| data on the computer where the
|docker| container is run and then how to restore them.

To back up the information from |opt.pmm-data|, you need to create a local
directory with essential sub folders and then run |docker| commands to copy
|pmm| related files into it.

#. Create a backup directory and make it the current working directory. In this
   example, we use *pmm-data-backup* as the directory name.

   .. include:: ../.res/code/mkdir.pmm-data-backup.cd.txt

#. Create the essential sub directory:

   .. include:: ../.res/code/mkdir.opt-prometheus.var-lib.txt

|tip.run-all.root|

#. Stop the docker container:

   .. include:: ../.res/code/docker.stop.pmm-server.txt

#. Copy data from the |opt.pmm-data| container:

   .. include:: ../.res/code/docker.cp.pmm-data.txt

Now, your |pmm| data are backed up and you can start |pmm-server| again:

.. include:: ../.res/code/docker.start.pmm-server.txt

.. seealso::

   Restoring |opt.pmm-data|
      :ref:`pmm.server.docker-restoring`

   Updating |pmm-server| run via |docker|
      :ref:`update-server.docker`

.. include:: ../.res/replace.txt
