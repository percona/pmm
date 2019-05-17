.. _pmm-admin.annotate:

--------------------------------------------------------------------------------
Annotating important Application Events
--------------------------------------------------------------------------------

.. _pmm-admin.annotate.adding:

`Adding annotations <client-annotations.html#adding-annotations>`_
--------------------------------------------------------------------------------

Use the |pmm-admin.annotate| command to set notifications about important
application events and display them on all dashboards. By using annotations, you
can conveniently analyze the impact of application events on your database.

.. _pmm-admin.annotate.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. include:: .res/code/pmm-admin.annotate.tags.txt

.. _pmm-admin.annotate.options:

.. rubric:: OPTIONS

The |pmm-admin.annotate| supports the following options:

|opt.tags|

   Specify one or more tags applicable to the annotation that you are
   creating. Enclose your tags in quotes and separate individual tags by a
   comma, such as "tag 1,tag 2".

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`.

.. _pmm.metrics-monitor.annotation.application-event.marking:

`Marking Important Events with Annotations <client-annotations.html#application-event-marking>`_
------------------------------------------------------------------------------------------------------------

Some events in your application may impact your database. Annotations
visualize these events on each dashboard of |pmm-server|.

.. figure:: ../.res/graphics/png/pmm-server.mysql-overview.mysql-client-thread-activity.1.png

   An annotation appears as a vertical line which crosses a graph at a
   specific point. Its text explains which event occurred at that time.

To create a new annotation, run |pmm-admin.annotate| command on
|pmm-client| passing it text which explains what event the new
annotation should represent. Use the |opt.tags| option to supply one
or more tags separated by a comma.

You may toggle displaying annotations on metric graphs by using the
|gui.pmm-annotations| checkbox.

.. figure:: ../.res/graphics/png/pmm-server.pmm-annotations.png

   Remove the checkmark from the |gui.pmm-annotations| checkbox to
   hide annotations from all dashboards.

.. seealso::

   Adding annotations
     :ref:`pmm-admin.annotate`

   |grafana| Documentation:
      - `Annotations <http://docs.grafana.org/reference/annotations/#annotations>`_
      - `Using annotations in queries <http://docs.grafana.org/reference/annotations/#querying-other-data-sources>`_

.. include:: ../.res/replace.txt
