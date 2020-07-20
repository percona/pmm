.. _pmm-admin.remove:
.. _pmm-admin.rm:

`Removing monitoring services with pmm-admin remove <pmm-admin.remove>`_
================================================================================

Use the ``pmm-admin remove`` command to remove monitoring services.

.. rubric:: USAGE

Run this command as root or by using the ``sudo`` command

.. _pmm-admin.remove.options.service:

.. include:: ../.res/code/pmm-admin.rm.options.service.txt
		
When you remove a service,
collected data remains in Metrics Monitor on PMM Server.

.. only:: showhidden

	To remove the collected data, use the **pmm-admin purge** command.

.. _pmm-admin.remove.services:

.. rubric:: SERVICES

Service type can be `mysql`, `mongodb`, `postgresql` or `proxysql`, and service
name is a monitoring service alias. To see which services are enabled,
run **pmm-admin list**.

.. _pmm-admin.remove.examples:

.. rubric:: EXAMPLES

* Removing MySQL service named "mysql-sl":

  .. code-block:: bash

     # pmm-admin remove mysql mysql-sl
     Service removed. 
		   
* To remove *MongoDB* service named "mongo":

  .. code-block:: bash

     # pmm-admin remove mongodb mongo
     Service removed.

* To remove *PostgreSQL* service named "postgres":

  .. code-block:: bash

     # pmm-admin remove postgresql postgres
     Service removed.

* To remove *ProxySQL* service named "ubuntu-proxysql":

  .. code-block:: bash

     # pmm-admin remove proxysql ubuntu-proxysql
     Service removed.
		
For more information, run ``pmm-admin remove --help``.


