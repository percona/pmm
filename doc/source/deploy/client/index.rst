.. _install-client:

================================================================================
Installing PMM Client
================================================================================

|pmm-client| is a package of agents and exporters installed on a |mysql| or
|mongodb| host that you want to monitor.  The components collect various data
about general system and database performance, and send this data to
corresponding |pmm-server| components.

Before installing the |pmm-client| package on a database host, make sure that
your |pmm-server| host is accessible.  For example, you can ``ping
192.168.100.1`` or whatever IP address |pmm-server| is running on.

You will need to have root access on the database host where you will be
installing |pmm-client| (either logged in as a user with root privileges or be
able to run commands with ``sudo``).

The minimum requirements for |qan.intro| are:

* |mysql| 5.1 or later (if using the slow query log)
* |mysql| 5.6.9 or later (if using Performance Schema)

.. note:: You should not install agents on database servers that have
   the same host name, because host names are used by |pmm-server| to
   identify collected data.

|pmm-client| should run on any modern |linux| 64-bit distribution, however
|percona| provides |pmm-client| packages for automatic installation
from software repositories only on the most popular Linux
distributions:

* :ref:`Install PMM Client on Debian or Ubuntu <install-client-apt>`
* :ref:`Install PMM Client on Red Hat or CentOS <install-client-yum>`

Minimum 100 MB of storage is required for installing the |pmm-client| package.
With good constant connection to |pmm-server|, additional storage is not
required.  However, the client needs to store any collected data that it is not
able to send over immediately, so additional storage may be required if
connection is unstable or throughput is too low.

If you are not able to install from |percona|'s software repositories or
running some other |linux| distribution,
try :ref:`deploy-pmm.client.installing`.

.. toctree::
   :hidden:

   apt
   yum
   remove

.. include:: ../../.res/replace/name.txt
