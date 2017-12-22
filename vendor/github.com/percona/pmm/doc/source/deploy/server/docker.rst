.. _run-server-docker:

================================================================================
Running |pmm-server| via |docker|
================================================================================

|docker| images of |pmm-server| are stored at the `percona/pmm-server`_ public
repository. The host must be able to run |docker| 1.12.6 or later, and have
network access.

For more information about using |docker|, see the `Docker Docs`_.

.. note::
   
   Make sure that the firewall and routing rules of the host
   do not constrain the |docker| container.
   For more information, see :ref:`troubleshoot-connection`.

.. toctree::
   :maxdepth: 1

   docker.setting-up
   docker.upgrading
   docker.backing-up
   docker.restoring

.. include:: ../../.res/replace/name.txt
.. include:: ../../.res/replace/url.txt
