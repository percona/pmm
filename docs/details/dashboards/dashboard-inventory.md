# PMM Inventory

The *Inventory* dashboard is a high level overview of all objects registered by PMM.

To see it select <i class="uil uil-cog"></i> *Configuration* → {{icon.inventory}} *PMM Inventory* → {{icon.inventory}} *Inventory list*.

![!image](../../_images/PMM_Inventory.jpg)

Inventory objects form a hierarchy with Node at the top, then Service and Agents assigned to a Node.

There are three tabs where items for each type are listed with their details:

- *Services*
    – Individual service names and where they run, against which agents will be assigned. Each instance of a service gets a `service_id` value that is related to a `node_id`. Examples are MySQL, Amazon Aurora MySQL. This feature also allows to support multiple mysqld instances on a single node, with different service names, e.g. `mysql1-3306`, and `mysql1-3307`.

- *Agents*
    – Each binary (exporter, agent) running on a client will get an `agent_id` value. Examples:

        - `pmm-agent` one is the top of the tree, assigned to a `node_id`
        - `node_exporter` is assigned to pmm-agent `agent_id`
        - `mysqld_exporter` and QAN MySQL Perfschema are assigned to a `service_id`.

- *Nodes*
    – Where the service and agents will run. Assigned a `node_id`, associated with a `machine_id` (from `/etc/machine-id`). Some examples are bare metal, virtualized, container.

## Removing items from the inventory

You can remove items from the inventory.

1. Select <i class="uil uil-cog"></i> *Configuration* → {{icon.inventory}} *PMM Inventory* → {{icon.inventory}} *Inventory list*.

2. In the first column, select the items to be removed.

    ![!image](../../_images/PMM_Inventory_Item_Selection.jpg)

3. Click *Delete*. The interface will ask you to confirm the operation:

    ![!image](../../_images/PMM_Inventory_Item_Delete.jpg)
