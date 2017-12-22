.. _pmm/server/docker.restoring:

Restoring the Backed Up Information to the PMM Data Container
================================================================================

If you have a backup copy of your |opt.pmm-data| container, you can restore it
into a |docker| container. Start with renaming the existing |pmm| containers to
prevent data loss, create a new |opt.pmm-data| container, and finally copy the
backed up information into the |opt.pmm-data| container.

|tip.run-all.root|

#. Stop the running |opt.pmm-server| container.

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.stop.pmm-server+
      :end-before: #+end-block

#. Rename the |opt.pmm-server| container to |opt.pmm-server-backup|.

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.rename.pmm-server.pmm-server-backup+
      :end-before: #+end-block

#. Rename the |opt.pmm-data| to |opt.pmm-data-backup|

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.rename.pmm-data.pmm-data-backup+
      :end-before: #+end-block

#. Create a new |opt.pmm-data| container

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.create.percona-pmm-server-latest+
      :end-before: #+end-block
   
.. important:: The last step creates a new |opt.pmm-data| container based on the
	       |opt.pmm-server.latest| image. If you do not intend to use the
	       |opt.latest| tag, specify the exact version instead, such as
	       **1.5.0**. You can find all available versions of
	       |opt.pmm-server| images at `percona/pmm-server`_.

Assuming that you have a backup copy of your |opt.pmm-data|, created according
to the procedure described in the:ref:`pmm/server/docker/backing-up` section,
restore your data as follows:

#. Change the working directory to the directory that contains your
   |opt.pmm-data| backup files.

   .. include:: ../../.res/code/sh.org
      :start-after: +cd.pmm-data-backup+
      :end-before: #+end-block

   .. note:: This example assumes that the backup directory is found in your
             home directory.
	     
#. Copy data from your backup directory to the |opt.pmm-data| container.

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.cp+
      :end-before: #+end-block
 
#. Apply correct ownership to |opt.pmm-data| files:

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.run.rm.it.chown+
      :end-before: #+end-block
 
#. Run (create and launch) a new |opt.pmm-server| container:

   .. include:: ../../.res/code/sh.org
      :start-after: +docker.run.latest+
      :end-before: #+end-block

To make sure that the new server is available run the |pmm-admin.check-network|
command from the computer where |pmm-client| is installed. |tip.run-this.root|.

.. include:: ../../.res/code/sh.org
   :start-after: +pmm-admin.check-network+
   :end-before: #+end-block

.. seealso::

   Setting up |pmm-server| via |docker|
      :ref:`pmm/server/docker.setting-up`
   Updating PMM
     :ref:`Updating PMM <deploy-pmm.updating>`
   Backing Up the |pmm-server| |docker| container
      :ref:`pmm/server/docker/backing-up`


.. include:: ../../.res/replace/fragment.txt
.. include:: ../../.res/replace/name.txt
.. include:: ../../.res/replace/option.txt
.. include:: ../../.res/replace/program.txt
.. include:: ../../.res/replace/url.txt
