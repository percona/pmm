# Percona Alerting issues

## No Alert rule templates tab on the Alerting page

Percona Alerting option isn't active.
{.power-number}

1. Go to **PMM Configuration > Settings > Advanced Settings**.
2. Enable **Alerting**.

## Custom alert rule templates not migrated to Percona Alerting

After upgrading from the latest PMM 2 version to PMM 3, you will find all your alert templates under **Alerting > Alert rule templates**.

If you have any templates available in the  `/srv/ia/templates` folder, make sure to transfer them to `/srv/alerting/templates` as PMM 3 will look for custom templates in this location.

## Unreachable external IP addresses

If you get an email or page from your system that the IP is not reachable from outside my organization, do the following:

To configure your PMM Server’s Public Address, select  **PMM Configuration > Settings > Advanced Settings**, and supply an address to use in your alert notifications.

## Alert Rule Templates are disabled

Built-in alerts are not editable, but you can copy them and edit the copies. 

If you create a custom alert rule template, you will have access to edit.