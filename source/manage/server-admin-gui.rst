.. _server-admin-gui-pmm-settings-page:

#################
PMM Settings Page
#################

The *PMM Settings* page lets you configure a number of PMM options. It can be accessed through the main menu:

.. image:: /.res/graphics/png/pmm-add-instance.png

********
Settings
********

The *Settings* section allows you to change :ref:`metrics resolution <metrics-resolution>`, :ref:`data retention <data-retention>`, as well as configure telemetry and automatic checking for updates:

.. image:: /.res/graphics/png/pmm.settings_settings.png

Press *Apply changes* to store any changes.

.. _server-admin-gui-metrics-resolution:

******************
Metrics resolution
******************

PMM collects metrics at three different time intervals.

Short time intervals are regarded as high resolution metrics, while those at longer time intervals are low resolution.

The :guilabel:`Metrics Resolution` slider lets you choose from three preset combinations of intervals corresponding to high, medium, and low resolution (short, medium, and long collection periods).

The presets are also labeled :guilabel:`Low`, :guilabel:`Medium`, and :guilabel:`High`.

The slider tool-tip shows the collection time corresponding to each resolution.

- Setting the slider to :guilabel:`Low` increases the time between collection, resulting in low-resolution metrics.

- Setting the slider to :guilabel:`High` decreases the time between collection, resulting in high-resolution metrics.

.. _server-admin-gui-telemetry:

*********
Telemetry
*********

The *Telemetry* switch enables gathering and sending basic **anonymous** data to Percona, which helps us to determine where to focus the development and what is the uptake of the various versions of PMM. Specifically, gathering this information helps determine if we need to release patches to legacy versions beyond support, determining when supporting a particular version is no longer necessary, and even understanding how the frequency of release encourages or deters adoption.

Currently, only the following information is gathered:

* PMM Version,
* Installation Method (Docker, AMI, OVF),
* the Server Uptime.

We do not gather anything that would make the system identifiable, but the following two things are to be mentioned:

1. The Country Code is evaluated from the submitting IP address before it is discarded.

2. We do create an “instance ID” - a random string generated using UUID v4.  This instance ID is generated to distinguish new instances from existing ones, for figuring out instance upgrades.

.. note:: The first telemetry reporting of a new PMM Server instance is delayed by 24 hours to allow sufficient time to disable the service for those that do not wish to share any information.

There is a landing page for this service, available at `check.percona.com <https://check.percona.com>`_, which clearly explains what this service is, what it’s collecting, and how you can turn it off.

.. note:: The `Grafana internal reporting feature <https://grafana.com/docs/grafana/latest/installation/configuration/#reporting-enabled>`_ is currently **not** managed by PMM. If you want to turn it, you need to go inside the PMM Server container and `change configuration <https://grafana.com/docs/grafana/latest/installation/configuration/#reporting-enabled>`_ after each update.

.. note:: Beside using *PMM Settings* page, you can also disable Telemetry with the ``-e DISABLE_TELEMETRY=1`` option in your docker run statement for the PMM Server.

.. important::

   1. If the Security Threat Tool is enabled in PMM Settings, Telemetry is automatically enabled.
   2. Telemetry is sent immediately; the 24-hour grace period is not honored.

.. _server-admin-gui-check-for-updates:      

*****************
Check for updates
*****************

When active, PMM will automatically check for updates and put a notification in the :guilabel:`Updates` dashboard if any are available.

.. _server-admin-gui-stt:

********************
Security Threat Tool
********************

The Security Threat Tool performs a range of security-related checks on a registered instance and reports the findings.

It is disabled by default.

It can be enabled in :menuselection:`PMM --> PMM Settings --> Settings --> Advanced Settings --> Security Threat Tool`.

The checks take 24 hours to complete.

The results can be viewed in :menuselection:`PMM --> PMM Database Checks`.

.. seealso:: :ref:`Security Threat Tool main page <platform.stt>`

***************          
SSH Key Details
***************

This section lets you upload your public SSH key to access the PMM Server via SSH (for example, when accessing PMM Server as a :ref:`virtual appliance <pmm.deploying.server.virtual>`).

.. image:: /.res/graphics/png/pmm.settings_ssh_key.png

Enter your **public key** in the *SSH Key* field and click *Apply SSH Key*.

.. _prometheus-alertmanager-integration:

***********************************
Prometheus Alertmanager integration
***********************************

The Prometheus Alertmanager manages alerts from Prometheus, deduplicating, grouping, and routing them to the appropriate receiver or display component.

This section lets you configure integration of Prometheus with an external Alertmanager. 

* The **Alertmanager URL** field should contain the URL of the Alertmanager which would serve your PMM alerts.

* The **Prometheus Alerting rules** field is used to specify alerting rules in the YAML configuration format.

.. image:: /.res/graphics/png/pmm.settings_alertmanager.png

Fill both fields and click the *Apply Alertmanager settings* button to proceed.

.. seealso::

   - `Prometheus Alertmanager documentation <https://prometheus.io/docs/alerting/alertmanager/>`_
   - `Prometheus Alertmanager alerting rules <https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/>`_

***********
Diagnostics
***********

PMM can generate a set of diagnostics data which can be examined and/or shared with Percona Support in case of some issue to solve it faster.  You can get collected logs from PMM Server 
by clicking the **Download PMM Server Logs** button.

.. image:: /.res/graphics/png/pmm.settings_iagnostics.png

.. seealso:: :ref:`troubleshoot-connection`
