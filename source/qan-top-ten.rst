.. _pmm.qan-top-ten:

--------------------------------------------------------------------------------
`Understanding Top 10 <pmm.qan-top-ten>`_
--------------------------------------------------------------------------------

.. raw:: html

	<a
	  id="another-doc-version-link"
	  data-location="https://www.percona.com/doc/percona-monitoring-and-management/qan.html#pmm-qan-home-page-opening"
	  href="https://www.percona.com/doc/percona-monitoring-and-management/2.x/qan-top-ten.html"
	  style="display:none;"
	></a>

By default, |qan| shows the top *ten* queries. You can sort queries by any
column - just click the small arrow to the right of the column name.
Also you can add a column for each additional field which is exposed by the
data source by clicking the ``+`` sign on the right edge of the header and
typing or selecting from the available list of fields.

.. figure:: .res/graphics/png/qan.query-summary-table.default.1.png

   The query summary table shows the monitored queries from the selected
   database.

To view more queries, use buttons below the query summary table.

.. _pmm.qan.query.selecting:

`Query Detail Section <pmm.qan.query.selecting>`_
--------------------------------------------------------------------------------
   
In addition to the metrics in the :ref:`query metrics summary table <Query-Metrics-Summary-Table>`,
:program:`QAN` displays more information about the query itself below the table.

.. figure:: .res/graphics/png/qan.query.1.png

   The Query Detail section shows the SQL statement for the selected query.

*Tables* tab for the selected query seen in the same section contains the table
definition and indexes for each table referenced by the query:

.. figure:: .res/graphics/png/qan.query.2.png

   Tables and indexes details the selected query.

.. include:: .res/replace.txt
