.. _pmm.pmm-admin.mongodb.add-mongodb:

`Adding MongoDB Service Monitoring <pmm-admin.html#pmm-pmm-admin-mongodb-add-mongodb>`_
========================================================================================

Before adding MongoDB should be `prepared for the monitoring <https://www.percona.com/doc/percona-monitoring-and-management/2.x/conf-mongodb.html>`_, which involves creating the user, and setting the profiling level.

When done, add monitoring as follows:

  .. code-block:: bash

     pmm-admin add mongodb --username=pmm --password=pmm

where username and password are credentials for the monitored MongoDB access,
which will be used locally on the database host. Additionally, two positional
arguments can be appended to the command line flags: a service name to be used
by PMM, and a service address. If not specified, they are substituted
automatically as ``<node>-mongodb`` and ``127.0.0.1:27017``.

The command line and the output of this command may look as follows:

  .. code-block:: bash

     # pmm-admin add mongodb --username=pmm --password=pmm mongo 127.0.0.1:27017
     MongoDB Service added.
     Service ID  : /service_id/f1af8a88-5a95-4bf1-a646-0101f8a20791
     Service name: mongo

Beside positional arguments shown above you can specify service name and
service address with the following flags: ``--service-name``, ``--host`` (the
hostname or IP address of the service), and ``--port`` (the port number of the
service). If both flag and positional argument are present, flag gains higher
priority. Here is the previous example modified to use these flags::

     pmm-admin add mongodb --username=pmm --password=pmm --service-name=mongo --host=127.0.0.1 --port=27017

.. include:: ../.res/replace.txt
