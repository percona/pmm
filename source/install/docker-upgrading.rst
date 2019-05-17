.. _update-server.docker:

Updating |pmm-server| Using |docker|
================================================================================

To check the version of |pmm-server|, run |docker.ps| on the host.

|tip.run-all.root|

.. include:: ../.res/code/docker.ps.1-4-0.txt

The version number is visible in the |gui.image| column. For a |docker|
container created from the image tagged |opt.latest|, the |gui.image| column
contains |opt.latest| and not the specific version number of |pmm-server|.

The information about the currently installed version of |pmm-server| is
available from the |srv.update.main.yml| file. You may extract the version
number by using the |docker.exec| command:

.. include:: ../.res/code/docker.exec.it.pmm-server.head.txt

To check if there exists a newer version of |pmm-server|,
visit `percona/pmm-server`_.

.. _pmm.deploying.server.docker-container.renaming:

`Creating a backup version of the current pmm-server Docker container <docker-upgrading.html#container-renaming>`_
----------------------------------------------------------------------------------------------------------------------------

You need to create a backup version of the current |opt.pmm-server| container if
the update procedure does not complete successfully or if you decide not to
upgrade your |pmm-server| after trying the new version.

The |docker.stop| command stops the currently running |opt.pmm-server| container:

.. include:: ../.res/code/docker.stop.pmm-server.txt

The following command simply renames the current |opt.pmm-server| container to
avoid name conflicts during the update procedure:

.. include:: ../.res/code/docker.rename.pmm-server.pmm-server-backup.txt

.. _pmm.deploying.docker-image.pulling:

`Pulling a new Docker Image <docker-upgrading.html#image-pulling>`_
--------------------------------------------------------------------------------

|docker| images for all versions of |pmm| are available from
`percona/pmm-server`_
|docker| repository.

When pulling a newer |docker| image, you may either use a specific version
number or the |opt.latest| image which always matches the highest version
number. 

This example shows how to pull a specific version:

.. include:: ../.res/code/docker.pull.percona-pmm-server.1-5-0.txt

This example shows how to pull the |opt.latest| version:
   
.. include:: ../.res/code/docker.pull.percona-pmm-server-latest.txt
   
.. _pmm.deploying.docker-container.creating:

`Creating a new Docker container based on the new image <docker-upgrading.html#container-creating>`_
-------------------------------------------------------------------------------------------------------

After you have pulled a new version of |pmm| from the |docker| repository, you can
use |docker.run| to create a |opt.pmm-server| container using the new image.

.. include:: ../.res/code/docker.run.latest.txt

.. important::

   The |opt.pmm-server| container must be stopped before attempting |docker.run|.

The |docker.run| command refers to the pulled image as the last parameter. If
you used a specific version number when running |docker.pull| (see
:ref:`pmm.server.docker-image.pulling`) replace |opt.latest| accordingly.

Note that this command also refers to |opt.pmm-data| as the value of
|opt.volumes-from| option. This way, your new version will continue to use the
existing data.

.. warning:: Do not remove the |opt.pmm-data| container when updating,
	     if you want to keep all collected data.

Check if the new container is running using |docker.ps|.

.. include:: ../.res/code/docker.ps.1-5-0.txt

Then, make sure that the |pmm| version has been updated (see :ref:`PMM
Version <PMM-Version>`) by checking the |pmm-server| web interface.

.. _pmm/docker/backup-container.removing:

`Removing the backup container <docker-upgrading.html#backup-container-removing>`_
----------------------------------------------------------------------------------

After you have tried the features of the new version, you may decide to
continupe using it. The backup container that you have stored
(:ref:`pmm.deploying.server.docker-container.renaming`) is no longer needed in this
case.

To remove this backup container, you need the |docker.rm| command:

.. include:: ../.res/code/docker.rm.pmm-server-backup.txt

As the parameter to |docker.rm|, supply the tag name of your backup container.

.. _pmm/docker/previous-version.restoring:

.. rubric:: **Restoring the previous version**

If, for whatever reason, you decide to keep using the old version, you just need
to stop and remove the new |opt.pmm-server| container.

.. include:: ../.res/code/docker.stop.pmm-server.rm.txt

Now, rename the |opt.pmm-server-backup| to |opt.pmm-server|
(see :ref:`pmm.deploying.server.docker-container.renaming`) and start it.

.. include:: ../.res/code/docker.start.pmm-server.txt

.. warning::

   Do not use the |docker.run| command to start the container. The |docker.run|
   command creates and then runs a new container.

   To start a new container use the |docker.start| command.

.. seealso:: 

   Setting up a |docker| container
      :ref:`pmm.server.docker-setting-up`
   Backing Up the |pmm-server| |docker| container
      :ref:`pmm.server.docker-backing-up`
   Updating the |pmm-server| and the |pmm-client|
      :ref:`deploy-pmm.updating` section.

.. References

.. _`percona/pmm-server`: https://hub.docker.com/r/percona/pmm-server/tags/
   
.. include:: ../.res/replace.txt

