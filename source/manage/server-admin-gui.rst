--------------------------------------------------------------------------------
PMM Settings Page
--------------------------------------------------------------------------------

|pmm-settings| is a special page dedicated to configuring a number of PMM
options. It can be accessed through the main menu:

   .. figure:: ../.res/graphics/png/pmm-add-instance.png

      Choosing the |pmm| *Settings* menu entry

|pmm-settings| page consists of the following sections, which allow to configure
different aspects of the PMM Server:

.. contents::
   :local:
   :depth: 1

Settings
================================================================================

*Settings* section allows you to change `metrics resolution <https://www.percona.com/doc/percona-monitoring-and-management/2.x/faq.html#what-resolution-is-used-for-metrics>`_, `data retention <https://www.percona.com/doc/percona-monitoring-and-management/2.x/faq.html#how-to-control-data-retention-for-pmm>`_,
as well as configure `telemetry <https://www.percona.com/doc/percona-monitoring-and-management/2.x/glossary-terminology.html#telemetry>`_ and automatic checking for `updates <https://www.percona.com/doc/percona-monitoring-and-management/2.x/glossary-terminology.html#PMM-Version>`_:

   .. figure:: ../.res/graphics/png/pmm.settings_settings.png

      Settings options

Don't forget to click the *Apply changes* button to make changed options work.

SSH Key Details
==========================================================================================================

This section allows you to upload your public SSH key which can be used to
access the PMM Server via SSH (e.g. the `PMM Server deployed as a virtual appliance <virtual-appliance.html#pmm-deploying-server-virtual-appliance-accessing>`_).

   .. figure:: ../.res/graphics/png/pmm.settings_ssh_key.png

      Submitting the public key

Submit your **public key** in the *SSH Key* field and click the
*Apply SSH Key* button.

AlertManager integration
================================================================================

This section allows you to configure `integration of Prometheus with an external Alertmanager <https://www.percona.com/doc/percona-monitoring-and-management/2.x/faq.html#how-to-integrate-alertmanager-with-pmm>`_. 

* The **Alertmanager URL** field should contain the URL of the Alertmanager
  which would serve your PMM alerts.
* The **Alertmanager rules** field is used to specify alerting rules in the YAML
  configuration format.

   .. figure:: ../.res/graphics/png/pmm.settings_alertmanager.png

      Configuring the Alertmanager integration

Fill both fields and click the *Apply Alertmanager settings* button to proceed.

Diagnostics
================================================================================

|pmm| can generate a set of diagnostics data which can be examined
and/or shared with Percona Support in case of some issue to solve it faster.
You can `get collected logs from PMM Server <https://www.percona.com/doc/percona-monitoring-and-management/2.x/faq.html#id13>`_ by clicking
the **Download PMM Server Logs** button.

   .. figure:: ../.res/graphics/png/pmm.settings_iagnostics.png

      Downloading the PMM Server logs

.. include:: ../.res/replace.txt
