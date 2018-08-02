:orphan: true

.. _install-client-yum:

Installing the |pmm-client| Package on |red-hat| and |centos|
================================================================================

If you are running an RPM-based |linux| distribution, use the |yum| package
manager to install |pmm-client| from the official Percona software repository.

Percona provides :file:`.rpm` packages for 64-bit versions
of Red Hat Enterprise Linux 6 (Santiago) and 7 (Maipo),
including its derivatives that claim full binary compatibility,
such as, CentOS, Oracle Linux, Amazon Linux AMI, and so on.

.. note::

   |pmm-client| should work on other RPM-based distributions,
   but it is tested only on RHEL and CentOS versions 6 and 7.

To install the |pmm-client| package, complete the following procedure. |tip.run-all.root|:

1. Configure Percona repositories as described in
   `Percona Software Repositories Documentation
   <https://www.percona.com/doc/percona-repo-config/index.html>`_.

#. Install the ``pmm-client`` package:


   .. include:: ../../.res/code/yum.install.pmm-client.txt

.. seealso::

   What other installation methods exist for |pmm-client|?
      :ref:`deploy-pmm.client.installing`

   Next steps: Connecting to |pmm-server|
      :ref:`deploy-pmm.client_server.connecting`
   
.. include:: ../../.res/replace.txt
