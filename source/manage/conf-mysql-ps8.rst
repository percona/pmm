`Configuring MySQL 8.0 for PMM <pmm.conf-mysql.8-0>`_
=========================================================

|mysql| 8 (in version 8.0.4) changes the way clients are authenticated by
default. The |opt.default-authentication-plugin| parameter is set to
``caching_sha2_password``. This change of the default value implies that |mysql|
drivers must support the SHA-256 authentication. Also, the communication channel
with |mysql| 8 must be encrypted when using ``caching_sha2_password``.

The |mysql| driver used with |pmm| does not yet support the SHA-256 authentication.

With currently supported versions of |mysql|, |pmm| requires that a dedicated |mysql|
user be set up. This |mysql| user should be authenticated using the
``mysql_native_password`` plugin.  Although |mysql| is configured to support SSL
clients, connections to |mysql| Server are not encrypted.

There are two workarounds to be able to add |mysql| Server version 8.0.4
or higher as a monitoring service to |pmm|:

1. Alter the |mysql| user that you plan to use with |pmm|
2. Change the global |mysql| configuration

.. rubric:: Altering the |mysql| User

Provided you have already created the |mysql| user that you plan to use
with |pmm|, alter this user as follows:

.. include:: .res/code/alter.user.identified.with.by.txt

Then, pass this user to ``pmm-admin add`` as the value of the ``--username``
parameter.

This is a preferred approach as it only weakens the security of one user.

.. rubric:: Changing the global |mysql| Configuration

A less secure approach is to set |opt.default-authentication-plugin|
to the value **mysql_native_password** before adding it as a
monitoring service. Then, restart your |mysql| Server to apply this
change.

.. include:: .res/code/my-conf.mysqld.default-authentication-plugin.txt
   
.. seealso::

   Creating a |mysql| User for |pmm|
      :ref:`privileges`

   More information about adding the |mysql| query analytics monitoring service
      :ref:`pmm-admin.add-mysql-queries`

   |mysql| Server Blog: |mysql| 8.0.4 : New Default Authentication Plugin : caching_sha2_password
      https://mysqlserverteam.com/mysql-8-0-4-new-default-authentication-plugin-caching_sha2_password/

   |mysql| Documentation: Authentication Plugins
      https://dev.mysql.com/doc/refman/8.0/en/authentication-plugins.html

   |mysql| Documentation: Native Pluggable Authentication
      https://dev.mysql.com/doc/refman/8.0/en/native-pluggable-authentication.html

.. include:: ../.res/replace.txt
