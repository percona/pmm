# List of advisors and checks

PMM ships with built-in Advisor checks, grouped into advisors by category. Each check targets one database technology: MySQL, PostgreSQL or MongoDB. To browse the checks in PMM, change their interval or turn them off, go to **Advisors > Catalog**. To add checks of your own, see [Developing Advisor checks](develop-advisor-checks.md).

All checks run at the **Standard** interval by default, except `mongodb_collection_fragmented` and `mongodb_dbpath_mount`, which run at the **Rare** interval, and `mongodb_cve_2025_14847_zlib`, which runs at the **Frequent** interval.

## Categories

The following table shows how many checks each category has for each technology:

| Category | MySQL | PostgreSQL | MongoDB |
| :------- | :---: | :--------: | :-----: |
| [Connections](#connections) | 1 | 1 | 4 |
| [Durability](#durability) | 9 | 2 | 1 |
| [Logging](#logging) | 2 | 3 | 1 |
| [Maintenance](#maintenance) | — | 6 | 1 |
| [Performance](#performance) | 6 | 2 | 3 |
| [Replication](#replication) | 5 | 1 | 5 |
| [Resources](#resources) | 3 | 1 | 7 |
| [Schema and indexes](#schema-and-indexes) | 2 | 3 | 2 |
| [Security](#security) | 18 | 3 | 6 |
| [Versions](#versions) | 3 | 4 | 4 |

## Connections

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_active_vs_available_connections` | MongoDB | MongoDB Active vs Available Connections | Warns if the ratio between active and available connections is higher than 75%. |
| `mongodb_connection_sudden_spike` | MongoDB | MongoDB - sudden increase in connection count | Warns about any significant increase in the number of connections exceeding 50% of the recent or typical connection count. |
| `mongodb_connections` | MongoDB | MongoDB High connections | Returns the current number of connections as a warning when connection counts exceed 5000. |
| `mongodb_maxsessions` | MongoDB | MongoDB maxSessions | Warns if MongoDB is configured with a maxSessions value other than the default value of 1000000. |
| `mysql_configuration_max_connections_usage` | MySQL | Check max connections usage | Checks the MySQL max_connections configuration option to ensure maximum utilization is achieved. |
| `postgresql_max_connections` | PostgreSQL | PostgreSQL max_connections too high | Notifies if the *max_connections* configuration option is set to a high value (above 300). PostgreSQL doesn't cope well with having many connections even if they are idle. The recommended value is below 300. |

## Durability

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_journal` | MongoDB | MongoDB Journal | Warns if the journal is disabled. |
| `mysql_config_binlog_retention_period` | MySQL | Binlogs retention check | Checks whether binlogs are being rotated too frequently, which is not recommended, except in very specific cases. |
| `mysql_config_binlog_row_image` | MySQL | Binlogs raw image is not set to FULL | Advises when to set binlog\_row\_image=FULL. |
| `mysql_config_binlogs_checksummed` | MySQL | Binlogs are not checksummed | Advises when to set binlog_checksum=CRC32 to improve consistency and reliability. |
| `mysql_config_innodb_redolog_disabled` | MySQL | Redo log is disabled in this instance | Warns when the MySQL InnoDB Redo log is set to OFF, which poses a significant security risk and compromises data integrity. The MySQL InnoDB Redo log is a crucial component for maintaining the ACID (Atomicity, Consistency, Isolation, Durability) properties in MySQL databases. |
| `mysql_config_log_bin` | MySQL | Binary Log is disabled | Checks whether the binlog is enabled or disabled. |
| `mysql_config_sql_mode` | MySQL | Server is not configured to enforce data integrity | Checks whether the server has specific values configured in sql_mode to ensure maximum data integrity. |
| `mysql_config_sync_binlog` | MySQL | Sync binlog is disabled | Checks whether the binlog is synchronized before a transaction is committed. |
| `mysql_configuration_innodb_strict_mode` | MySQL | InnoDB strict mode | Warns if `innodb_strict_mode` is disabled. |
| `mysql_timezone` | MySQL | MySQL time zone tables are not loaded | Verifies whether the time zone is correctly loaded. |
| `postgresql_archiver_failing` | PostgreSQL | PostgreSQL Archiver is failing | Verifies if the archiver has failed. |
| `postgresql_fsync` | PostgreSQL | PostgreSQL fsync is set to off | Returns an error if the *fsync* configuration option is OFF, as this can lead to database corruption. |

## Logging

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_loglevel` | MongoDB | MongoDB Non-Default Log Level | Warns if MongoDB is not using the default Log level. |
| `mysql_config_general_log` | MySQL | General Log is enabled | Checks whether the general log is enabled. |
| `mysql_configuration_log_verbosity` | MySQL | Check log verbosity | Checks whether warnings are being printed on the log. |
| `postgresql_log_autovacuum_min_duration` | PostgreSQL | PostgreSQL Autovacuum Logging Is Disabled | Notifies if the *log\_autovacuum\_min_duration configuration* option is set to -1 (disabled). It is recommended to enable the logging of autovacuum run information, as it provides a lot of useful information with almost no drawbacks. |
| `postgresql_log_checkpoints` | PostgreSQL | PostgreSQL Checkpoints Logging Disabled | Notifies if the *log_checkpoints* configuration option is not enabled. It is recommended to enable the logging of checkpoint information, as that provides a lot of useful information with almost no drawbacks. |
| `postgresql_logging_recommendation_checks` | PostgreSQL | Check for minimal logging | Verifies whether the recommended minimum logging features are enabled. |

## Maintenance

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_collection_fragmented` | MongoDB | MongoDB Collections Fragmented | Warns if the storage size exceeds the data size of a collection, indicating potential fragmentation. This suggests the need for compaction or an initial sync to reclaim disk space. |
| `postgresql_config_changes_need_restart` | PostgreSQL | Configuration change requires restart/reload | Warns if there are any settings or configurations that have been changed and require a server restart or reload. |
| `postgresql_table_autovac_settings` | PostgreSQL | Check whether there is any table level autovacuum settings | Returns tables where autovacuum parameters are specified along with the corresponding autovacuum settings. |
| `postgresql_table_bloat_bytes` | PostgreSQL | Check amount of bloat in tables if greater than 250MB | Verifies the size of the table bloat in bytes across all databases and raises alerts accordingly. |
| `postgresql_table_bloat_in_percentage` | PostgreSQL | PostgreSQL Table Bloat in percentage of the table size | Verifies the size of the table bloat in the percentage of the total table size and alerts accordingly. |
| `postgresql_txid_wraparound_approaching` | PostgreSQL | PostgreSQL Transaction ID Wraparound approaching | Verifies the age of databases and alerts if the transaction ID wraparound issue is nearing. |
| `postgresql_vacuum_sanity_check` | PostgreSQL | PostgreSQL vacuum setting quick check | This performs a quick check of some vacuum parameters. |

## Performance

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_read_tickets` | MongoDB | MongoDB Read Tickets | Warns if MongoDB is using more than 128 read tickets. |
| `mongodb_write_tickets` | MongoDB | MongoDB write Tickets | Warns if the MongoDB startup options set more than 128 write tickets or, starting with MongoDB 7.0, any fixed number of write tickets. |
| `mongodb_write_tickets_runtime` | MongoDB | MongoDB Configuration Write ticket Check | Warns if MongoDB is using more than 128 write tickets during runtime or, starting with MongoDB 7.0, any fixed number of write tickets. |
| `mysql_ahi_efficiency_performance_basic_check` | MySQL | InnoDB Adaptive Hash Index (AHI) efficiency checker | Checks the efficiency and effectiveness of InnoDB's Adaptive Hash Index (AHI). |
| `mysql_config_tmp_table_size_limit` | MySQL | Temp table size is larger than Heap Table size | Checks whether the size of temporary tables exceeds the size of heap tables. |
| `mysql_configuration_innodb_file_format` | MySQL | MySQL InnoDB file format | Verifies whether InnoDB is configured with the recommended file format. |
| `mysql_configuration_innodb_flush_method` | MySQL | MySQL InnoDB flush method | Checks whether InnoDB is configured with the recommended flush method. |
| `mysql_innodb_redo_logs_not_sized_correctly` | MySQL | Checks if InnoDB redo log size is not configured correctly | Reviews the InnoDB redo log size and provides suggestions if it is configured too low. |
| `mysql_performance_temp_ondisk_table_high` | MySQL | Too many on-disk temporary tables | Warns if there are too many on-disk temporary tables being created due to unoptimized query execution. |
| `postgresql_cache_hit_ratio` | PostgreSQL | PostgreSQL cache hit ratio | Checks the hit ratio of one or more databases and raises a complaint when they are too low. |
| `postgresql_tmpfiles_check` | PostgreSQL | PostgreSQL temporary file statistics | Reports the number of temporary files and the number of bytes written to disk since the last statistics reset. |

## Replication

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_balancer` | MongoDB | MongoDB Balancer is disabled | Warns if the balancer process is disabled. |
| `mongodb_oplog_size_recommendation` | MongoDB | MongoDB Oplog Recovery Window Low | Warns if the oplog window is below a 24-hour period and provides a recommended oplog size based on your instance. |
| `mongodb_psa_architecture_check` | MongoDB | MongoDB PSA Architecture | Raises an error if the replicaSet is utilizing a PSA (Primary-Secondary-Arbiter) architecture. |
| `mongodb_replicaset_topology` | MongoDB | MongoDB Replica Set Topology | Warns if the Replica Set has less than three data-bearing nodes. |
| `mongodb_replication_lag` | MongoDB | MongoDB Replication Lag | Warns if the replica set member lags behind the primary by more than 10 seconds. |
| `mysql_config_relay_log_purge` | MySQL | Automatic relay log purging is off | Identifies whether a replica node has relay-logs purge set. |
| `mysql_config_replication_bp1` | MySQL | Checks for basic best practices when setting a replica node | Identifies whether a replica node is in read-only mode and if *checksum* is enabled. |
| `mysql_config_slave_parallel_workers` | MySQL | Replication is single threaded | Identifies whether replication is single-threaded. |
| `mysql_log_replica_updates` | MySQL | Replicated transactions are not logged | Checks if a replica is safely logging replicated transactions. |
| `mysql_replica_running_skipping_errors_or_idempotent_mode` | MySQL | Checks if replica configured is skipping errors or slave_exec_mode is idempotent | Reviews replication status to check if it is configured to skip errors or if the slave\_exec\_mode is set to be *idempotent*. |
| `postgresql_stale_replication_slot` | PostgreSQL | PostgreSQL Stale Replication Slot | Warns if there is a stale replication slot. Stale replication slots will lead to WAL file accumulation and can result in a database server outage. |

## Resources

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_cache_size` | MongoDB | Mongo Storage Cache | Warns when Mongo wiredtiger cache size is greater than the default 50%. |
| `mongodb_cpucores` | MongoDB | MongoDB CPU cores | Warns if the number of CPU cores does not meet the minimum recommended requirements according to best practices. |
| `mongodb_dbpath_mount` | MongoDB | MongoDB separate mount point other than "/" partition for dbpath | Warns if dbpath does not have a dedicated mount point. |
| `mongodb_multiple_services` | MongoDB | MongoDB - Multiple mongod services | Warns if multiple mongod services are detected running on a single node. |
| `mongodb_swap_allocation` | MongoDB | MongoDB - allocate swap memory | Warns if there is no swap memory allocated to your instance. |
| `mongodb_taskexecutor` | MongoDB | MongoDB TaskExecutorPoolSize High | Warns if the count of MongoDB TaskExecutorPoolSize exceeds the number of available CPU cores. |
| `mongodb_xfs_ftype` | MongoDB | MongoDB - xfs | Warns if dbpath is not using the XFS filesystem type. |
| `mysql_32binary_on_64system` | MySQL | Check if binaries are 32 bits | Notifies if version\_compile\_machine equals i686. |
| `mysql_configuration_innodb_file_maxlimit` | MySQL | InnoDB Tablespace size has a maximum limit | Checks whether InnoDB is configured with the recommended auto-extend settings. |
| `mysql_configuration_innodb_file_per_table_not_enabled` | MySQL | innodb_file_per_table not enabled | Warns when innodb\_file\_per_table is not enabled. |
| `postgresql_wal_retention_check` | PostgreSQL | Check for WAL file accumulation | Checks if there are too many WAL files retained in the WAL directory. |

## Schema and indexes

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_shard_collection_inconsistent_indexes` | MongoDB | MongoDB Inconsistent Indexes Across Shards | Warns if there are inconsistent indexes across shards for sharded collections. Missing or inconsistent indexes across shards can have a negative impact on performance. |
| `mongodb_unused_index` | MongoDB | MongoDB - Unused Indexes | Warns if there are unused indexes on any database collection in your instance. This requires enabling the "indexStats" collector. |
| `mysql_indexes_larger` | MySQL | Are there tables with index sizes larger than data? | Check all the tables to see if any have indexes larger than data. This indicates a sub-optimal schema and should be reviewed. |
| `mysql_tables_without_pk` | MySQL | MySQL check for table without Primary Key | Checks tables without primary keys. |
| `postgresql_number_of_index_check` | PostgreSQL | Check for relations with a high number of indexes | Lists relations with more than ten indexes. |
| `postgresql_sequential_scan_check` | PostgreSQL | PostgreSQL sequential scan check | Checks for tables with excessive sequential scans. |
| `postgresql_unused_index_check` | PostgreSQL | Check for relations that have unused indexes | Lists relations with indexes that have not been used since the statistics were last reset. |

## Security

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_auth` | MongoDB | MongoDB authentication | Warns if MongoDB authentication is disabled. |
| `mongodb_authmech_scramsha256` | MongoDB | MongoDB Security AuthMech Check | Warns if MongoDB is not using the default SHA-256 hashing function as its SCRAM authentication method. |
| `mongodb_bindip` | MongoDB | MonogDB IP bindings | Warns if the MongoDB network binding is not set as Recommended. |
| `mongodb_cve_2025_14847_zlib` | MongoDB | MongoDB Zlib Compression Heap Memory Vulnerability (CVE-2025-14847) | Checks whether MongoDB Server is vulnerable to CVE-2025-14847 with active Zlib compression. Fixed versions: 4.4.30+, 5.0.32+, 6.0.27+, 7.0.28+, 8.0.17+, 8.2.3+. End-of-life versions without a fix: 3.6.x, 4.0.x, 4.2.x. |
| `mongodb_cve_version` | MongoDB | MongoDB CVE Version | Shows an error if MongoDB or Percona Server for MongoDB version is older than the latest version containing CVE (Common Vulnerabilities and Exposures) fixes. |
| `mongodb_localhost_auth_bypass` | MongoDB | MongoDB localhost authentication bypass enabled | Warns if MongoDB localhost bypass is enabled. |
| `mysql_automatic_expired_password` | MySQL | MySQL Automatic User Expired Password | Warns if the MySQL parameter for automatic password expiry is not active. |
| `mysql_automatic_sp_privileges_enabled` | MySQL | Checks if automatic_sp_privileges configuration is ON | Checks if the automatic\_sp\_privileges configuration is ON. |
| `mysql_config_local_infile` | MySQL | Load data in file active | Checks if the "LOAD DATA INFILE" functionality is active. |
| `mysql_configuration_secure_file_priv_empty` | MySQL | secure_file_priv is empty | Warns when  secure\_file\_priv is empty as this enables users with FILE privilege to create files at any location where MySQL server has Write permission. |
| `mysql_password_expiry` | MySQL | Check MySQL user password expiry | Checks if MySQL user passwords are expired or expiring within the next 30 days. |
| `mysql_private_networks_only` | MySQL | MySQL Users With Granted Public Networks Access | Notifies about MySQL accounts that are allowed to connect from public networks. |
| `mysql_replication_grants` | MySQL | MySQL security check for replication user | Checks if replication is configured on a node without user grants. |
| `mysql_require_secure_transport` | MySQL | Secure transport is not required | Checks whether `require_secure_transport` is enabled. |
| `mysql_security_anonymous_user` | MySQL | Anonymous user (you must remove any anonymous user) | Verifies if anonymous users are present, as this would contradict security best practices. |
| `mysql_security_open_to_world_host` | MySQL | User(s) has/have host definition '%' which is too open | Checks whether host definitions are set as '%' since this is overly permissive and could pose security risks. |
| `mysql_security_password_lifetime` | MySQL | Password lifetime is not enforced | Warns about password lifetime. |
| `mysql_security_password_policy` | MySQL | MySQL security check for password | Checks for password policy. |
| `mysql_security_replication_grants_mixed` | MySQL | Replication privileges | Checks if replication privileges are mixed with more elevated privileges. |
| `mysql_security_root_not_local` | MySQL | Root user can connect from non local location | Checks whether the root user has a host definition that is not set to 127.0.0.1 or localhost. |
| `mysql_security_user_ssl` | MySQL | User(s) not using secure SSL protocol to connect | Reports users who are not using a secure SSL protocol to connect. |
| `mysql_security_user_super_not_local` | MySQL | Users have super privileges with remote or lack access restrictions | Reports users with super privileges who are not connecting from the local host or the host is not fully restricted (e.g., 192.168.%). |
| `mysql_security_user_without_password` | MySQL | User(s) without password | Reports users without passwords. |
| `mysql_test_database` | MySQL | MySQL test Database | Notifies if there are database named 'test' or 'test_%'. |
| `postgresql_cve_check` | PostgreSQL | Check for security vulnerabilities | Checks if the currently installed version has reported security vulnerabilities. |
| `postgresql_expiring_passwd_check` | PostgreSQL | Check for password expiration | Checks for passwords that are expiring and displays the time left before they expire. |
| `postgresql_super_role` | PostgreSQL | PostgreSQL Super Role | Notifies if there are users with Superuser role. |

## Versions

| Check name | Technology | Summary | Description |
| :--------- | :--------- | :------ | :---------- |
| `mongodb_EOL` | MongoDB | MongoDB version EOL | Raises an error or a warning if your current PSMDB or MongoDB version has reached or is nearing its End-of-Life (EOL) status. |
| `mongodb_fcv_check` | MongoDB | MongoDB - FCV mismatch | Warns if there is a mismatch between the MongoDB version and the internal FCV (Feature Compatibility Version) parameter setting. |
| `mongodb_unsupported_version` | MongoDB | MongoDB Unsupported version check | Raises an error if your current PSMDB or MongoDB version is not supported. |
| `mongodb_version` | MongoDB | MongoDB version check | Provides information on current MongoDB or Percona Server for MongoDB versions used in your environment. It also offers details on other available minor or major versions that you may consider for upgrades. |
| `mysql_unsupported_version_check` | MySQL | Checks mysql version for support | Warns against an unsupported Mysql version. |
| `mysql_version` | MySQL | MySQL Version | Warns if MySQL, Percona Server for MySQL, or MariaDB version is not the latest available one. |
| `mysql_version_eol_57` | MySQL | End Of Life server version (5.7) | Checks if the server version is EOL. |
| `postgresql_eol_check` | PostgreSQL | Check if PostgreSQL version is EOL | Checks if the currently installed PostgreSQL version has reached its EOL and is no longer supported. |
| `postgresql_extension_check` | PostgreSQL | Check for outdated extensions | Lists outdated extensions with newer versions available. |
| `postgresql_unsupported_check` | PostgreSQL | Check for unsupported PostgreSQL | Verifies if the currently installed version is supported by Percona. |
| `postgresql_version_check` | PostgreSQL | Check for newer version of PostgreSQL | Checks if the currently installed version is outdated for its release level. |
