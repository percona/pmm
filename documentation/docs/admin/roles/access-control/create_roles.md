
# Create access roles

Roles are essential components of PMM's access control system. They allow you to limit users' access to specific metrics based on their responsibilities and permissions.

## Before you begin

- You must have administrator privileges to create roles. For more information, see [Manage users](../../manage-users/index.md).
- Access control must be enabled in PMM settings

## Create a new role

To create access roles in PMM:
{.power-number}

1. From the [main menu](../../../reference/ui/ui_components.md), go to **PMM Configuration > Settings > Advanced Settings** and enable the **Access control** option.
2. Go to **Administration > Users and access > Access Roles**.

    ![PMM Access Control - Create role](../../../images/lbac/PMM_access_control_create_role.png)

3. Click **Create**.
4. On the **Create role** page, enter the Role name and Role description.
5. Configure metrics access by setting label selectors:
    - select a Label (e.g., "service_name", "environment")
    - choose an Operator (e.g., "=", "!=", "=~")
    - enter the Value for the selected label

    If you want to add more than one label for a role, click *+* and select the values from the drop-down.

    For information on how the Prometheus selectors work, see [Prometheus selectors](https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors).

6. Review your selections, then click **Create** to finalize the role.


