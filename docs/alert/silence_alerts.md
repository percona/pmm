# Silence alerts

Create a silence when you want to suppress/stop alerts and their associated notifications for a very specific amount of time. 
Silences default to today’s current date and have a default duration of two hours. 

You can also schedule a silence for a future date and time. This is referred to as a `Pending` silence, which can be observed on the Silences page.

During a silence, PMM continues to track metrics but does not trigger alerts or send notifications to any specified contact points. Once the silence expires alerts and notifications will resume.

Silenced alerts are still recorded under **Alerting > Fired Alerts** so that you can review them later. Silenced alerts show up as **Surpressed** and are disabled for as long as it's specified in the **Silence Duration**, or until you remove a silence.

## Using silences

You can silence an alert from the **Fired alerts** page or from the **Alert rules** page by expanding the Alert Rule and clicking the *Silence* button.

![!](../_images/alerting-silence-alert-button.png)

You can also create a silence from the **Silences** page, but here you would also need to define labels that match the alert that you want to silence.

To create a new silence from the **Silences** page:

1. Click the **New Silence** button.
2. Select the start and end date to indicate when the silence should go into effect and expire.
3. Optionally, update the duration to alter the time for the end of silence in the previous step to correspond to the start plus the duration.
4. Enter one or more matching labels by filling out the **Name** and **Value** fields. Matchers determine which rules the silence will apply to. Note that all labels specified here must be matched by an alert for it to be silenced.
5. Enter any additional comments you would like about this silence - by default, the date the silence was created is placed here.
6. Review the affected alert instances that will be silenced.
7. Click **Submit** to create the silence.

For more information on working with silences, see [About alerting silences](https://grafana.com/docs/grafana/latest/alerting/manage-notifications/create-silence/) in the Grafana documentation.

## Alerting compatibility

### Template compatibility with previous PMM versions

If you have used Integrated Alerting in previous PMM versions, your custom alert rule templates will be automatically migrated to PMM 2.31. After upgrading to this new version, you will find all your alert templates under **Alerting > Alert Templates**.

If you have any templates available in the  ``/srv/ia/templates`` folder, make sure to transfer them to ``/srv/alerting/templates`` as PMM 2.31 and later will look for custom templates in this location.

If you are upgrading from PMM 2.25 and earlier, alert templates will not be automatically migrated. This is because PMM 2.26.0 introduced significant changes to the core structure of rule templates.

In this scenario, you will need to manually recreate any custom rule templates that you want to transfer to PMM 2.26.0 or later.

### Template compatibility with other alerting tools

If you have existing YAML alert templates that you want to leverage in Percona Alerting:

1. Go to **Alerting > Alert Rule Templates** tab and click **Add** at the top right-hand side of the table.
2. Click **Add** and upload a local .yaml file that contains the definition of one or more alert templates. Alert templates added in bulk will be displayed individually on **Alert rule templates** page.

### Migrate alert rules

Alert rules created with Integrated Alerting in PMM 2.30 and earlier are not automatically migrated to Percona Alerting.

After upgrading to PMM 2.31, make sure to manually migrate any alert rules that you want to transfer to PMM 2.31 using the [Integrated Alerting Migration Script](https://github.com/percona/pmm/blob/main/ia_migration.py).

#### Script commands

The default command for migrating rules is:
```yaml 
python3 ia_migration.py -u admin -p admin
```
To see all the available options, check the scrip help using `ia_migration.py -h`

#### Script prerequisites

- Python version 3.x, which you can download from [Python Downloads centre](https://www.python.org/downloads/).
- [Requests library](https://requests.readthedocs.io/en/latest/user/install/#install), which you can install with the following command: ```pip3 install requests```.

!!! caution alert alert-warning "Important"
    The script sets all migrated alert rules to Active. Make sure to silence any alerts that should not be firing.

For more information about the script and advanced migration options, check out the help information embedded in the script.

## Disable Percona Alerting

If for some reason you want to disable PMM Alert templates and keep only Grafana-managed alerts:

1. Go to **Configuration > PMM Settings**.
2. Disable the **Alerting** option. The **Alerting** page will now display only Grafana-managed alert rules.