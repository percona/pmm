.. _deploy-pmm.client.installing:

`Installing Clients <clients.html#installing>`_
================================================================================

|pmm-client| is a package of agents and exporters installed on a database host
that you want to monitor. Before installing the |pmm-client| package on each
database host that you intend to monitor, make sure that your |pmm-server| host
is accessible.

For example, you can run the |ping| command passing the IP address of the
computer that |pmm-server| is running on. For example:

.. code-block:: bash

   $ ping 192.168.100.1

You will need to have root access on the database host where you will be
installing |pmm-client| (either logged in as a user with root privileges or be
able to run commands with |sudo|).

.. rubric:: Supported platforms

|pmm-client| should run on any modern |linux| 64-bit distribution, however
|percona| provides |pmm-client| packages for automatic installation from
software repositories only on the most popular |linux| distributions:

* :ref:`DEB packages for Debian based distributions such as Ubuntu <install-client-apt>`
* :ref:`RPM packages for Red Hat based distributions such as CentOS <install-client-yum>`

It is recommended that you install your |abbr.pmm| client by using the
software repository for your system. If this option does not work for you,
|percona| provides downloadable |pmm-client| packages
from the `Download Percona Monitoring and Management
<https://www.percona.com/downloads/pmm-client>`_ page.

In addition to DEB and RPM packages, this site also offers:

* Generic tarballs that you can extract and run the included ``install`` script.
* Source code tarball to build your |abbr.pmm| client from source.

.. warning:: You should not install agents on database servers that have
   the same host name, because host names are used by |pmm-server| to
   identify collected data.

.. rubric:: Storage requirements
   
Minimum **100** MB of storage is required for installing the |pmm-client|
package. With a good constant connection to |pmm-server|, additional storage is
not required. However, the client needs to store any collected data that it is
not able to send over immediately, so additional storage may be required if
connection is unstable or throughput is too low.

.. include:: ../.res/replace.txt
