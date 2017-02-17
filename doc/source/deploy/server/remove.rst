.. _remove-server:

===================
Removing PMM Server
===================

Before you stop and remove *PMM Server*,
make sure that the related *PMM Clients* are not collecting any data
by removing all monitored instances as described in :ref:`pmm-admin-rm`.

* If you are :ref:`running a Docker container <run-server-docker>`:

  1. Stop and remove the ``pmm-server`` container:

     .. code-block:: bash

        $ docker stop pmm-server && docker rm pmm-server

  #. If you also want to discard all collected data,
     remove the ``pmm-data`` container:

     .. code-block:: bash

        $ docker rm pmm-data

* If you are :ref:`running an image in VirtualBox <run-server-vbox>`,
  stop the *PMM Server* appliance.

  Remove the appliance if necessary.

* If you are :ref:`running an Amazon Machine Image <run-server-ami>`,
  terminate the instance using the ``terminate-instances`` command.
  For example:

  .. code-block:: bash

     $ aws ec2 terminate-instances --instance-ids i-XXXX-INSTANCE-ID-XXXX

