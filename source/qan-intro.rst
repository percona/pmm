.. _pmm.qan:

--------------------------------------------------------------------------------
Introduction
--------------------------------------------------------------------------------

.. raw:: html

	<a
	  id="another-doc-version-link"
	  data-location="https://www.percona.com/doc/percona-monitoring-and-management/qan.html"
	  href="https://www.percona.com/doc/percona-monitoring-and-management/2.x/qan-intro.html"
	  style="display:none;"
	></a>

The QAN is a special dashboard which enables database administrators and
application developers to analyze database queries over periods of time and find performance problems. QAN helps you optimize database
performance by making sure that queries are executed as expected and within the
shortest time possible.  In case of problems, you can see which queries may be
the cause and get detailed metrics for them.

.. image:: /_images/PMM_Query_Analytics.jpg

.. important::
   
   PMM Query Analytics supports MySQL and MongoDB. The minimum requirements
   for MySQL are:

   * MySQL 5.1 or later (if using the slow query log)
   * MySQL 5.6.9 or later (if using Performance Schema)
 
QAN displays its metrics in both visual and numeric form: the performance
related characteristics appear as plotted graphics with summaries.
