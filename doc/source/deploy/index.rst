.. _deploy-pmm:

===========================================
Deploying |product-name|
===========================================

|product-intro| is designed to be scalable for various environments.
If you have just one MySQL or MongoDB server, you can install and run
both |product-abbrev| server and |product-abbrev| clients on one
database host.

It is more typical to have several MySQL and MongoDB server instances
distributed over different hosts. In this case, you need to install
the |product-abbrev| client package on each database host that you want
to monitor. In this scenario, the |product-abbrev| server is set up on
a dedicated monitoring host.

.. _deploy-pmm.server.installing:

Installing the Server
================================================================================

To install and set up the |product-abbrev| server, use one of the
following options:

-  :ref:`run-server-docker`
-  :ref:`run-server-ova`
-  :ref:`run-server-ami`

.. toctree::
   :hidden:

   server/docker
   server/virtual-appliance
   server/ami

.. _deploy-pmm.server.verifying:

Verifying |product-abbrev| Server
--------------------------------------------------------------------------------

In your browser, go to the server by its IP address. In the admin interface that
opens, set up the user name, password, and your public key if you intend to
connect to the server by using ssh.

In the given example, you would need to direct your browser to
*http://192.168.100.1*. Since you have not added any monitoring services yet, the
site will not show any data.

.. table:: Accessing the Components of the |product-abbrev| Server Web Interface

   ==================================== ======================================
   Component                            URL
   ==================================== ======================================
   :term:`PMM Home Page`                ``http://192.168.100.1``
   :term:`Query Analytics (QAN)`        ``http://192.168.100.1/qan/``
   :term:`Metrics Monitor (MM)`         | ``http://192.168.100.1/graph/``
                                        | User name: ``admin``
                                        | Password: ``admin``
   Orchestrator                         ``http://192.168.100.1/orchestrator``
   ==================================== ======================================

.. _deploy-pmm.client.installing:

Installing Clients
================================================================================

|company-name| provides |product-abbrev| client packages through
software repositories of popular Linux distributions:

* :ref:`DEB packages for Debian or Ubuntu <install-client-apt>`
* :ref:`RPM packages for Red Hat or CentOS <install-client-yum>`

It is recommended that you install your |product-abbrev| client by using the
software repository for your system. If this option does not work for you,
|company-name| provides downloadable |product-abbrev| client packages
from the `Download Percona Monitoring and Management
<https://www.percona.com/downloads/pmm-client>`_ page.

In addition to DEB and RPM packages, this site also offers:

* Generic tarballs that you can extract and run the included ``install`` script.
* Source code tarball to build your |product-abbrev| client from source.

   
.. _deploy-pmm.client_server.connecting:

Connecting |product-abbrev| Clients to the |product-abbrev| Server
================================================================================

With your server and clients set up, you need to establish connection
from clients to the server by specifying the IP address of the server
as a parameter to the ``pmm-admin config --server`` command.

For example, if your |product-abbrev| server is running on `192.168.100.1`,
and you have installed |product-abbrev| client on a machine with IP
`192.168.200.1`, run the following in the terminal of your client:

.. code-block:: bash

   $ sudo pmm-admin config --server 192.168.100.1
   OK, PMM server is alive.

   PMM Server      | 192.168.100.1
   Client Name     | ubuntu-amd641
   Client Address  | 192.168.200.1

.. note:: If you change the default port 80
   when :ref:`running PMM Server <deploy-pmm.server.installing>`,
   specify it after the server's IP address. For example:

   .. code-block:: bash

      $ sudo pmm-admin config --server 192.168.100.1:8080


.. _deploy-pmm.data-collecting:

Collecting Data from |product-abbrev| Clients on |product-abbrev| Server
========================================================================

To start collecting data on each |product-abbrev| client connected to a
|product-abbrev| server, run :program:`pmm-admin add` command along with the
name of the selected monitoring service.

For example, to enable general system metrics, MySQL metrics,
as well as MySQL query analytics, run :program:`pmm-admin` as follows:

.. code-block:: bash

   $ sudo pmm-admin add mysql

To enable general system metrics, MongoDB metrics,
and MongoDB query analytics, run:

.. code-block:: bash

   $ sudo pmm-admin add mongodb

To enable ProxySQL performance metrics, run:

