.. _pmm/release/1-4-0:

|pmm.name| |release|
********************************************************************************

:Date: October 20, 2017

|percona| announces the release of |pmm.name| |release|.

This release introduces the support of external |prometheus| exporters so that you
can create dashboards in the Metrics monitor even for the monitoring services
other than those provided with PMM client packages. To attach an existing
external |prometheus| exporter, run :command:`pmm-admin add external:metrics NAME_OF_EXPORTER
URL:PORT`.

.. code-block:: js
   :emphasize-lines: 11
   :caption: Example of JSON output of the ``pmm-admin list --json`` command. It also contains an external monitoring service in the value of the `ExternalServices` element.
		
   {"Version":"1.4.0",
    "ServerAddress":"127.0.0.1:80",
    "ServerSecurity":"",
    "ClientName":"percona",
    "ClientAddress":"172.17.0.1",
    "ClientBindAddress":"",
    "Platform":"linux-systemd",
    "Err":"",
    "Services":[{"Type":"linux:metrics","Name":"percona","Port":"42000","Running":true,"DSN":"-","Options":"","SSL":"","Password":""}],
    "ExternalErr":"",
    "ExternalServices":[{"JobName":"postgres","ScrapeInterval":1000000000,"ScrapeTimeout":1000000000,"MetricsPath":"/metrics","Scheme":"http","StaticTargets":["127.0.0.1:5432"]}]}

The list of attached monitoring services is now available not only in the
tabular format but also as a JSON file to enable automatic verification of your
configuration. To view the list of monitoring services in the JSON format run
|pmm-admin.list| |opt.json|.

In this release, |prometheus| and |grafana| have
been upgraded. |prometheus| version 1.7.2, shipped with this
release, offers a number of bug fixes that will contribute to its
smooth operation inside PMM. For more information,
see `the Prometheus change log <https://github.com/prometheus/prometheus/blob/v1.7.2/CHANGELOG.md#172--2017-09-26>`_.

Version 4.5.2 of |grafana|, included in this release of PMM, offers a
number of new tools that will facilitate data analysis in PMM:

- New ``query editor`` for |prometheus| expressions which features syntax highlighting
  and autocompletion for metrics, functions and range vectors.
  
- ``Query inspector`` which provides detailed information about the query. The
  primary goal of graph inspector is to enable analyzing a graph which does not
  display data as expected.
  
The complete list of new features in :program:`Grafana` 4.5.0 is
available from `What's New in Grafana v4.5
<http://docs.grafana.org/guides/whats-new-in-v4-5/>`_.

For install and upgrade instructions, see :ref:`deploy-pmm`.

.. rubric:: New features

- :pmmbug:`1520`: Prometheus upgraded to version 1.7.2.
- :pmmbug:`1521`: Grafana upgraded to version 4.5.2.
- :pmmbug:`1091`: The :command:`pmm-admin list` produces a JSON document as
  output if the :option:`--json` option is supplied.
- :pmmbug:`507`: :ref:`External exporters are supported <pmm/pmm-admin/external-monitoring-service.adding>` with |pmm-admin|.
- :pmmbug:`1622`: :program:`docker` images of PMM Server are
  `available for downloading
  <https://www.percona.com/downloads/pmm/>`_ as :program:`tar`
  packages.

.. rubric:: Bug fixes

- :pmmbug:`1172`: In some cases, the ``TABLES`` section of a query in
  |qan| could contain no data and display the *List of tables is empty*
  error. The ``Query`` and ``Explain`` sections had the relevant values.
- :pmmbug:`1519`: A Prometheus instance could be forced to shut down if it
  contained too many targets (more than 50).  When started the next time,
  Prometheus initiated a time consuming crash recovery routine which took long
  on large installations.

.. |release| replace:: 1.4.0

.. include:: .res/replace.txt

