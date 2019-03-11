.. _install-client-apt:

Installing DEB packages using apt-get
================================================================================

If you are running a DEB-based |linux| distribution, use the |apt| package
manager to install |pmm-client| from the official Percona software repository.

|percona| provides :file:`.deb` packages for 64-bit versions of the following
distributions:

.. include:: ../.res/contents/list.pmm-client.supported-apt-platform.txt

.. note::

   |pmm-client| should work on other DEB-based distributions, but it is tested
   only on the platforms listed above.

To install the |pmm-client| package, complete the following
procedure. |tip.run-all.root|:

1. Configure |percona| repositories as described in `Percona Software
   Repositories Documentation
   <https://www.percona.com/doc/percona-repo-config/index.html>`_.

#. Install the |pmm-client| package:

   .. include:: ../.res/code/apt-get.install.pmm-client.txt

.. seealso::

   What other installation methods exist for |pmm-client|?
      :ref:`deploy-pmm.client.installing`

.. include:: ../.res/replace.txt
