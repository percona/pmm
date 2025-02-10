# Data handling in PMM


The following questions are being answered related to personal and confidential data handling in PMM:
{.power-number}

1. Which type of data is transmitted?

      |**Data collection source**                                       | **Data collected** |
      | --------------------------------------------------------------- | ------------------------------------------------------
      | DB host to PMM                                                  | Database performance metrics <br/> SQL query examples for query analytics (optional).
      | PMM to DB Host                                                  | DSN and credentials for database access. A separate DB user is used (limited access) to retrieve metrics from the database.
      | DB Host to S3 compatible storage location                       | Database backup - optional if PMM Administrator configures it with Public Cloud (AWS, GCP, etc) as a possible storage location.
      | PMM Server to Percona Cloud                                     | Telemetry data is collected. </br/> PMM Server collects varying amounts of data from version to version, and no personal or confidential information is collected. See [Telemetry](../configure-pmm/advanced_settings#telemetry) for details on the data being transmitted.


2. Where is the data obtained from the DB host transmitted?

    All data gathered from the DB Host is transmitted to the PMM Server. It is possible to transmit DB backups to Cloud S3 storage (optional). 

    Telemetry data is sent to Percona Cloud. This does not contain any sensitive or personally identifiable information.

3. What is the purpose and nature of data processing?

    As per our [Privacy Policy](https://www.percona.com/privacy-policy), the data collection purposes are to provide the services and product enhancements.

    Although, PMM does not collect nor transfer personal data explicitly, in case query analytics is enabled and query examples collection is not disabled, we gather SQL query examples with real data and personal data may appear there if it is stored in DB.  All QAN data always remains within the PMM Server, and is never transmitted anywhere else.

4. What is the frequency and volume of processed data?

    By default, metrics data is gathered every 5, 10 or 60 minutes. In case Query Analytics is enabled and SQL query examples are gathered every minute, we don't use any special processing for personal or confidential data. PMM Server has no clue about the meaning of the data inside the SQL query.
    
    So it is processed as usual, which is to store inside the PMM Server and present on the PMM UI by request.

    Other than email addresses for Grafana users, PMM does not directly ask or collect any other personal data. For more information about the telemetry data that is collected, please refer to the [Percona Privacy Policy](http://www.percona.com/privacy-policy/). 

5. What applications or third parties can access the data created and processed by the cloud service?

    Third parties or other applications are not able to access the data gathered by the PMM Server.


6. Is Personal Data processed for other applications or parties, and should the data that is processed in the cloud service be available to other applications or 3rd parties?

    PMM Server doesn't pass any gathered, personal or confidential data to any third party or other applications nor to Percona Cloud.

7. How safe is the encryption? 

    It's a must to encrypt all connections to and from the cloud including the data in the cloud storage and PMM does so by default. 

    We use TLS (v1.2 at least) for connections between:

    - Database host to PMM Server (optionally, depending on user configuration)

    - PMM Server to Percona Cloud
    - PMM Server to remote database (optionally, depending on user configuration)
    - End-user to PMM Server web interface/api (self-signed by default)

    For more information about Percona security posture, please refer to our [Trust Center here](https://trust.percona.com/).

