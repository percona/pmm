# Add users

Adding users with basic authentication (username and password stored in PMM) from **Users and access > Users** tab.

If your organization uses LDAP, OAuth, SAML, or other authentication methods, users are managed through your external authentication system. See [authentication methods](../../reference/ui/log_in.md) for more information.

To add a new user in PMM:
{.power-number}

1. Go to **Users and access > Users > New user**.
2. On the **New user** dialog box, fill in the user's details. Use a password that meets the [password requirements](#password-requirements). For existing Grafana users, you can enter their Grafana username in the **Email** field.

3. Click **Create user**.

The new user can now log in to PMM using the username and password you created.

## Password requirements

Starting with PMM 3.11.0, PMM accepts a new or changed password for a user with basic authentication only if it:

- is at least 12 characters long
- contains at least one uppercase and one lowercase letter
- contains at least one digit
- contains at least one symbol, such as `!`, `-` or `_`

PMM rejects a password that doesn't meet these requirements, whether you set it in the PMM UI, through the API, or with the `change-admin-password` script. Passwords set before the upgrade keep working until you change them.

If you use the password in `--server-url` to register PMM Client, [encode its symbols](../../troubleshoot/config_issues.md#passwords), for example `#` as `%23` and `@` as `%40`.

To turn off the password policy, [set the environment variable](../../install-pmm/install-pmm-server/deployment-options/docker/env_var.md#use-environment-variables) `GF_AUTH_BASIC_PASSWORD_POLICY` to `false` on the PMM Server container. Turning it off isn't recommended in production.

## Assign user roles

After creating a user, you may want to assign them specific roles or permissions. See [Edit users](edit_users.md) for information on:

- Granting or revoking admin privileges
- Changing organization roles (Admin, Editor, Viewer)
- Managing user permissions