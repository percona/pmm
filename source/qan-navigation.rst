.. _pmm.qan.home-page.opening:

--------------------------------------------------------------------------------
`Navigating to Query Analytics <pmm.qan.home-page.opening>`_
--------------------------------------------------------------------------------
   
To start working with |qan|, choose the *Query analytics", which is the very
left item of the system menu on the top. The |qan| dashboard will show up
several panels: a search panel, followed by a filter panel on the left, and a
panel with the list of queries in a summary table. The columns on this panel are
highly customizable, and by default, it displays *Query* column, followed by
three essential metrics: *Load*, *Count*, and *Latency*.

.. figure:: .res/graphics/png/qan01.png

   The query summary table.

Also it worth to mention that |qan| data come in with typical 1-2 min delay,
though it is possible to be delayed more because of specific network condition
and state of the monitored object. In such situations |qan| reports "no data"
situation, using sparkline to and showing a gap in place of the time interval,
for which data are not available yet.

.. figure:: .res/graphics/png/qan.query-summary-table.sparkline.png

   Showing intervals for which data are unavailable yet.

.. include:: .res/replace.txt
