.. _pmm-admin.annotate:

#######################################
Annotating important Application Events
#######################################

******************
Adding annotations
******************

The ``pmm-admin annotate`` command registers a moment in time, marking it with a text string called an *annotation*.

The presence of an annotation shows as a vertical dashed line on a dashboard graph; the annotation text is revealed by mousing over the caret indicator below the line.

Annotations are useful for recording the moment of a system change or other significant application event.

They can be set globally or for specific nodes or services.

.. image:: /_images/pmm-server.mysql-overview.mysql-client-thread-activity.1.png

.. rubric:: USAGE

``pmm-admin annotate [--node|--service] <annotation> [--tags <tags>] [--node-name=<node>] [--service-name=<service>]``

.. rubric:: OPTIONS

``<annotation>``
    The annotation string. If it contains spaces, it should be quoted.

``--node``
   Annotate the current node or that specified by ``--node-name``.

``--service``
   Annotate all services running on the current node, or that specified by ``--service-name``.

``--tags``
   A quoted string that defines one or more comma-separated tags for the annotation. Example: ``"tag 1,tag 2"``.

``--node-name``
    The node name being annotated.

``--service-name``
    The service name being annotated.

.. seealso:: :ref:`pmm.ref.pmm-admin`

.. _application-event-marking:

*********************
Annotation Visibility
*********************

You can toggle the display of annotations on graphs with the *PMM Annotations* checkbox.

.. image:: /_images/pmm-server.pmm-annotations.png

Remove the check mark to hide annotations from all dashboards.

.. seealso:: `docs.grafana.org: Annotations <http://docs.grafana.org/reference/annotations/>`_
