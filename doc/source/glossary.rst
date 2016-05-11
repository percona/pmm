==========
 Glossary
==========

.. glossary::

   Metrics Monitor (MM)
    Component of :term:`PMM Server` that provides a historical view of metrics
    critical to a MySQL server instance.

   PMM Client
    Collects MySQL server metrics, general system metrics,
    and query analytics data for a complete performance overview.
    Collected data is sent to :term:`PMM Server`.

    For more information, see :ref:`architecture`.

   PMM Server
    Aggregates data collected by :term:`PMM Client`
    and presents it in the form of tables, dashboards,
    and graphs in a web interface.

    For more information, see :ref:`architecture`.

   Query Analytics (QAN)
    Component of :term:`PMM Server` that enables you to analyze
    MySQL query performance over periods of time.
