.. _dashboard-home:

Inventory Dashboard
================================================================================

The *Inventory* dashboard is a high level overview of all objects |pmm| "knows"
about.

It contains three tabs (*services*, *agents*, and *nodes*) with lists of the 
correspondent objects and details about them, so that users are better able to
understand which objects are registered against PMM Server. These objects are
composing a hierarchy with Node at the top, then Service and Agents assigned to
a Node.

* **Nodes** – Where the service and agents will run. Assigned a ``node_id``,
  associated with a ``machine_id`` (from ``/etc/machine-id``). Few examples are
  bare metal, virtualized, container.

* **Services** – Individual service names and where they run, against which
  agents will be assigned. Each instance of a service gets a ``service_id``
  value that is related to a ``node_id``. Examples are MySQL, Amazon Aurora
  MySQL. This feature also allows to support multiple mysqld instances on
  a single node, with different service names, e.g. mysql1-3306, and mysql1-3307.

* **Agents** – Each binary (exporter, agent) running on a client will get an
  ``agent_id`` value. 

  * pmm-agent one is the top of the tree, assigned to a ``node_id``

  * node_exporter is assigned to pmm-agent ``agent_id``

  * mysqld_exporter & QAN MySQL Perfschema are assigned to a service_id.

  Examples are pmm-agent, node_exporter, mysqld_exporter, QAN MySQL Perfschema.

.. include:: ../.res/replace.txt
