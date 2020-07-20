.. _pmm.qan.home-page.opening:

--------------------------------------------------------------------------------
`Navigating to Query Analytics <pmm.qan.home-page.opening>`_
--------------------------------------------------------------------------------

To start working with QAN, choose *Query analytics*, which is the very
left item of the system menu on the top. The QAN dashboard will show up
several panels: a search panel, followed by a filter panel on the left, and a
panel with the list of queries in a summary table. The columns on this panel are
highly customizable, and by default, it displays *Query* column, followed by
few essential metrics, such as *Load*, *Count*, and *Latency*.

.. image:: /_images/PMM_Query_Analytics.jpg

Also it worth to mention that QAN data come in with typical 1-2 min delay,
though it is possible to be delayed more because of specific network condition
and state of the monitored object. In such situations QAN reports "no data"
situation, using sparkline to and showing a gap in place of the time interval,
for which data are not available yet.
