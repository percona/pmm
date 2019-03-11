================================================================================
PMM for Percona Customers
================================================================================

.. _pmm.metrics-monitor.dashboard.snapshot.creating:

`Creating a Metrics Monitor dashboard Snapshot as part of a Percona Support engagement <pmm.metrics-monitor.dashboard.snapshot.creating>`_
==========================================================================================================================================

A snapshot is a way to securely share your dashboard with |percona|. When
created, we strip sensitive data like queries (metrics, template variables, and
annotations) along with panel links. The shared dashboard will only be available
for viewing by |percona| engineers. The content on the dashboard will assist
|percona| engineers in troubleshooting your case.

You can safely leave the defaults set as they are, but for further information:

Snapshot name
   The name |percona| will see when viewing your dashboard.

Expire 
   How long before snapshot should expire, configure lower if
   required. |percona| automatically purges shared dashboards after 90 days.

Timeout (seconds)
   Duration the dashboard will take to load before the snapshot is
   generated.

First, open the dashboard that you would like to share. Click the
|gui.share| button at the top of the page and select the
|gui.snapshot| command. Finally, click the
|gui.share-with-percona-team| button.

.. figure:: .res/graphics/png/metrics-monitor.share.snapshot.png

   The |gui.snapshot| tab in the |gui.share| dialog window.

.. rubric:: What to do next

After clicking |gui.share-with-percona-team|, wait for the dashboard to be generated,
and you will be provided a unique URL that then needs to be communicated to
|percona| via the ticket.

.. include:: .res/replace.txt
