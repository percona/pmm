.. _pmm.server.docker-backing-up:

Backing Up PMM Data from the Docker Container
================================================================================

When PMM Server is run via Docker, its data are stored in the ``pmm-data``
container. To avoid data loss, you can extract the data and store outside of the
container.

This example demonstrates how to back up PMM data on the computer where the
Docker container is run and then how to restore them.

To back up the information from ``pmm-data``, you need to create a local
directory with essential sub folders and then run Docker commands to copy
PMM related files into it.

#. Create a backup directory and make it the current working directory. In this
   example, we use *pmm-data-backup* as the directory name.

   .. include:: ../.res/code/mkdir.pmm-data-backup.cd.txt

#. Create the essential sub directory:

   .. include:: ../.res/code/mkdir.opt-prometheus.var-lib.txt

Run the following commands as root or by using the ``sudo`` command

#. Stop the docker container:

   .. include:: ../.res/code/docker.stop.pmm-server.txt

#. Copy data from the ``pmm-data`` container:

   .. include:: ../.res/code/docker.cp.pmm-data.txt

Now, your PMM data are backed up and you can start PMM Server again:

.. include:: ../.res/code/docker.start.pmm-server.txt

.. seealso::

   Restoring ``pmm-data``
      :ref:`pmm.server.docker-restoring`

   Updating PMM Server run via Docker
      :ref:`update-server.docker`


