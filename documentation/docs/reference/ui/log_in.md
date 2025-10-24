# Log into PMM

!!! note "Percona Platform deprecation"
    Starting with PMM 3.7.0, the **Sign in with Percona Account** option will be removed as part of Percona Platform discontinuation. [Migrate to a supported authentication method](../../admin/manage-users/edit_users.md#migrate-from-percona-account-authentication-deprecated) before March 2026.

PMM supports multiple authentication methods. The most common are:

## Basic authentication

Basic authentication uses usernames and passwords stored in PMM. This is the default authentication method.

To log into PMM with basic authentication:
{.power-number}

1. Start a web browser and enter the server name or IP address of the PMM Server host in the address bar: 
   
   ![PMM Login](../../images/PMM_Login.jpg)

2. Enter your PMM username and password:
   - **Default username/password:** `admin`/`admin` 
   - **AWS deployments:** For security reasons, the default password on AWS installations is your EC2 Instance ID, which you can find in the AWS Console.

3. Click **Log in**.

4. If this is your first time logging in, you'll be asked to set a new password. You can either:
   - enter a new password in both fields and click **Submit**, or
   - click **Skip** to use the default password (not recommended for production).

The PMM Home dashboard loads:

![PMM Home dashboard](../../images/PMM_Home_Dashboard.png)

## Other authentication methods

PMM supports all authentication methods available in Grafana, including:

- **LDAP** - Integrate with your directory service
- **OAuth 2.0** - GitHub, GitLab, Google, Azure AD, Okta, and other providers
- **SAML** - Enterprise single sign-on
- **Other Grafana-supported methods**

For setup instructions, see [Grafana's authentication documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/).

## Sign in with Percona Account (deprecated)

!!! warning "Deprecated feature"
    This authentication method is deprecated as of PMM 3.5.0 and will be removed in PMM 3.7.0 (March 2026).

If you currently use Percona Account authentication, [migrate to a supported authentication method](../../admin/manage-users/edit_users.md#migrate-from-percona-account-authentication-deprecated) before the Platform shutdown.

## Next steps

- [Create additional users](../../admin/manage-users/add_users.md)
- [Configure user roles and permissions](../../admin/roles/index.md)
- [Configure authentication methods](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/)