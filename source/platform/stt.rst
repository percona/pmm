.. _platform.stt:
             
################################################################################
Security Threat Tool
################################################################################

The Security Threat Toll runs regular checks against connected databases,
alerting you if any servers pose a potential security threat.

The checks are automatically downloaded from Percona Platform
and run every 24 hours. (This period is not configurable.)

They run on the PMM Client side with the results passed to PMM Server
for display in the :guilabel:`Failed security checks` summary dashboard
and the :guilabel:`PMM Database Checks` details dashboard.

.. important::

   Check results data *always* remains on the PMM Server, and is not to be
   confused with anonymous data sent for :ref:`server-admin-gui-telemetry` purposes.
  
********************************************************************************
Where to see the results of checks
********************************************************************************

On your PMM home page, the :guilabel:`Failed security checks` dashboard
shows a count of the number of failed checks.

.. figure:: /.res/graphics/png/pmm.failed-checks.png

   Failed Checks summary dashboard

More details can be seen by opening the :guilabel:`Failed Checks` dashboard
using :menuselection:`PMM --> PMM Database Checks`.

.. figure:: /.res/graphics/png/pmm.database-checks.failed-checks.png

   Failed Checks details dashboard

.. note::

   After :ref:`activating <server-admin-gui-stt>` the Security Threat Tool, you must wait 24 hours
   for data to appear in the dashboard.

********************************************************************************
How to enable the Security Threat Tool
********************************************************************************

The Security Threat Tool is disabled by default. It can be enabled in
:menuselection:`PMM --> PMM Settings`
(see :ref:`server-admin-gui-pmm-settings-page`).

.. figure:: /.res/graphics/png/pmm.failed-checks.failed-security-checks-off.png

   Failed security checks summary dashboard when checks are disabled

.. figure:: /.res/graphics/png/pmm.failed-checks.failed-database-checks.png

   Failed database checks dashboard when disabled
   
********************************************************************************
Checks made by the Security Threat Tool
********************************************************************************

.. The range of checks can be classified as

.. - :ref:`Generic <stt-generic-checks>`, affecting all database types;
.. - :ref:`Specific <stt-specific-checks>`, specific to a particular vendor.

.. .. _stt-generic-checks:

..
   ================================================================================
   Generic checks
   ================================================================================

   +------------------------------+-----------------------------------------------+
   | Check                        | Description                                   |
   +==============================+===============================================+
   | Latest version               | Check server software is the latest version.  |
   +------------------------------+-----------------------------------------------+
   | CVE                          | Check whether any CVEs are assigned to the    |
   |                              | software.                                     |
   +------------------------------+-----------------------------------------------+
   | Password                     | Check for empty/blank passwords or default    |
   |                              | passwords.                                    |
   +------------------------------+-----------------------------------------------+


.. .. _stt-specific-checks:

..
   ================================================================================
   Database-specific checks
   ================================================================================

+------------------------------+-----------------------------------------------+
| Name                         | Description                                   |                                
+==============================+===============================================+
| ``mongodb_auth``             | This check returns a warning if MongoDB       |
|                              | authentication is disabled.                   |
+------------------------------+-----------------------------------------------+
| ``mongodb_version``          | Warn if MongoDB/PSMDB version is not the      |
|                              | latest.                                       |
+------------------------------+-----------------------------------------------+
| ``mysql_empty_password``     | Warn if there are users without passwords.    |
+------------------------------+-----------------------------------------------+
| ``mysql_version``            | Warn if MySQL/PS/MariaDB version is not the   |
|                              | latest.                                       |
+------------------------------+-----------------------------------------------+
| ``postgresql_version``       | Warn if PostgreSQL version is not the latest. |
+------------------------------+-----------------------------------------------+
