# Export PMM data with PMM Dump

PMM data dumps are compressed tarball files containing a comprehensive export of your PMM metrics and QAN data collected by PMM Server.

You can download these dataset files locally, or share them with Percona Support via an SFTP server. This enables you to share PMM data securely, which is especially useful when you need to troubleshoot PMM issues without providing access to your PMM instance.

PMM3 enables you to generate datasets straight from PMM UI. PMM 2.41 or older use the [standalone PMM Dump utility](https://docs.percona.com/pmm-dump-documentation/installation.html) instead.

## Access requirements

PMM Dump access is restricted based on user roles:

| User role | Can access PMM Dump | Can create datasets |
|-----------|---------------------|---------------------|
| Admin (with or without Grafana Admin) | Yes | Yes |
| Editor with Grafana Admin | Yes | Yes  |
| Editor without Grafana Admin | No | No |
| Viewer with Grafana Admin | Yes | Yes  |
| Viewer without Grafana Admin | No | No |

If you cannot see the **PMM Dump** option under **Help > Help Center** or receive access errors when trying to access it directly, check that your user account has the necessary permissions.

## Dump contents

The **dump.tar.gz** dump file is a .TAR archive compressed via Gzip. Here's what's inside the folders it contains:

 - **meta.json**: metadata about the data dump
 - **vm**: Victoria Metrics data chunks in native VM format, organized by timeframe
 - **ch**: Query Analytics (QAN) data stored in ClickHouse format, organized by row count
 - **log.json**: logs detailing the export and archive creation process. Passwords and credentials are automatically masked.

## Create a data dump

To create a dump of your dataset:
{.power-number}

1. From the main menu, click **Help > PMM Dump > Manage datasets**. If you don't see PMM Dump in the menu, your user account may not have sufficient permissions.
2. Click **Create dataset** to go to the **Export new dataset** page.
3. Choose the service for which you want to create the dataset or leave it empty to export all data.
4. Define the time range for the dataset.
5. Toggle on **Export QAN** to include Query Analytics (QAN) metrics alongside the core metrics.
6. (Optional) Toggle on **Enable encryption** to encrypt the dump file and enter a password in the **Encryption password**. PMM does not store it, and if you lose it you will need to create a new dump. See [Encrypted dumps](#encrypted-dumps).
7. Click **Create dataset**. This will generate a data dump file and automatically record an entry in the PMM Dump table. From there, you can use the options available in the **Options** menu to send the dump file to Percona Support or download it locally for internal usage.

## Encrypted dumps

Encrypt your dump when you want to protect sensitive data before sharing it externally, for example when [sending a dump file to Percona Support](#send-a-data-dump-to-percona-support).

When you encrypt a dump, PMM saves the file with an `.enc` suffix, for example `69d4df06-f87a-4cfe-aa7b-9a79d449a9b4.tar.gz.enc`. The encryption password is not stored by PMM so if you lose it, you will need to create a new dump.

PMM encrypts dump files using AES-256-CTR. To decrypt a dump, use the [pmm-dump CLI tool](https://docs.percona.com/pmm-dump-documentation/installation.html) or run the following `openssl` command:

```bash
openssl enc -d -aes-256-ctr -pbkdf2 -in dump.tar.gz.enc -out dump.tar.gz
```

When importing a non-encrypted dump, pass the `--no-encryption` flag to the pmm-dump CLI tool to skip the password prompt.


## Send a data dump to Percona Support

If you are a Percona Customer, you can securely share PMM data dumps with Percona Support via SFTP.
{.power-number}

1. From the main menu, go to <i class="uil uil-question-circle"></i>  **Help > PMM Dump > Manage datasets**.
2. Select the PMM dump entry which you want to send to Support.
3. In the **Options** column, expand the table row to check the PMM Service associated with the dataset, then click the ellipsis (three vertical dots) and select **Send to Support**.
4. Fill in the [details of the SFTP server](https://percona.service-now.com/percona?id=kb_article_view&sysparm_article=KB0010247&sys_kb_id=bebd04da87e329504035b8c9cebb35a7&spa=1), then click **Send**.
5. Update your Support ticket to let Percona know that you've uploaded the dataset on the SFTP server.

## Troubleshoot access issues

### PMM Dump is missing from the Help Center

Verify that your account has the Admin role or Grafana Admin privileges. Editor and Viewer roles without Grafana Admin cannot access this feature.

### Dump creation fails with "Failed to compose meta error"

Enable the **Ignore load** option on the **PMM Dump > Export new datasheet** page before creating the dataset.

### Access denied messages appear

Your account lacks the permissions required for PMM Dump. Contact your administrator to request access.

### Password prompt appears when importing a non-encrypted dump

Pass the `--no-encryption` flag to the `pmm-dump` CLI tool to skip the password prompt.
