.. _install-client:

=====================
Installing PMM Client
=====================

*PMM Client* is a package of agents and exporters
installed on a MySQL or MongoDB host that you want to monitor.
The components collect various data
about general system and database performance,
and send this data to corresponding *PMM Server* components.

Before installing the *PMM Client* package on a database host,
make sure that your *PMM Server* host is accessible.
For example, you can ``ping 192.168.100.1``
or whatever IP address *PMM Server* is running on.

You will need to have root access on the database host
where you will be installing *PMM Client*
(either logged in as a user with root privileges
or be able to run commands with ``sudo``).

The minimum requirements for Query Analytics (QAN) are:

* MySQL 5.1 or later (if using the slow query log)
* MySQL 5.6.9 or later (if using Performance Schema)

.. note:: You should not install agents on database servers
   that have the same host name,
   because host names are used by *PMM Server* to identify collected data.

*PMM Client* should run on any modern Linux distribution,
however Percona provides PMM Client packages for automatic installation
from software repositories only on the most popular Linux distributions:

* :ref:`Install PMM Client on Debian or Ubuntu <install-client-apt>`

* :ref:`Install PMM Client on Red Hat or CentOS <install-client-yum>`

If you are not able to install from Percona's software repositories or
running some other Linux distribution,
try :ref:`install-client-manual`.

.. toctree::
   :hidden:

   apt
   yum
   manual
   remove
   upgrade

