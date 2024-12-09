# Silence alerts

Create a silence when you want to suppress/stop alerts and their associated notifications for a very specific amount of time. 
Silences default to today’s current date and have a default duration of two hours.

You can also schedule a silence for a future date and time. This is referred to as a `Pending` silence, which can be observed on the Silences page.

During a silence, PMM continues to track metrics but does not trigger alerts or send notifications to any specified contact points. Once the silence expires alerts and notifications will resume.

Silenced alerts are still recorded under **Alerting > Fired Alerts** so that you can review them later. Silenced alerts show up as **Surpressed** and are disabled for as long as it's specified in the **Silence Duration**, or until you remove a silence.

## Using silences

You can silence an alert by creating a silence from the **Silences** page.  Here you define labels that match the alert that you want to silence.

To create a new silence:
{.power-number}

1. Click the **Create silence** button.
2. Select the start and end date to indicate when the silence should go into effect and expire.
3. Optionally, update the duration to alter the time for the end of silence in the previous step to correspond to the start plus the duration.
4. Enter one or more matching labels by filling out the **Name** and **Value** fields. Matchers determine which rules the silence will apply to. Note that all labels specified here must be matched by an alert for it to be silenced.
5. Enter any additional comments you would like about this silence - by default, the date the silence was created is placed here.
6. Review the affected alert instances that will be silenced.
7. Click **Save silece**.

For more information on working with silences, see [About alerting silences](https://grafana.com/docs/grafana/latest/alerting/manage-notifications/create-silence/) in the Grafana documentation.

## Alerting compatibility

### Template compatibility with PMM 2

After upgrading from the latest PMM 2 version to PMM 3, you will find all your alert templates under **Alerting > Alert rule templates**.

If you have any templates available in the  `/srv/ia/templates` folder, make sure to transfer them to `/srv/alerting/templates` as PMM 3 will look for custom templates in this location.

### Template compatibility with other alerting tools

If you have existing YAML alert templates that you want to leverage in Percona Alerting:
{.power-number}

1. Go to **Alerting > Alert rule templates** tab and click **Add template** at the top right-hand side of the table.
2. Upload a local .yaml file that contains the definition of one or more alert templates then click **Add**. Alert templates added in bulk will be displayed individually on **Alert rule templates** page.

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