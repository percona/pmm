# Percona Alerting issues

## No Alert rule templates tab on the Alerting page

Percona Alerting option isn't active.
{.power-number}

1. Go to **PMM Configuration > Settings > Advanced Settings**.
2. Enable **Alerting**.

## Custom alert rule templates not migrated to Percona Alerting
If you have used Integrated Alerting in previous PMM versions, and had custom templates under ``/srv/ia/templates``, make sure to transfer them to ``/srv/alerting/templates``. 
PMM is no longer sourcing templates from the ``ia`` folder, since we have deprecated Integrated Alerting with the 2.31 release. 

## Unreachable external IP addresses

If you get an email or page from your system that the IP is not reachable from outside my organization, do the following:

To configure your PMM Server’s Public Address, select <i class="uil uil-cog"></i> **Configuration** → <i class="uil uil-setting"></i> **Settings* → *Advanced Settings**, and supply an address to use in your alert notifications.

## Alert Rule Templates are disabled

Built-In alerts are not editable, but you can copy them and edit the copies. (In [PMM 2.14.0](../release-notes/2.14.0.md) and above).

If you create a custom alert rule template, you will have access to edit.

