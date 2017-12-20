.. _update-server.docker:

Updating |pmm-server| Using |docker|
================================================================================

To check the version of |pmm-server|, run |docker.ps| on the host.

|tip.run-all.root|

.. include:: ../../.res/code/sh.org
   :start-after: +docker.ps.+1.4.0+
   :end-before: #+end-block

The version number is visible in the :guilabel:`IMAGE` column. For a
|docker| container created from the image tagged |opt.latest|, the
:guilabel:`IMAGE` column contains |opt.latest| and not the specific
version number of |pmm-server|.

The information about the currently installed version of |pmm-server| is
available from the |srv.update.main.yml| file. You may extract the version number by using
the |docker.exec| command:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.exec.it.pmm-server.head+
   :end-before: #+end-block

To check if there exists a newer version of |pmm-server|,
visit `percona/pmm-server`_.

.. _pmm/docker/pmm-server/container.renaming:

Creating a backup version of the current |opt.pmm-server| container
--------------------------------------------------------------------------------

You need to create a backup version of the current |opt.pmm-server| container if
the update procedure does not complete successfully or if you decide not to
upgrade your |pmm-server| after trying the new version.

The |docker.stop| command stops the currently running |opt.pmm-server| container:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.stop.pmm-server+
   :end-before: #+end-block

The following command simply renames the current |opt.pmm-server| container to
avoid name conflicts during the update procedure:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.rename.pmm-server.pmm-server-backup+
   :end-before: #+end-block

.. _pmm/docker/image.pulling:

Pulling a new |docker| image
--------------------------------------------------------------------------------

|docker| images for all versions of |pmm| are available from
`percona/pmm-server`_
|docker| repository.

When pulling a newer |docker| image, you may either use a specific version
number or the |opt.latest| image which always matches the highest version
number. 

This example shows how to pull a specific version:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.pull.percona-pmm-server.+1.5.0+
   :end-before: #+end-block

This example shows how to pull the |opt.latest| version:
   
.. include:: ../../.res/code/sh.org
   :start-after: +docker.pull.percona-pmm-server-latest+
   :end-before: #+end-block
   
Creating a new container based on the new image
--------------------------------------------------------------------------------

After you have pulled a new version of |pmm| from the |docker| repository, you can
use |docker.run| to create a |opt.pmm-server| container using the new image.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.run.latest+
   :end-before: #+end-block

.. important::

   The |opt.pmm-server| container must be stopped before attempting |docker.run|.


The |docker.run| command refers to the pulled image as the last parameter. If
you used a specific version number when running |docker.pull| (see
:ref:`pmm/docker/image.pulling`) replace |opt.latest| accordingly.

Note that this command also refers to |opt.pmm-data| as the value of
|opt.volumes-from| option. This way, your new version will continue to use the
existing data.

.. warning:: Do not remove the |opt.pmm-data| container when updating,
	     if you want to keep all collected data.

Check if the new container is running using |docker.ps|.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.ps.+1.5.0+
   :end-before: #+end-block

Then, make sure that the |pmm| version has been updated (see :term:`PMM
Version`) by checking the |pmm-server| web interface.

.. _pmm/docker/backup-container.removing:

Removing the backup container
--------------------------------------------------------------------------------

After you have tried the features of the new version, you may decide to
continupe using it. The backup container that you have stored
(:ref:`pmm/docker/pmm-server/container.renaming`) is no longer needed in this
case.

To remove this backup container, you need the |docker.rm| command:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.rm.pmm-server-backup+
   :end-before: #+end-block

As the parameter to |docker.rm|, supply the tag name of your backup container.

.. _pmm/docker/previous-version.restoring:

.. rubric:: **Restoring the previous version**

If, for whatever reason, you decide to keep using the old version, you just need
to stop and remove the new |opt.pmm-server| container.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.stop.pmm-server&docker.rm.pmm-server+
   :end-before: #+end-block

Now, rename the |opt.pmm-server-backup| to |opt.pmm-server|
(see :ref:`pmm/docker/pmm-server/container.renaming`) and start it.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.start.pmm-server+
   :end-before: #+end-block

.. warning::

   Do not use the |docker.run| command to start the container. The |docker.run|
   command creates and then runs a new container.

   To start a new container use the |docker.start| command.

.. seealso:: 

   Setting up a |docker| container
      :ref:`pmm/server/docker.setting-up`
   Backing Up the |pmm-server| |docker| container
      :ref:`pmm/server/docker/backing-up`
   Updating the |pmm-server| and the |pmm-client|
      :ref:`deploy-pmm.updating` section.

.. References
   
.. include:: ../../.res/replace/name.txt
.. include:: ../../.res/replace/program.txt
.. include:: ../../.res/replace/option.txt
.. include:: ../../.res/replace/fragment.txt
.. include:: ../../.res/replace/url.txt
