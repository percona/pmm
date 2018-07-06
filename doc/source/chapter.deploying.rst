.. _deploy-pmm:

Deploying |pmm.name|
********************************************************************************

|abbr.pmm| is designed to be scalable for various environments.  If you have
just one |mysql| or |mongodb| server, you can install and run both |abbr.pmm|
server and |abbr.pmm| clients on one database host.

It is more typical to have several |mysql| and |mongodb| server instances
distributed over different hosts. In this case, you need to install the
|abbr.pmm| client package on each database host that you want to monitor. In
this scenario, the |abbr.pmm| server is set up on a dedicated monitoring host.

|chapter.toc|

.. contents::
   :local:
   :depth: 1

.. _deploy-pmm.server.installing:

Installing |pmm-server|
================================================================================

To install and set up the |pmm-server|, use one of the following options:

.. -  :ref:`run-server-docker`
.. -  :ref:`pmm/deploying/server/virtual-appliance`
.. -  :ref:`run-server-ami`

.. toctree::
   :maxdepth: 1

   chapter.deploying.pmm-server.docker
   chapter.deploying.pmm-server.virtual-appliance
   chapter.deploying.pmm-server.ami

.. include:: .res/contents/important.port.txt

.. _deploy-pmm.server.verifying:

Verifying |pmm-server|
--------------------------------------------------------------------------------

In your browser, go to the server by its IP address. If you run your server as a
virtual appliance or by using an |amazon| machine image, you will need to setup
the user name, password and your public key if you intend to connect to the
server by using ssh. This step is not needed if you run |pmm-server| using
|docker|.

In the given example, you would need to direct your browser to
*http://192.168.100.1*. Since you have not added any monitoring services yet,
the site will not show any data.

.. _deploy-pmm.table.web-interface.component.access:

.. table:: Accessing the Components of the Web Interface

   ==================================== ======================================
   Component                            URL
   ==================================== ======================================
   :term:`PMM Home Page`                ``http://192.168.100.1``
   :term:`Metrics Monitor (MM)`         | ``http://192.168.100.1/graph/``
                                        | User name: ``admin``
                                        | Password: ``admin``
   Orchestrator                         ``http://192.168.100.1/orchestrator``
   ==================================== ======================================

You can also check if |pmm-server| is available requesting the /ping
URL as in the following example:

.. include:: .res/code/sh.org
   :start-after: +curl.url-ping+
   :end-before: #+end-block

.. _deploy-pmm.client.installing:

Installing Clients
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
   
.. _deploy-pmm.client_server.connecting:

Connecting |abbr.pmm| Clients to the |pmm-server|
================================================================================

With your server and clients set up, you must configure each |pmm-client| and
specify which |pmm-server| it should send its data to.

To connect a |pmm-client|, enter the IP address of the |pmm-server| as the value
of the |opt.server| parameter to the |pmm-admin.config| command.

.. code-block:: bash

   $ sudo pmm-admin config --server 192.168.100.1:8080

For example, if your |pmm-server| is running on `192.168.100.1`, and you have
installed |pmm-client| on a machine with IP `192.168.200.1`, run the following
in the terminal of your client. |tip.run-all.root|:

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.config.server.url+
   :end-before: #+end-block

If you change the default port **80** when :ref:`running PMM Server
<deploy-pmm.server.installing>`, specify the new port number after the IP
address of |pmm-server|. For example:

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.config.server.url.port+
   :end-before: #+end-block

.. include:: .res/contents/important.port.txt

.. seealso::

   What other options can I pass to |pmm-admin.config|?
      Run |pmm-admin.config| |opt.help|

.. _deploy-pmm.data-collecting:

Collecting Data from |abbr.pmm| Clients on |pmm-server|
================================================================================

To start collecting data on each |pmm-client| connected to a |abbr.pmm|
server, run the |pmm-admin.add| command along with the name of the selected
monitoring service.

|tip.run-all.root|.

Enable general system metrics, |mysql| metrics, |mysql| query analytics:
   .. code-block:: bash

      $ pmm-admin add mysql

Enable general system metrics, |mongodb| metrics, and |mongodb| query analytics:
   .. code-block:: bash

      $ pmm-admin add mongodb

