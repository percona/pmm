--------------------------------------------------------------------------------
Exploring PMM API
--------------------------------------------------------------------------------

|pmm-server| allows user to visually interact with the API’s resources reflecting
all objects which |pmm| "knows" about. Browsing the API can be done using
`Swagger <https://swagger.io/tools/swagger-ui/>`_ UI, accessible at the
``/swagger`` endpoint URL:

.. figure:: ../.res/graphics/png/swagger.png

Clicking an objects allows to examine objects and execute requests to them:

.. figure:: ../.res/graphics/png/api.png

Objects which can be found while exploring are nodes, services, or agents.

* A **Node** represents a bare metal server, virtual machine or Docker container. It can also be of more specific type: one example is Amazon RDS Node. Node runs zero or more Services and Agents. It also has zero or more Agents providing insights for it.

* A **Service** represents something useful running on the Node: Amazon Aurora MySQL, MySQL, MongoDB, Prometheus, etc. It runs on zero (Amazon Aurora Serverless), single (MySQL), or several (Percona XtraDB Cluster) Nodes. It also has zero or more Agents providing insights for it.

* An **Agent** represents something that runs on the Node which is not useful itself but instead provides insights (metrics, query performance data, etc) about Nodes and/or Services. Always runs on the single Node (except External Exporters), provides insights for zero or more Services and Nodes.

Nodes, Services, and Agents have **Types** which define specific properties they have, and the specific logic they implement.

Nodes and Services are external by nature – we do not manage them (create, destroy), but merely maintain a list of them (add to inventory, remove from inventory) in pmm-managed. Most Agents, on the other hand, are started and stopped by pmm-agent. The only exception is the External Exporter Type which is started externally.

.. include:: ../.res/replace.txt
