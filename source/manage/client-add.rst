.. _pmm-admin.add:

:ref:`Adding monitoring services <pmm-admin.add>`
================================================================================

Use the |pmm-admin.add| command to add monitoring services.

.. _pmm-admin.add.usage:

.. rubric:: USAGE

.. code-block:: bash

   $ pmm-admin add [OPTIONS] [SERVICE]

When you add a monitoring service |pmm-admin| automatically creates
and sets up a service in the operating system. You can tweak the
|systemd| configuration file and change its behavior.
   
For example, you may need to disable the HTTPS protocol for the
|prometheus| exporter associated with the given service. To accomplish this
task, you need to remove all SSL related options.

|tip.run-all.root|:

1. Open the |systemd| unit file associated with the
   monitoring service that you need to change, such as
   |pmm-mysql-metrics.service|.

   .. include:: ../.res/code/cat.etc-systemd-system-pmm-mysql-metrics.txt
   
#. Remove the SSL related configuration options (key, cert) from the
   |systemd| unit file or `init.d` startup
   script. :ref:`sample.systemd` highlights the SSL related options in
   the |systemd| unit file.

   The following code demonstrates how you can remove the options
   using the |sed| command. (If you need more information about how
   |sed| works, see the documentation of your system).
   
   .. include:: ../.res/code/sed.e.web-ssl.pmm-mysql-metrics-service.txt
   
#. Reload |systemd|:

   .. include:: ../.res/code/systemctl.daemon-reload.txt

#. Restart the monitoring service by using |pmm-admin.restart|:

   .. include:: ../.res/code/pmm-admin.restart.mysql-metrics.txt

.. _pmm-admin.add-options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.add| command:

|opt.dev-enable|
  Enable experimental features.

|opt.disable-ssl|
  Disable (otherwise enabled) SSL for the connection between |pmm-client| and
  |pmm-server|. Turning off SSL encryption for the data acquired from some
  objects of monitoring allows to decrease the overhead for a |pmm-server|
  connected with a lot of nodes.

|opt.service-port|

  Specify the :ref:`service port <service-port>`.

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`.

.. _pmm-admin.add.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`,
along with any relevant additional arguments.

For more information, run
|pmm-admin.add|
|opt.help|.


.. include:: ../.res/replace.txt
