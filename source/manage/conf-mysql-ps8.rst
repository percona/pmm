.. _pmm.conf-mysql.8-0:

#############################
Configuring MySQL 8.0 for PMM
#############################

MySQL 8 (in version 8.0.4) changes the way clients are authenticated by
default. The ``default_authentication_plugin`` parameter is set to
``caching_sha2_password``. This change of the default value implies that MySQL
drivers must support the SHA-256 authentication. Also, the communication channel
with MySQL 8 must be encrypted when using ``caching_sha2_password``.

The MySQL driver used with PMM does not yet support the SHA-256 authentication.

With currently supported versions of MySQL, PMM requires that a dedicated MySQL
user be set up. This MySQL user should be authenticated using the
``mysql_native_password`` plugin.  Although MySQL is configured to support SSL
clients, connections to MySQL Server are not encrypted.

There are two workarounds to be able to add MySQL Server version 8.0.4
or higher as a monitoring service to PMM:

1. Alter the MySQL user that you plan to use with PMM
2. Change the global MySQL configuration

.. rubric:: Altering the MySQL User

Provided you have already created the MySQL user that you plan to use
with PMM, alter this user as follows:

.. code-block:: sql

   mysql> ALTER USER pmm@'localhost' IDENTIFIED WITH mysql_native_password BY '$eCR8Tp@s$w*rD';

Then, pass this user to ``pmm-admin add`` as the value of the ``--username``
parameter.

This is a preferred approach as it only weakens the security of one user.

.. rubric:: Changing the global MySQL Configuration

A less secure approach is to set ``default_authentication_plugin``
to the value **mysql_native_password** before adding it as a
monitoring service. Then, restart your MySQL Server to apply this
change.

.. code-block:: sql

   [mysqld]
   default_authentication_plugin=mysql_native_password

.. seealso::

   Creating a MySQL User for PMM
      :ref:`privileges`

   MySQL Server Blog: MySQL 8.0.4 : New Default Authentication Plugin : caching_sha2_password
      https://mysqlserverteam.com/mysql-8-0-4-new-default-authentication-plugin-caching_sha2_password/

   MySQL Documentation: Authentication Plugins
      https://dev.mysql.com/doc/refman/8.0/en/authentication-plugins.html

   MySQL Documentation: Native Pluggable Authentication
      https://dev.mysql.com/doc/refman/8.0/en/native-pluggable-authentication.html
