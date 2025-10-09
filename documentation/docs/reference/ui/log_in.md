# Log into PMM

!!! note "Percona Platform deprecation"
    Starting with PMM 3.7.0, the **Sign in with Percona Account** option will be removed as part of Percona Platform discontinuation. [Migrate to local PMM authentication ](../admin/manage-users/edit_users.html#migrating-from-percona-account-authentication-deprecated) before March 2026.

PMM offers two authentication methods:

## Local authentication (recommended)

To log into PMM with local credentials:
{.power-number}

1. Start a web browser and enter the server name or IP address of the PMM Server host in the address bar: ![PMM Login](../../images/PMM_Login.jpg)
3. Enter your local PMM username and password:
   - **Default username/password:** `admin`/`admin` 
   - **AWS deployments:** For security reasons, the default password on AWS installations is your EC2 Instance ID, which you can find in the AWS Console.
4. Click **Log in**.
5. If this is your first time logging in, you'll be asked to set a new password. You can either:
   - enter a new password in both fields and click **Submit**, or
   - click **Skip** to use the default password (not recommended for production).

The PMM Home dashboard loads:

![PMM Home dashboard](../../images/PMM_Home_Dashboard.png)

## Sign in with Percona Account (deprecated)

!!! warning "Deprecated feature"
    This authentication method is deprecated as of PMM 3.5.0 and will be removed in PMM 3.7.0 (March 2026).

If you currently use Percona Account authentication, make sure to [create local PMM user accounts](../../admin/manage-users/add_users.html) before the Platform shutdown.


## Next steps

- [Change your default password](../../admin/security/change_password.html)
- [Create additional users](../../admin/manage-users/add_users.html)
- [Configure user roles and permissions](../../admin/roles/index.html)

