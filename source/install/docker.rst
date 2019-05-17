.. _run-server-docker:
--------------------------------------------------------------------------------
Running |pmm-server| via |docker|
--------------------------------------------------------------------------------

|docker| images of |pmm-server| are stored at the `percona/pmm-server`_ public
repository. The host must be able to run |docker| 1.12.6 or later, and have
network access.

|pmm| needs roughly 1GB of storage for each monitored database node with data
retention set to one week. Minimum memory is 2 GB for one monitored database
node, but it is not linear when you add more nodes.  For example, data from 20
nodes should be easily handled with 16 GB.

Make sure that the firewall and routing rules of the host do not constrain the
|docker| container. For more information, see :ref:`troubleshoot-connection`.

For more information about using |docker|, see the `Docker Docs`_.

.. important::

   By default, :ref:`retention <data-retention>` is set to 30 days for
   |metrics-monitor| and to 8 days for |qan.name|.  Also consider
   :ref:`disabling table statistics <performance-issues>`, which can greatly
   decrease |prometheus| database size.

.. toctree::
   :name: dockertoc
   :maxdepth: 1

   docker-setting-up
   docker-upgrading
   docker-backing-up
   docker-restoring

.. _`percona/pmm-server`: https://hub.docker.com/r/percona/pmm-server/tags/
.. _`Docker Docs`: https://docs.docker.com

.. include:: ../.res/replace.txt