Enable |proxysql| performance metrics:
   .. code-block:: bash

      $ pmm-admin add proxysql:metrics

To see what is being monitored, run |pmm-admin.list|. For example, if you enable
general OS and |mongodb| metrics monitoring, the output should be similar to the
following:

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

.. seealso::

   What other monitoring services can I add using the |pmm-admin.add| command?
      Run :program:`pmm-admin add --help` in your terminal

.. _deploy-pmm.updating:

Updating
================================================================================

When changing to a new version of |pmm|, you update the |pmm-server| and each
|pmm-client| separately.

.. rubric:: Updating the |pmm-server|

The updating procedure of your |pmm-server| depends on the option that you
selected for installing it.

If you are running |pmm-server| as a :ref:`virtual appliance
<pmm/deploying/server/virtual-appliance>` or using an :ref:`Amazon Machine Image
<run-server-ami>`, use the |gui.check-for-updates-manually| button on the Home
dashboard (see :term:`PMM Home Page`).

.. figure:: .res/graphics/png/pmm.home-page.1.png

   Click |gui.check-for-updates-manually| to updating the |pmm-server| from the
   |pmm| home page.

.. seealso::

   How to update |pmm-server| installed using |docker|?
      :ref:`update-server.docker`

.. rubric:: Updating a |pmm-client|

When a newer version of |pmm-client| becomes available, you can update to it
from  the |percona| software repositories:

|debian| or |ubuntu|
   .. code-block:: bash

      $ sudo apt-get update && sudo apt-get install pmm-client

|red-hat| or |centos|
   .. code-block:: bash

      $ yum update pmm-client

If you installed your |abbr.pmm| client manually, :ref:`remove it
<deploy-pmm.removing>` and then :ref:`download and install a newer version
<deploy-pmm.client.installing>`.

.. _deploy-pmm.removing:

Uninstalling |pmm| Components
================================================================================

Each |pmm-client| and the |pmm-server| are removed separately. First, remove all
monitored services by using the |pmm-admin.remove| command (see
:ref:`pmm-admin.rm`). Then you can remove each |pmm-client| and the
|pmm-server|.

.. _deploy.pmm-client.removing:

Removing the |pmm-client|
--------------------------------------------------------------------------------

Remove all monitored instances as described in :ref:`pmm-admin.rm`. Then,
uninstall the |pmm-admin| package. The exact procedure of removing the
|pmm-client| depends on the method of installation.

|tip.run-all.root|

Using YUM
   .. include:: .res/code/sh.org
      :start-after: +yum.remove.pmm-client+
      :end-before: #+end-block
		  
Using APT
   .. include:: .res/code/sh.org
      :start-after: +apt-get.remove.pmm-client+
      :end-before: #+end-block
		  
Manually installed RPM package
   .. include:: .res/code/sh.org
      :start-after: +rpm.e.pmm-client+
      :end-before: #+end-block

Manually installed DEB package
   .. include:: .res/code/sh.org
      :start-after: +dpkg.r.pmm-client+
      :end-before: #+end-block

Using the generic |pmm-client| tarball.
  |cd| into the directory where you extracted the tarball
  contents. Then, run the :file:`unistall` script:
  
  .. include:: .res/code/sh.org
     :start-after: +uninstall+
     :end-before: #+end-block

Removing the |pmm-server|
--------------------------------------------------------------------------------

If you run your |pmm-server| using |docker|, stop the container as follows:

.. include:: .res/code/sh.org
   :start-after: +docker.stop.pmm-server&docker.rm.pmm-server+
   :end-before: #+end-block

To discard all collected data (if you do not plan to use
|pmm-server| in the future), remove the ``pmm-data``
container:

.. include:: .res/code/sh.org
   :start-after: +docker.rm.pmm-data+
   :end-before: #+end-block

If you run your |pmm-server| using a virtual appliance, just stop and
remove it.

To terminate the |pmm-server| running from an |amazon| machine image, run
the following command in your terminal:

.. include:: .res/code/sh.org
   :start-after: +aws.ec2.terminate-instances+
   :end-before: #+end-block

.. seealso::

   |pmm| Building Blocks
      :ref:`pmm/architecture`
   About using the |pmm-admin.add| command
      :ref:`pmm-admin.add`

.. include:: .res/replace.txt
