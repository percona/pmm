.. _update-server.docker:

================================================================================
Updating |pmm-server| Using |docker|
================================================================================

To check the version of |pmm-server|, run |docker.ps| on the host.

|tip.run-all.root|

.. include:: ../../.res/code/sh.org
   :start-after: +docker.ps+1.4.0+
   :end-before: #+end-block

The version number is visible in the :guilabel:`IMAGE` column. For a
|docker| container created from the image tagged |opt.latest|, the
:guilabel:`IMAGE` column contains |opt.latest| and not the specific
version number of |pmm-server|.

To check if there exists a newer version of |pmm-server|,
visit https://hub.docker.com/r/percona/pmm-server/tags/.

In outline, the procedure for updating the |pmm| version includes 1) creating a
backup version of the currently used |pmm-server| image, 2) removing the current
|opt.pmm-server| container, and 3) creating a new container based on the |docker|
image with the new |pmm| version.

.. _pmm/docker/backup-image.creating:

Creating a back up version of the current image
================================================================================

Copy the image being used by the |opt.pmm-server|. For this, you use the
|docker.tag| command. This command effectively creates a copy of the image that
you specify as the first parameter to |docker.tag| and applies the new name to
it (the last parameters to the |docker.tag| command).

.. include:: ../../.res/code/sh.org
   :start-after: +docker.tag+
   :end-before: #+end-block

Note that both images have the same ID.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.images.percona+
   :end-before: #+end-block

By running |docker.tag|, you are creating a backup of your current image. If the
update should fail, you can
:ref:`restore and continue using the old version <pmm/docker/previous-version.restoring>`.

.. _pmm/docker/pmm-server/container.removing:

Removing the |opt.pmm-server| container
================================================================================

#. Stop the |opt.pmm-server| container.

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.stop.pmm-server+
      :end-before: #+end-block

   This command may take a few seconds. When the command completes successfully,
   the name of the stopped container appears on the screen.

#. Remove the |opt.pmm-server| container.

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.rm.pmm-server+
      :end-before: #+end-block

   The name of the removed container is printed to the screen when the command
   completes successfully.

   .. warning:: Do not remove the |opt.pmm-data| container when updating,
      if you want to keep all collected data.

.. _pmm/docker/image.pulling:

Pulling a new |docker| image
================================================================================

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
================================================================================

Now, you can use |docker.run| (this command creates and launches the new container in one
step) the |opt.pmm-server| container using the new image.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.run.latest+
   :end-before: #+end-block

The |docker.run| command refers to the pulled image as the last parameter. If
you used a specific version number when running |docker.pull| (see
:ref:`pmm/docker/image.pulling`) replace |opt.latest| accordingly.

Note that this command also refers to |opt.pmm-data| as the value of
|opt.volumes-from| option. This way, your new version will continue to use the
existing data.

Check if the new container is running using |docker.ps|.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.ps.+1.5.0+
   :end-before: #+end-block

Then, make sure that the |pmm| version has been updated (see :term:`PMM
Version`) by checking the |pmm-server| web interface.

.. _pmm/docker/backup-image.removing:

Removing the backup image
================================================================================

After you have tried the features of the new version, you may decide to continue
using it. The backup image that you have created
(:ref:`pmm/docker/backup-image.creating`) is no longer needed in this case.

To remove this backup image, you need the |docker.rmi| command:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.rmi.backup-latest+
   :end-before: #+end-block

As the parameter to |docker.rmi|, supply the tag name of your backup image. To
check which one it is, use the |docker.images| command. If you run this command
without any parameters, it will list all available images. You may specify a
filter and only see the repositories that match your criterion:

.. include:: ../../.res/code/sh.org
   :start-after: +docker.images.percona+
   :end-before: #+end-block


.. _pmm/docker/previous-version.restoring:

.. rubric:: **Restoring the previous version**

If, for whatever reason, you decide to keep using the old version, you need to
remove the updated container and recreate it based on the backup image you have
(see :ref:`pmm/docker/backup-image.creating`).

To remove the updated container, use the steps of the
:ref:`pmm/docker/pmm-server/container.removing` procedure. Then, create
the |opt.pmm-server| container by using the |docker.run| command supplying it the
backup image.

.. include:: ../../.res/code/sh.org
   :start-after: +docker.run.d.p.volumes-from.name.restart.+backup+
   :end-before: #+end-block

.. seealso:: 

   Updating the |pmm-server| and the |pmm-client|
      :ref:`deploy-pmm.updating` section.

.. References
   
.. include:: ../../.res/replace/name.txt
.. include:: ../../.res/replace/program.txt
.. include:: ../../.res/replace/option.txt
.. include:: ../../.res/replace/fragment.txt
