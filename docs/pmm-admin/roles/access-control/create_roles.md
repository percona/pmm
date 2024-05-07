
# Create access roles

Roles are a vital part of Access control. Roles provide users with access to specific, role-based metrics.

To create access roles in PMM, do the following:
{.power-number}

1. From the [main menu](../../../reference/ui/ui_components.md), go to **PMM Configuration  > Access Roles**. **Access Roles** tab
 opens.

    ![!](../../../_images/PMM_access_control_create_role.png)

1. Click **Create**. Create role page opens.


3. Enter the Role name and Role description.

    ![!](../../../_images/PMM_access_control_role_name.png)

4. Select the following from the drop-downs for metrics access:
    - Label
    - Operator
    - Value of the label.

    If you want to add more than one label for a role, click *+* and select the values from the drop-down.

    For information on how the Prometheus selectors work, see [Prometheus selectors](https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors).

5. Click **Create** role.

!!! note alert alert-primary "Note"
    To create roles, you must have admin privileges. For more information, see [Manage users](../../manage-users/index.md).