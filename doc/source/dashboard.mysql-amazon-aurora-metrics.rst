.. _dashboard.mysql-amazon-aurora-metrics:

|mysql| |amazon-aurora| Metrics
================================================================================

This dashboard provides metrics for analyzing |amazon-aurora| instances.

.. contents::
   :local:

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-transaction-commits:

|amazon-aurora| Transaction Commits
--------------------------------------------------------------------------------

This graph shows number of commits which the |amazon-aurora| engine performed as
well as the average commit latency. Graph Latency does not always correlates
with number of commits performed and can quite high in certain situations.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-load:

Amazon Aurora Load
--------------------------------------------------------------------------------

This graph shows us what statements contribute most load on the system as well
as what load corresponds to |amazon-aurora| transaction commit.

.. _dashboard.mysql-amazon-aurora-metrics.aurora-memory-used:

Aurora Memory Used
--------------------------------------------------------------------------------

This graph shows how much memory is used by |amazon-aurora| lock manager as well
as amount of memory used by |amazon-aurora| to store Data Dictionary.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-statement-latency:

Amazon Aurora Statement Latency
--------------------------------------------------------------------------------

This graph shows average latency for most important types of statements. Latency
spikes are often indicative of the instance overload.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-special-command-counters:

Amazon Aurora Special Command Counters
--------------------------------------------------------------------------------

|amazon-aurora| |mysql| allows a number of commands which are not available from
standard |mysql|. This graph shows usage of such commands. Regular
:code:`unit_test` calls can be seen in default |amazon-aurora| install, the rest
will depend on your workload.

.. _dashboard.mysql-amazon-aurora-metrics.amazon-aurora-problems:

Amazon Aurora Problems
--------------------------------------------------------------------------------

This metric shows different kinds of internal |amazon-aurora| |mysql| problems
which should be zero in case of normal operation.

.. include:: .res/replace/name.txt
