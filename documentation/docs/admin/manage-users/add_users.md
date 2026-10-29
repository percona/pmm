# Add users

Adding users with basic authentication (username and password stored in PMM) from **Users and access > Users** tab.

If your organization uses LDAP, OAuth, SAML, or other authentication methods, users are managed through your external authentication system. See [authentication methods](../../reference/ui/log_in.md) for more information.

To add a new user in PMM:
{.power-number}

1. Go to **Users and access > Users > New user**.
2. On the **New user** dialog box, fill in the user's details. For existing Grafana users, you can enter their Grafana username in the **Email** field.

3. Click **Create user**.

The new user can now log in to PMM using the username and password you created.

## Assign user roles

After creating a user, you may want to assign them specific roles or permissions. See [Edit users](edit_users.md) for information on:

- Granting or revoking admin privileges
- Changing organization roles (Admin, Editor, Viewer)
- Managing user permissions