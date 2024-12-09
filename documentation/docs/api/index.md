# About PMM API

PMM Server lets you visually interact with API resources representing all objects within PMM. You can browse the API using the [Swagger](https://swagger.io/tools/swagger-ui/) UI, accessible at the `/swagger/` endpoint URL:

![!image](../images/PMM_Swagger_API_Get_Logs_View.jpg)

Clicking an object lets you examine objects and execute requests on them:

![!image](../images/PMM_Swagger_API_Get_Logs_Execute.jpg)

The objects visible are nodes, services, and agents:

- A **Node** represents a bare metal server, a virtual machine, a Docker container, or a more specific type such as an Amazon RDS Node. A node runs zero or more Services and Agents, and has zero or more Agents providing insights for it.

- A **Service** represents something useful running on the Node: Amazon Aurora MySQL, MySQL, MongoDB, etc. It runs on zero (Amazon Aurora Serverless), single (MySQL), or several (Percona XtraDB Cluster) Nodes. It also has zero or more Agents providing insights for it.

- An **Agent** represents something that runs on the Node which is not useful in itself, but instead provides insights (metrics, query performance data, etc.) about Nodes and/or Services. An agent always runs on the single Node (except External Exporters), and provides insights for zero or more Services and Nodes.

Nodes, Services, and Agents have **Types** which define specific their properties, and their specific logic.

Nodes and Services are inherently external. We don't manage their creation or deletion, but rather maintain a list of them within PMM Server by adding them to or removing them from the inventory. The majority of Agents are initiated and halted by pmm-agent, with one exception being the External Exporter Type, which is initiated externally.

## Service accounts and authentication

For information about controlling access to the PMM Server components and resources, see the **[Authentication with service accounts](../api/authentication.md)** topic.