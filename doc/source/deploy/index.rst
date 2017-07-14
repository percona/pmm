.. _deploy-pmm:

===========================================
Deploying Percona Monitoring and Management
===========================================

The following procedure describes how to properly deploy PMM:

1. :ref:`Run PMM Server <run-server>` on the host
   that will be used to access collected data,
   view time-based graphs, and carry out performance analysis.

   The following options are available:

   * :ref:`Run PMM Server using Docker <run-server-docker>`

   * :ref:`Run PMM Server as a virtual appliance <run-server-ova>`

   * :ref:`Run PMM Server using Amazon Machine Image (AMI) <run-server-ami>`

#. :ref:`Install PMM Client <install-client>`
   on every MySQL and MongoDB instance
   that you want to monitor.

   Percona provides *PMM Client* packages for automatic installation
   from software repositories on the most popular Linux distributions:

   * :ref:`Install PMM Client on Debian or Ubuntu <install-client-apt>`

   * :ref:`Install PMM Client on Red Hat or CentOS <install-client-yum>`

#. :ref:`Connect PMM Client to PMM Server <connect-client>`

#. :ref:`Start data collection <start-collect>`

Upgrading
=========

To upgrade PMM:

1. :ref:`Upgrade PMM Server <upgrade-server>`

#. :ref:`Upgrade PMM Client <upgrade-client>`
   on all hosts that you are monitoring

Removing
========

For more information about removing PMM, see the following:

* :ref:`remove-server`
* :ref:`remove-client`

.. toctree::
   :hidden:

   server/index
   client/index
   connect-client
   start-collect

