# Edit users

You can edit users by changing the information or settings for an individual user account.

!!! caution alert alert-warning "Important"
    After changing the default admin password for the PMM Server, register the pmm-agent using the same credentials and add the services again. Otherwise, PMM will cease to monitor the service/nodes.

## Grant or revoke admin privileges

You can grant or revoke admin access to a user as follows:
{.power-number}

1. Go to **Users and access**.

2. Locate the user account you want to update and click the Edit (pencil) icon. 

3. In the **User information** dialog, scroll to **Permissions** section and click **Change**

4. Choose **Yes/No**, depending on whether you want to provide admin access or not.

4. Click **Change**.

## Change organization role

To change the organization role assigned to your user account:
{.power-number}

1. On the **Users and access**, click the user for whom you want to change the role.

2. Locate the user account you want to update and click the Edit (pencil) icon. 

3. In the **Organizations** section, click **Change role**.

4. Select the role from the drop-down and click **Save**.

Here are the privileges for the various *roles*:

- **Admin** - Managing data sources, teams, and users within an organization

- **Editor** - Creating and editing dashboards

- **Viewer** - Viewing dashboards

For detailed information on the privileges for these roles and the different tasks that they can perform, see [Grafana organization roles](https://grafana.com/docs/grafana/latest/permissions/organization_roles/).
