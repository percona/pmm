# Manage access roles

You can manage roles in PMM by editing or deleting a role.

## Edit roles

To edit access roles, do the following:

1. From the *Main* menu, navigate to {{icon.configuration}} *Configuration → Access Roles*. The *Access Roles* tab opens.

2. On the role you want to edit, click the *ellipsis (three vertical dots) > edit role* in the *Options* column. The *Edit* role page opens.

    ![!](../../images/PMM_access_control_edit_role.png)

3. Make the required changes to the role.

    ![!](../../images/PMM_access_control_edit_role_changes.png)


4. Click Save Changes.


## Set a role as default

When a user signs in to PMM for the first time and the user has no role assigned, the user is automatically assigned the *Default* role. For administrators, the default role provides a convenient way to configure default permissions for new users.


To set a role as default, do the following:

1. From the *Main* menu, navigate to {{icon.configuration}} *Configuration → Access Roles*. The *Access Roles* tab opens.

2. On the role you want to set as default, click the *ellipsis (three vertical dots) → set as default* in the *Options* column.

 ![!](../../images/PMM_access_control_default_role_changes.png)


## Remove roles

To remove access roles, do the following:

1. From the *Main* menu, navigate to {{icon.configuration}} *Configuration → Access Roles*. The *Access Roles* tab opens.

2. On the role you want to remove, click the *ellipsis (three vertical dots) →  Delete* in the *Options* column. Delete role pop-up opens.

    ![!](../../images/PMM_access_control_delete_role.png)

3. Starting with **PMM 2.36.0**, if the role that you want to delete is already assigned to a user, you will see a drop-down with replacement roles. Select the replacement role and the selected role will be assigned to the user.


    ![!](../../images/PMM_access_control_delete_replace_role.png)

4. Click *Confirm* and delete the role.















