
# Create access roles

Roles are a vital part of Access control. Roles provide users with access to specific, role-based metrics.

To create access roles in PMM, do the following:
{.power-number}

1. From the [main menu](../../../reference/ui/ui_components.md), go to **PMM Configuration > Settings > Advanced Settings** and enable the **Access Roles** option.
2. Go to **Administration > Users and access > Access Roles**.

    ![!](../../../images/PMM_access_control_create_role.png)

3. Click **Create**.
4. On the **Create role** page, enter the Role name and Role description.
5. Select the following from the drop-downs for metrics access:
    - Label
    - Operator
    - Value of the label

    If you want to add more than one label for a role, click *+* and select the values from the drop-down.

    For information on how the Prometheus selectors work, see [Prometheus selectors](https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors).

6. Click **Create** role.

!!! note alert alert-primary "Note"
    To create roles, you must have admin privileges. For more information, see [Manage users](../../manage-users/index.md).