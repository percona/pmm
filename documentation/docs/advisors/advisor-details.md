## List of database Advisors
 
Percona Monitoring and Management (PMM) offers four categories of database Advisors to help you improve database performance: Configuration, Performance, Query and Security Advisors.

Each Advisor includes a set of automated checks, which investigate a specific range of possible issues and areas of improvement: security threats, non-compliance issues, performance degradation, query and index optimization strategies etc.

This page presents the complete list of database Advisors along with the corresponding subscription tier for which they are available.

You can also access this list through the [**Advisor checks for PMM**](https://portal.percona.com/advisors) section in the Percona Portal documentation, as the Advisors are hosted on the Percona Platform. PMM Server automatically downloads them from this source when the Advisors and Telemetry options are enabled in PMM under **Configuration > Settings > Advanced Settings**. Both options are enabled by default.

### Configuration Advisors

| Advisor Name | Description | Subscription | Database Technology|
| :--------- | :---------- | :--- |:--- |
| **Version Configuration** | Notifies of newly released database versions to streamline database maintenance and ensure the most up-to-date performance. |All Users | MySQL, MongoDB, PostgreSQL|
| **Generic Configuration** | Provides basic recommendations for improving your database configuration.   | All Users |MySQL, MongoDB, PostgreSQL| 
| **Resources Configuration** | Watches your database and gives you recommendations for efficient management of resources like binaries architecture, CPU number versus DB Configuration, etc. | All Users | MySQL, MongoDB|
| **Connection Configuration** |Provides recommendations on configuring database connection parameters for improving database performance.  | Customers only |MySQL, MongoDB, PostgreSQL|
| **Replication Configuration** | Provides recommendations for scalable replication in database clusters. | Customers only | MySQL, MongoDB|
| **InnoDB Configuration** | Advises on configuring InnoDB optimization for high performance. | Customers only | MySQL|
| **Vacuum Configuration** | Provides recommendations on optimizing Vacuum operations. | Customers only | PostgreSQL|

### Performance Advisors

| Advisor Name | Description | Subscription | Database Technology|
| :--------- | :---------- | :--- |:--- |
| **Generic Performance** | Provides basic database configuration recommendations for high-performance query execution. | All Users | MongoDB, PostgreSQL|
| **Vacuum Performance** | Helps improve the efficiency and execution speed of database Vacuum commands. |  Customers only | PostgreSQL|
| **Replication Performance** |Checks efficient replication usage of your database. | Customers only | MongoDB, PostgreSQL|

### Security Advisors

| Advisor Name | Description | Subscription | Database Technology|
| :--------- | :---------- | :--- |:--- |
| **CVE Security** | Informs you of any database versions affected by CVE. |All Users | MongoDB, PostgreSQL |
| **Configuration Security** | Checks your database configuration to ensure that security best practices are correctly implemented.  | All Users |MySQL, MongoDB, PostgreSQL|
| **Authentication Security** | Ensures that all database authentication parameters are configured securely. | Customers only |MySQL, MongoDB, PostgreSQL|
| **Replication Security** | Helps safeguard data replication by assessing security risks and providing recommendations for improving protection. | Customers only |MySQL|
| **Connection Security** | Helps identify security issues on network connections and provides recommendations for enhancing security. | Customers only |MySQL, MongoDB|

### Query Advisors

| Advisor Name | Description | Subscription | Database Technology|
| :--------- | :---------- | :--- |:--- |
| **Index Query** | Provides query and index optimization strategies for peak database performance. | Customers only | MySQL, MongoDB, PostgreSQL |
| **Schema Design Query** | Helps create efficient database schemas by analyzing queries and offering suggestions for optimization. | All Users |MySQL|

## List of checks

Every Advisor consists of one or more Advisor checks. 
We have listed the checks and their details here.

### MongoDB

| Advisor| Check Name | Description | Summary |
| :--------- | :---------- | :--- |:--- |
|Connection Configuration| mongodb\_connection\_sudden_spike | Warns about any significant increase in the number of connections exceeding 50% of the recent or typical connection count. | MongoDB Sudden Increase in Connection Count |
|Connection Configuration| mongodb_connections | Returns the current number of connections as an informational notice when connection counts exceed 5000. | MongoDB High Connections |
| Generic Configuration | mongo\_cache\_size | Warns when Mongo wiredtiger cache size is greater than the default 50%. | Mongo Storage Cache |
| Generic Configuration | mongodb\_active\_vs\_available\_connections | Warns if the ratio between active and available connections is higher than 75%. | MongoDB Active vs Available Connections |
| Generic Configuration | mongodb_journal | Warns if the journal is disabled. | MongoDB Journal |
| Generic Configuration | mongodb_loglevel | Warns if MongoDB is not using the default Log level. | MongoDB Non-Default Log Level |
| Generic Configuration | mongodb\_read\_tickets | Warns if MongoDB is using more than 128 read tickets. | MongoDB Read Tickets |
| Generic Configuration | mongodb\_write\_tickets | Warns if MongoDB is using more than 128 write tickets. | MongoDB Write Tickets |
| Generic Configuration | mongodb\_write\_tickets_runtime | Warns if MongoDB is using more than 128 write tickets during runtime. | MongoDB - Configuration Write Ticket Check |
| Replication Configuration| mongodb\_psa\_architecture_check | Raises an error if the replicaSet is utilizing a PSA (Primary-Secondary-Arbiter) architecture.| MongoDB PSA Architecture |
| Replication Configuration| mongodb\_replicaset\_topology | Warns if the Replica Set has less than three data-bearing nodes.| MongoDB Replica Set Topology |
| Resources Configuration| mongodb\_collection\_fragmented | Warns if the storage size exceeds the data size of a collection, indicating potential fragmentation. This suggests the need for compaction or an initial sync to reclaim disk space.| MongoDB Collections Fragmented |
| Resources Configuration| mongodb_cpucores | Warns if the number of CPU cores does not meet the minimum recommended requirements according to best practices. | MongoDB CPU Cores |
| Resources Configuration| mongodb\_dbpath\_mount | Warns if dbpath does not have a dedicated mount point. | MongoDB - Separate Mount Point Other Than "/" Partition for dbpath. |
| Resources Configuration| mongodb\_fcv\_check | Warns if there is a mismatch between the MongoDB version and the internal FCV (Feature Compatibility Version) parameter setting. | MongoDB - FCV Mismatch |
| Resources Configuration| mongodb_maxsessions | Warns if MongoDB is configured with a maxSessions value other than the default value of 1000000.| MongoDB maxSessions |
| Resources Configuration| mongodb\_swap\_allocation | Warns if there is no swap memory allocated to your instance. | MongoDB - Allocate Swap Memory |
| Resources Configuration| mongodb_taskexecutor | Warns if the count of MongoDB TaskExecutorPoolSize exceeds the number of available CPU cores. | MongoDB TaskExecutorPoolSize High |
| Resources Configuration| mongodb\_xfs\_ftype | Warns if dbpath is not using the XFS filesystem type.| MongoDB - XFS |
| Version Configuration| mongodb_EOL | Raises an error or a warning if your current PSMDB or MongoDB version has reached or is nearing its End-of-Life (EOL) status. | MongoDB Version EOL |
| Version Configuration| mongodb\_unsupported\_version | Raises an error if your current PSMDB or MongoDB version is not supported. | MongoDB Unsupported Version |
| Version Configuration| mongodb_version | Provides information on current MongoDB or Percona Server for MongoDB versions used in your environment. It also offers details on other available minor or major versions that you may consider for upgrades. | MongoDB Version Check |
| Generic Performance| mongodb\_multiple\_services | Warns if multiple mongod services are detected running on a single node. | MongoDB - Multiple mongod Services |
| Replication Performance| mongodb\_chunk\_imbalance | Warns if the distribution of chunks across shards is imbalanced.| MongoDB Sharding - Chunk Imbalance Across Shards |
| Replication Performance| mongodb\_oplog\_size_recommendation |Warns if the oplog window is below a 24-hour period and provides a recommended oplog size based on your instance. | MongoDB - Oplog Recovery Window is Low |
| Replication Performance| mongodb\_replication\_lag | Warns if the replica set member lags behind the primary by more than 10 seconds. | MongoDB Replication Lag |
| Index Query| mongodb\_shard\_collection\_inconsistent\_indexes | Warns if there are inconsistent indexes across shards for sharded collections. Missing or inconsistent indexes across shards can have a negative impact on performance. | MongoDB Sharding - Inconsistent Indexes Across Shards |
| Index Query| mongodb\_unused\_index | Warns if there are unused indexes on any database collection in your instance. This requires enabling the "indexStats" collector. | MongoDB - Unused Indexes |
| Authentication Security| mongodb_auth | Warns if MongoDB authentication is disabled. | MongoDB Authentication |
| Authentication Security| mongodb\_localhost\_auth_bypass | Warns if MongoDB localhost bypass is enabled. | MongoDB localhost authentication bypass enabled |
| Configuration Security| mongodb\_authmech\_scramsha256 | Warns if MongoDB is not using the default SHA-256 hashing function as its SCRAM authentication method. | MongoDB Security AuthMech Check |
| Connection Security| mongodb_bindip | Warns if the MongoDB network binding is not set as Recommended. | MonogDB IP Bindings |
| CVE Security| mongodb\_cve\_version | Shows an error if MongoDB or Percona Server for MongoDB version is older than the latest version containing CVE (Common Vulnerabilities and Exposures) fixes. | MongoDB CVE Version |

### MySQL

| Advisor| Check Name | Description | Summary |
| :--------- | :---------- | :--- |:--- |
|Connection Configuration| mysql\_configuration\_max\_connections\_usage |Checks the MySQL max_connections configuration option to ensure maximum utilization is achieved.| Check Max Connections Usage |
| Generic Configuration | mysql\_automatic\_sp\_privileges\_enabled | Checks if the automatic\_sp\_privileges configuration is ON. | Checks if automatic\_sp\_privileges configuration is ON. |
| Generic Configuration | mysql\_config\_binlog\_retention\_period | Checks whether binlogs are being rotated too frequently, which is not recommended, except in very specific cases. | Binlogs Retention Check |
| Generic Configuration | mysql\_config\_binlog\_row\_image | Advises when to set binlog\_row\_image=FULL. | Binlogs Raw Image is Not Set to FULL |
| Generic Configuration | mysql\_config\_binlogs_checksummed | Advises when to set binlog_checksum=CRC32 to improve consistency and reliability. | Server is Not Configured to Enforce Data Integrity |
| Generic Configuration | mysql\_config\_general_log | Checks whether the general log is enabled. | General Log is Enabled |
| Generic Configuration | mysql\_config\_log_bin | Checks whether the binlog is enabled or disabled. | Binary Log is disabled |
| Generic Configuration | mysql\_config\_sql_mode | Checks whether the server has specific values configured in sql_mode to ensure maximum data integrity. | Server is Not Configured to Enforce Data Integrity |
| Generic Configuration | mysql\_config\_tmp\_table\_size_limit | Checks whether the size of temporary tables exceeds the size of heap tables.| Temp Table Size is Larger Than Heap Table Size |
| Generic Configuration | mysql\_configuration\_log_verbosity | Checks whether warnings are being printed on the log. | Check Log Verbosity |
| Generic Configuration | mysql\_test\_database | Notifies if there are database named 'test' or 'test_%'. | MySQL Test Database |
| Generic Configuration | mysql_timezone | Verifies whether the time zone is correctly loaded.| MySQL configuration check |
| InnoDB Configuration| innodb\_redo\_logs\_not\_sized_correctly | Reviews the InnoDB redo log size and provides suggestions if it is configured too low. | InnoDB Redo Log Size is Not Configured Correctly. |
| InnoDB Configuration| mysql\_ahi\_efficiency\_performance\_basic_check | Checks the efficiency and effectiveness of InnoDB's Adaptive Hash Index (AHI). | InnoDB Adaptive Hash Index (AHI) Efficiency |
| InnoDB Configuration| mysql\_config\_innodb\_redolog\_disabled | Warns when the MySQL InnoDB Redo log is set to OFF, which poses a significant security risk and compromises data integrity. The MySQL InnoDB Redo log is a crucial component for maintaining the ACID (Atomicity, Consistency, Isolation, Durability) properties in MySQL databases. | Redo Log is Disabled in This Instance |
| InnoDB Configuration| mysql\_configuration\_innodb\_file\_format | Verifies whether InnoDB is configured with the recommended file format. | MySQL InnoDB File Format |
| InnoDB Configuration| mysql\_configuration\_innodb\_file\_maxlimit | Checks whether InnoDB is configured with the recommended auto-extend settings. | InnoDB Tablespace Size Has a Maximum Limit. |
| InnoDB Configuration| mysql\_configuration\_innodb\_file\_per\_table\_not_enabled | Warns when innodb\_file\_per_table is not enabled. | innodb\_file\_per_table Not Enabled |
| InnoDB Configuration| mysql\_configuration\_innodb\_flush\_method | Checks whether InnoDB is configured with the recommended flush method. | MySQL InnoDB Flush Method |
| InnoDB Configuration| mysql\_configuration\_innodb\_strict\_mode | Warns about password lifetime. | InnoDB strict mode |
| Replication Configuration| mysql\_config\_relay\_log\_purge | Identifies whether a replica node has relay-logs purge set.| Automatic Relay Log Purging is OFF |
| Replication Configuration| mysql\_config\_replication_bp1 | Identifies whether a replica node is in read-only mode and if *checksum* is enabled. | Checks Basic Best Practices When Setting Replica Node. |
| Replication Configuration| mysql\_config\_slave\_parallel\_workers | Identifies whether replication is single-threaded.| Replication is Single-Threaded |
| Replication Configuration| mysql\_config\_sync_binlog | Checks whether the binlog is synchronized before a transaction is committed. | Sync Binlog Disabled |
| Replication Configuration| mysql\_log\_replica_updates | Checks if a replica is safely logging replicated transactions. | MySQL Configuration Check |
| Replication Configuration| replica\_running\_skipping\_errors\_or\_idempotent\_mode | Reviews replication status to check if it is configured to skip errors or if the slave\_exec\_mode is set to be *idempotent*. | Replica is skipping errors or slave\_exec\_mode is Idempotent. |
| Resources Configuration| mysql\_32binary\_on_64system | Notifies if version\_compile\_machine equals i686. | Check if Binaries are 32 Bits |
| Version Configuration| mysql\_unsupported\_version_check | Warns against an unsupported Mysql version. | Checks Mysql Version |
| Version Configuration| mysql_version | Warns if MySQL, Percona Server for MySQL, or MariaDB version is not the latest available one. | MySQL Version |
| Version Configuration| mysql\_version\_eol_57 | Checks if the server version is EOL. | End Of Life Server Version (5.7). |
| Index Query| mysql\_performance\_temp\_ondisk\_table_high | Warns if there are too many on-disk temporary tables being created due to unoptimized query execution. | Too Many on Disk Temporary Tables |
| Index Query| mysql\_tables\_without_pk | Checks tables without primary keys. | MySQL check for a table without Primary Key |
| Schema Design Query | mysql\_indexes\_larger | Check all the tables to see if any have indexes larger than data. This indicates a sub-optimal schema and should be reviewed. |Tables With Index Sizes Larger Than Data |
| Authentication Security| mysql\_automatic\_expired_password | Warns if the MySQL parameter for automatic password expiry is not active. | MySQL Automatic User Expired Password |
| Authentication Security| mysql\_security\_anonymous_user | Verifies if anonymous users are present, as this would contradict security best practices.| Anonymous User (You Must Remove Any Anonymous User) |
| Authentication Security| mysql\_security\_open\_to\_world_host | Checks whether host definitions are set as '%' since this is overly permissive and could pose security risks. | UserS Have Host Definition '%' Which is Too Open |
| Authentication Security| mysql\_security\_root\_not\_local | Checks whether the root user has a host definition that is not set to 127.0.0.1 or localhost.| Root User Can Connect From Non-local Location |
| Authentication Security| mysql\_security\_user_ssl | Reports users who are not using a secure SSL protocol to connect.| Users Not Using Secure SSL |
| Authentication Security| mysql\_security\_user\_super\_not_local | Reports users with super privileges who are not connecting from the local host or the host is not fully restricted (e.g., 192.168.%). | Users have Super privileges With Remote and Too Open Access |
| Authentication Security| mysql\_security\_user\_without\_password | Reports users without passwords. | Users Without Password |
| Configuration Security| mysql\_config\_local_infile | Checks if the "LOAD DATA INFILE" functionality is active.| Load Data in File Active |
| Configuration Security| mysql\_configuration\_secure\_file\_priv_empty | Warns when  secure\_file\_priv is empty as this enables users with FILE privilege to create files at any location where MySQL server has Write permission. | secure\_file\_priv is Empty |
| Configuration Security| mysql\_password\_expiry |Checks if MySQL user passwords are expired or expiring within the next 30 days. | Check MySQL User Password Expiry |
| Configuration Security| mysql\_require\_secure_transport | Checks the status of *mysql_secure_transport_only*. | MySQL configuration check |
| Configuration Security| mysql\_security\_password_lifetime |Warns about password lifetime. | InnoDB Password Lifetime |
| Configuration Security| mysql\_security\_password_policy | Checks for password policy. | MySQL Security Check for Password |
| Connection Security| mysql\_private\_networks_only | Notifies about MySQL accounts that are allowed to connect from public networks. | MySQL Users With Granted Public Networks Access |
| Replication Security| mysql\_replication\_grants | Checks if replication is configured on a node without user grants.| MySQL Security Check for Replication User |
| Replication Security| mysql\_security\_replication\_grants\_mixed | Checks if replication privileges are mixed with more elevated privileges. | Replication Privileges |

### PostgreSQL

| Advisor| Check Name | Description | 
| :--------- | :---------- | :--- |
|Connection Configuration| postgresql\_max\_connections_1 | Notifies if the *max_connections* configuration option is set to a high value (above 300). PostgreSQL doesn't cope well with having many connections even if they are idle. The recommended value is below 300. |
| Generic Configuration | postgresql\_archiver\_failing_1 | Verifies if the archiver has failed. |
| Generic Configuration | postgresql\_fsync\_1 | Returns an error if the *fsync* configuration option is set to OFF, as this can lead to database corruptions. |
| Generic Configuration | postgresql\_log\_checkpoints_1 | Notifies if the *log_checkpoints* configuration option is not enabled. It is recommended to enable the logging of checkpoint information, as that provides a lot of useful information with almost no drawbacks. |
| Generic Configuration | postgresql\_logging\_recommendation_checks | Verifies whether the recommended minimum logging features are enabled.|
| Generic Configuration | postgresql\_wal\_retention_check | Checks if there are too many WAL files retained in the WAL directory. |
| Vacuum Configuration| postgresql\_log\_autovacuum\_min\_duration_1 | Notifies if the *log\_autovacuum\_min_duration configuration* option is set to -1 (disabled). It is recommended to enable the logging of autovacuum run information, as it provides a lot of useful information with almost no drawbacks. |
| Vacuum Configuration| postgresql\_table\_autovac_settings | Returns tables where autovacuum parameters are specified along with the corresponding autovacuum settings.|
| Vacuum Configuration| postgresql\_txid\_wraparound_approaching | Verifies the age of databases and alerts if the transaction ID wraparound issue is nearing. |
| Vacuum Configuration| postgresql\_vacuum\_sanity_check | This performs a quick check of some vacuum parameters. |
| Version Configuration| postgresql\_eol\_check |Checks if the currently installed PostgreSQL version has reached its EOL and is no longer supported. |
| Version Configuration| postgresql\_extension\_check | Lists outdated extensions with newer versions available. |
| Version Configuration| postgresql\_unsupported\_check | Verifies if the currently installed version is supported by Percona. |
| Version Configuration| postgresql\_version\_check | Checks if the currently installed version is outdated for its release level. |
| Generic Performance| postgresql\_cache\_hit\_ratio\_1 |Checks the hit ratio of one or more databases and raises a complaint when they are too low. |
| Generic Performance| postgresql\_config\_changes\_need\_restart_1 | Warns if there are any settings or configurations that have been changed and require a server restart or reload.|
| Generic Performance| postgresql\_tmpfiles\_check | Reports the number of temporary files and the number of bytes written to disk since the last statistics reset.|
| Replication Performance| postgresql\_stale\_replication\_slot\_1 | Warns if there is a stale replication slot. Stale replication slots will lead to WAL file accumulation and can result in a database server outage. |
| Vacuum Performance| postgresql\_table\_bloat_bytes | Verifies the size of the table bloat in bytes across all databases and raises alerts accordingly.|
| Vacuum Performance| postgresql\_table\_bloat\_in\_percentage | Verifies the size of the table bloat in the percentage of the total table size and alerts accordingly. |
| Index Query| postgresql\_number\_of\_index\_check | Lists relations with more than ten indexes. |
| Index Query| postgresql\_sequential\_scan_check | Checks for tables with excessive sequential scans. |
| Index Query| postgresql\_unused\_index_check | Lists relations with indexes that have not been used since the statistics were last reset. |
| Authentication Security| postgresql\_super\_role | Notifies if there are users with Superuser role. |
| Configuration Security| postgresql\_expiring\_passwd_check |Checks for passwords that are expiring and displays the time left before they expire. |
| CVE Security| postgresql\_cve\_check | Checks if the currently installed version has reported security vulnerabilities. |
