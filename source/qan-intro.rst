.. _pmm.qan:

--------------------------------------------------------------------------------
Introduction
--------------------------------------------------------------------------------

The |qan| is a special dashboard which enables database administrators and
application developers to |qan.what-is|. |qan| helps you optimize database
performance by making sure that queries are executed as expected and within the
shortest time possible.  In case of problems, you can see which queries may be
the cause and get detailed metrics for them.

.. figure:: .res/graphics/png/qan01.png
	    
   |qan| helps analyze database queries over periods of time and find
   performance problems.

.. important::
   
   |qan.name| supports |mysql| and |mongodb|. The minimum requirements
   for |mysql| are:

   * |mysql| 5.1 or later (if using the slow query log)
   * |mysql| 5.6.9 or later (if using Performance Schema)
 
   .. tell about 8.0 |qan| 

|qan| displays its metrics in both visual and numeric form: the performance
related characteristics appear as plotted graphics with summaries.

.. include:: .res/replace.txt
