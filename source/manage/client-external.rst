

########################
Adding External Services
########################

.. _pmm-admin-add-external-services:

******************************************************************************************
`Adding general external services <client-external.html#pmm-admin-add-external-services>`_
******************************************************************************************

You can collect metrics from an external (custom) exporter on a node when:

- there is already a PMM Agent instance running and,

- this node has been `configured <https://www.percona.com/doc/percona-monitoring-and-management/2.x/manage/client-config.html#deploy-pmm-client-server-connecting>`_ using the ``pmm-admin config`` command.

=====
Usage
=====

.. code-block:: bash

   pmm-admin add external [--service-name=<service-name>] [--listen-port=<listen-port>] [--metrics-path=<metrics-path>] [--scheme=<scheme>]
