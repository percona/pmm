.. _run-server:

==================
Running PMM Server
==================

*PMM Server* combines the backend API and storage for collected data
with a frontend for viewing time-based graphs
and performing thorough analysis of your MySQL and MongoDB hosts
through a web interface.
Run it on a host that you will use to access this data.

There are several options available to run *PMM Server*:

* :ref:`Run PMM Server using Docker <run-server-docker>`

* :ref:`Run PMM Server using VirtualBox <run-server-vbox>`

* :ref:`Run PMM Server using Amazon Machine Image (AMI) <run-server-ami>`

.. _verify-server:

Verifying PMM Server
====================

When you run *PMM Server*,
you should be able to access the PMM web interface
using the IP address of the host where the container is running.
For example, if it is running on 192.168.100.1 with default port 80,
you should be able to access the following:

==================================== ======================================
Component                            URL
==================================== ======================================
PMM landing page                     ``http://192.168.100.1``
Query Analytics (QAN web app)        ``http://192.168.100.1/qan/``
Metrics Monitor (Grafana)            | ``http://192.168.100.1/graph/``
                                     | User name: ``admin``
                                     | Password: ``admin``
Orchestrator                         ``http://192.168.100.1/orchestrator``
==================================== ======================================

.. toctree::
   :hidden:

   docker
   vbox
   ami
   remove
   upgrade

