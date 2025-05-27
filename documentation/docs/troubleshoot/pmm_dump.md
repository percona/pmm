# Export PMM data with PMM Dump

PMM data dumps are compressed tarball files containing a comprehensive export of your PMM metrics and QAN data collected by PMM Server.

You can download these dataset files locally, or share them with Percona Support via an SFTP server. This enables you to share PMM data securely, which is especially useful when you need you troubleshoot PMM issues without without providing access to your PMM instance.

Starting with 2.41, PMM enables you to generate PMM Datasets straight from PMM. If you are using an older PMM version, you can use the [standalone PMM Dump utility](https://docs.percona.com/pmm-dump-documentation/installation.html) instead.

## Dump contents

The **dump.tar.gz** dump file is a .TAR archive compressed via Gzip. Here's what's inside the folders it contains:

 - **meta.json**: metadata about the data dump
 - **vm**: Victoria Metrics data chunks in native VM format, organized by timeframe
 - **ch**: Query Analytics (QAN) data stored in ClickHouse, organized by rows count
 - **log.json**: logs detailing the export and archive creation process

## Create a data dump

To create a dump of your dataset:
{.power-number}

1. From the top-right corner of the PMM home page, go to <i class="uil uil-question-circle"></i>  **Help > PMM Dump**.
2. Click **Create dataset** to go to the **Export new dataset** page.
3. Choose the service for which you want to create the dataset or leave it empty to export all data.
4. Define the time range for the dataset.
5. Enable **Export QAN** to include Query Analytics (QAN) metrics alongside the core metrics.
6. Enable **Ignore load** to export the dump bypassing the default resource limit restrictions.
7. Click **Create dataset**. This will generate a data dump file and automatically record an entry in the PMM Dump table. From there, you can use the options available in the **Options** menu to send the dump file to Percona Support or download it locally for internal usage.

## Send a data dump to Percona Support

If you are a Percona Customer, you can securely share PMM data dumps with Percona Support via SFTP.
{.power-number}

1. From the top-right corner of the PMM home page, go to <i class="uil uil-question-circle"></i>  **Help > PMM Dump**.
2. Select the PMM dump entry which you want to send to Support.
3. In the **Options** column, expand the table row to check the PMM Service associated with the dataset, click the ellipsis (three vertical dots) and select **Send to Support**.
4. Fill in the [details of the SFTP server](https://percona.service-now.com/percona?id=kb_article_view&sysparm_article=KB0010247&sys_kb_id=bebd04da87e329504035b8c9cebb35a7&spa=1), then click **Send**.
5. Update your Support ticket to let Percona know that you've uploaded the dataset on the SFTP server.