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

1. If your system does not already have
   Percona's ``yum`` repository configured,
   run the following command:

   .. code-block:: bash

      sudo yum install http://www.percona.com/downloads/percona-release/redhat/0.1-4/percona-release-0.1-4.noarch.rpm

#. Install the ``pmm-client`` package:

   .. code-block:: bash

      sudo yum install pmm-client

.. _yum-testing-repo:

Testing and Experimental Repositories
=====================================

Percona offers pre-release builds from the testing repo,
and early-stage development builds from the experimental repo.
You can enable either one in the Percona repository configuration file
:file:`/etc/yum.repos.d/percona-release.repo`.
There are three sections in this file,
for configuring corresponding repositories:

* stable release
* testing
* experimental

The latter two repositories are disabled by default.

If you want to install the latest testing builds,
set ``enabled=1`` for the following entries: ::

  [percona-testing-$basearch]
  [percona-testing-noarch]

If you want to install the latest experimental builds,
set ``enabled=1`` for the following entries: ::

  [percona-experimental-$basearch]
  [percona-experimental-noarch]

Next Steps
==========

After you install *PMM Client*,
:ref:`connect it to PMM Server <connect-client>`.