.. code-block:: bash

   $ sudo pmm-admin add proxysql:metrics

To see what is being monitored, run:

.. code-block:: bash

   $ sudo pmm-admin list

For example, if you enable general OS and MongoDB metrics monitoring,
the output should be similar to the following:

.. code-block:: text

   $ sudo pmm-admin list

   ...

   PMM Server      | 192.168.100.1
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.200.1
   Service manager | linux-systemd

   ---------------- ----------- ----------- -------- ---------------- --------
   SERVICE TYPE     NAME        LOCAL PORT  RUNNING  DATA SOURCE      OPTIONS
   ---------------- ----------- ----------- -------- ---------------- --------
   linux:metrics    mongo-main  42000       YES      -
   mongodb:metrics  mongo-main  42003       YES      localhost:27017

For more information about the available commands of :program:`pmm-admin add`,
run :program:`pmm-admin add --help` in your terminal.

.. _deploy-pmm.updating:

Updating
================================================================================

When changing to a new version of |product-abbrev|, you update the
|product-abbrev| server and each |product-abbrev| client separately.

The updating procedure of your |product-abbrev| server, depends on the option
that you selected for installing it. If you have isntalled your |product-abbrev|
server from a :program:`docker` image, follow instruction in the
:ref:`update-server.docker` section.

If you are running *PMM Server* as a :ref:`virtual appliance <run-server-ova>`
or using :ref:`Amazon Machine Image <run-server-ami>`, you can use the update
button in the bottom right corner of the |product-abbrev| home page (see
:term:`PMM Home Page`).

.. figure:: ../images/update-button.png

   Update your server by clicking the *Update* button on the |product-abbrev|
   landing page.

.. rubric:: Updating |product-abbrev| clients

When a newer version of *PMM Client* becomes available, you can update to it
from the Percona software repositories:

* For Debian or Ubuntu::

   $ sudo apt-get update && sudo apt-get install pmm-client

* For RedHat or CentOS::

   $ yum update pmm-client

If you have installed your |product-abbrev| client manually, you need
to :ref:`remove it <deploy-pmm.removing>` and then :ref:`download and
install a newer version <deploy-pmm.client.installing>`.

.. _deploy-pmm.removing:

Removing the |product-abbrev| Client and |product-abbrev| Server
================================================================================

Each |product-abbrev| client and the |product-abbrev| server are removed
separately. First, remove all monitored services by using the
:program:`pmm-admin remove` command (see :ref:`pmm-admin-rm`). Then you can
remove each |product-abbrev| client and the |product-abbrev| server.

.. rubric:: Removing the |product-abbrev| Client

The exact procedure of removing the |product-abbrev| client, depends
on the method of installation:

- Removing an installed package using YUM:

  .. code-block:: bash

     $ sudo yum remove pmm-client
  
- Removing an installed package using APT:

  .. code-block:: bash

     $ sudo apt-get remove pmm-client
  
- Removing a manually installed RPM package:

  .. code-block:: bash

     $ rpm -e pmm-client

- Removing a manually installed DEB package:

  .. code-block:: bash

     $ dpkg -r pmm-client
  
- Removing a binary installed by using the generic |product-abbrev|
  client tarball (assuming you have changed into the directory where
  the tarball contents was extracted to):
  
  .. code-block:: bash

      $ sudo ./uninstall

.. rubric:: Removing the |product-abbrev| Server

If you run your |product-abbrev| server using a :program:`Docker`,
stop the container as follows:

.. code-block:: bash

   $ docker stop pmm-server && docker rm pmm-server

To discard all collected data (if you do not plan to user
|product-abbrev| server in the future), remove the ``pmm-data``
container:

.. code-block:: bash

   $ docker rm pmm-data

If you run your |product-abbrev| server using a virtual appliance,
just stop and remove it.

To terminate the |product-abbrev| server running from an Amazon
machine image, run the following command in your terminal:

.. code-block:: bash

   $ aws ec2 terminate-instances --instance-ids -i-XXXX-INSTANCE-ID-XXXX

.. toctree::
   :hidden:

   server/index
   client/index
   connect-client
   start-collect

.. seealso::

   - :ref:`architecture`
   - :ref:`pmm-admin-add`.

.. include:: ../replace.txt
