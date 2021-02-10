# PMM Inventory

![image](../../_images/PMM_Inventory.jpg)

The *Inventory* dashboard is a high level overview of all objects PMM “knows” about.

It contains three tabs (*services*, *agents*, and *nodes*) with lists of the correspondent objects and details about them, so that users are better able to understand which objects are registered against PMM Server. These objects are composing a hierarchy with Node at the top, then Service and Agents assigned to a Node.

* **Nodes** – Where the service and agents will run. Assigned a `node_id`, associated with a `machine_id` (from `/etc/machine-id`). Few examples are bare metal, virtualized, container.

* **Services** – Individual service names and where they run, against which agents will be assigned. Each instance of a service gets a `service_id` value that is related to a `node_id`. Examples are MySQL, Amazon Aurora MySQL. This feature also allows to support multiple mysqld instances on a single node, with different service names, e.g. mysql1-3306, and mysql1-3307.

* **Agents** – Each binary (exporter, agent) running on a client will get an `agent_id` value.

    * `pmm-agent` one is the top of the tree, assigned to a `node_id`

    * `node_exporter` is assigned to pmm-agent `agent_id`

    * `mysqld_exporter` & QAN MySQL Perfschema are assigned to a `service_id`.

Examples are `pmm-agent`, `node_exporter`, `mysqld_exporter`, QAN MySQL Perfschema.

## Removing items from the inventory

You can remove items from the inventory.

1. Open *Home Dashboard > PMM Inventory*

2. In the first column, select the items to be removed.

    ![image](../../_images/PMM_Inventory_Item_Selection.jpg)

3. Click *Delete*. The interface will ask you to confirm the operation:

    ![image](../../_images/PMM_Inventory_Item_Delete.jpg)
