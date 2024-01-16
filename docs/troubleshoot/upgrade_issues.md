# Upgrade issues

## PMM server not updating correctly


If the PMM server wasn't updated correctly, or if you have concerns about the release, you can force the update process in 2 ways:
{.power-number}

1. From the UI - Home panel: click the Alt key on the reload icon in the Update panel to make the Update Button visible even if you are on the same version as available for update. Pressing this button will force the system to rerun the update so that any broken or not installed components can be installed. In this case, you'll go through the usual update process with update logs and successful messages at the end.

2. By API call (if UI not available): You can call the Update API directly with:

    ```sh
    curl --user admin:admin --request POST 'http://PMM_SERVER/v1/Updates/Start'
    ```

    Replace `admin:admin` with your username/password, and replace `PMM_SERVER` with your server address.

    !!! note alert alert-primary "Note"
        You will not see the logs using this method.

    Refresh The Home page in 2-5 minutes, and you should see that PMM was updated.

3. Upgrade PMM server using [Docker](../pmm-upgrade/upgrade_docker.md).


## PMM server not showing latest versions available with the instances created from AWS

For PMM versions prior to 2.33.0, in specific environments, including AWS, some EPEL repository mirrors did not respond within the time limit defined by `pmm-update` (currently set to 30 seconds). It was causing supervisord to kill pmm-update-checker, which determines if a newer PMM Server is available for upgrade.

**Solution**

Log in to the PMM Server and run the following command as a root user:

```sh
   $ yum-config-manager --setopt=epel.timeout=1 --save
```

## PMM server fails while upgrading

A bug in PMM Server ansible scripts caused PMM to upgrade Nginx's dependencies without updating Nginx itself. Due to this, PMM throws an error while upgrading and cannot upgrade to a newer version. 

!!! caution alert alert-warning "Important"
    This issue has been resolved for PMM version 2.33.0. However, the issue persists on all the versions prior to 2.33.0.


**Solution**

While PMM is being upgraded, log in to the PMM server and run the following command:

```sh
   sed -i 's/- nginx/- nginx*/' /usr/share/pmm-update/ansible/playbook/tasks/update.yml
```


## Admin user cannot access PMM after upgrading

After upgrading PMM from version 2.39.0 to 2.40.0 (not el7) using Docker, the `admin` user cannot access the PMM UI.

**Solution**: To fix the problem and gain back admin access to the PMM interface execute the following:

```sh
# psql -U grafana
grafana=> update "user" set id='1' where login='admin';
UPDATE 1
grafana=> \q

# grafana cli --homepath=/usr/share/grafana --config=/etc/grafana/grafana.ini admin reset-admin-password <PASS>
``` 


