.. include:: /.res/replace.txt

.. _perf-disable-table-stats:
.. _performance-issues:

################################################################################
Improving |pmm| Performance with Table Statistics Options
################################################################################

If a |mysql| instance has a lot of schemas or tables,
there are two options to help improve the performance of |pmm|
when adding instances with |pmm-admin.add|:
|opt.dis-tablestats| and |opt.dis-tablestats-limit|.

.. important::

   - These settings can only be used when adding an instance.
     To change them, you must remove and re-add the instances.

   - You can only use one of these options when adding an instance.

.. contents::
   :local:
   :depth: 1

***********************************************************************************************
`Disable per-table statistics for an instance <pmm.conf.mysql.perf.metrics.tablestats>`_
***********************************************************************************************

When adding an instance with |pmm-admin.add|,
the |opt.dis-tablestats| option
disables table statistics collection
when there are more than the default number (1000) of tables in the instance.

=====
USAGE
=====

.. code-block:: sh

   sudo pmm-admin add mysql --disable-tablestats

******************************************************************************************************************************
`Change the number of tables beyond which per-table statistics is disabled <pmm.conf.mysql.perf.metrics.tablestats.limit>`_
******************************************************************************************************************************

When adding an instance with |pmm-admin.add|,
the |opt.dis-tablestats-limit| option
changes the number of tables (from the default of 1000)
beyond which per-table statistics collection is disabled.

=====
USAGE
=====

.. code-block:: sh

   sudo pmm-admin add mysql --disable-tablestats-limit=<LIMIT>

=======
EXAMPLE
=======

Add a |mysql| instance,
disabling per-table statistics collection
when the number of tables in the instance reaches 2000.

.. code-block:: sh

   sudo pmm-admin add mysql --disable-tablestats-limit=2000
