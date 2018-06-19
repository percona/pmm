.. _pmm/release/1-1-1:

|pmm.name| |release|
********************************************************************************

:Date: February 20, 2017
:PMM Server: https://hub.docker.com/r/percona/pmm-server/
:PMM Client: https://www.percona.com/downloads/pmm-client/

For install instructions, see :ref:`deploy-pmm`.

This release introduces new ways for running *PMM Server*:

* :ref:`Run PMM Server as a virtual appliance <pmm/deploying/server/virtual-appliance>`

* :ref:`Run PMM Server using Amazon Machine Image (AMI) <run-server-ami>`

.. note:: These images are experimental and not recommended for production.
   It is best to :ref:`run PMM Server using Docker <run-server-docker>`.

There are no changes compared to previous :ref:`1.1.0 Beta <pmm/release/1-1-0>` release,
except small fixes for MongoDB metrics dashboards.

.. |release| replace::  1.1.1

.. include:: .res/replace.txt
