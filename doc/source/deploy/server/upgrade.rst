.. _upgrade-server:

====================
Upgrading PMM Server
====================

When a new version of PMM becomes available,
:ref:`remove PMM Server <remove-server>`
and :ref:`run the new version <run-server>`.

For example, if you are :ref:`running a Docker container <run-server-docker>`:

.. code-block:: bash

   $ docker stop pmm-server
   $ docker rm pmm-server
   $ docker run -d \
      -p 80:80 \
      --volumes-from pmm-data \
      --name pmm-server \
      --restart always \
      percona/pmm-server:1.1.5

.. warning:: Do not remove the ``pmm-data`` container when upgrading,
   if you want to keep all collected data.

