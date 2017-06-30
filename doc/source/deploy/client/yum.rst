.. _install-client-yum:

===========================================
Installing PMM Client on Red Hat and CentOS
===========================================

If you are running an RPM-based Linux distribution,
use the :command:`yum` package manager
to install *PMM Client* from the official Percona software repository.

Percona provides :file:`.rpm` packages for 64-bit versions
of Red Hat Enterprise Linux 6 (Santiago) and 7 (Maipo),
including its derivatives that claim full binary compatibility,
such as, CentOS, Oracle Linux, Amazon Linux AMI, and so on.

.. note:: *PMM Client* should work on other RPM-based distributions,
   but it is tested only on RHEL and CentOS versions 6 and 7.

To install *PMM Client*:

1. Configure Percona repositories as described in
   `Percona Software Repositories Documentation
   <https://www.percona.com/doc/percona-repo-config/index.html>`_.

#. Install the ``pmm-client`` package:

   .. code-block:: bash

      sudo yum install pmm-client

Next Steps
==========

After you install *PMM Client*,
:ref:`connect it to PMM Server <connect-client>`.

