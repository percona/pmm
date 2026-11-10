# List of advisors and checks

Use this list to find a built-in Advisor check and learn what it finds. Each check runs on one type of database: MySQL, PostgreSQL or MongoDB. To see how to fix what a check finds, click its summary. To check for something that no built-in check covers, [develop your own check](develop-advisor-checks.md).

The checks are grouped by the same categories that **Advisors > Catalog** uses: [Connections](#connections), [Durability](#durability), [Logging](#logging), [Maintenance](#maintenance), [Performance](#performance), [Replication](#replication), [Resources](#resources), [Schema & indexes](#schema-indexes), [Security](#security) and [Versions](#versions).

## Connections

Use these checks to catch connection limits that are set too high, and connection counts that approach the limit or spike suddenly, before clients start getting connection errors:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB Active vs Available Connections](checks/mongodb-active-vs-available-connections.md) | `mongodb_active_vs_available_connections` | MongoDB | Warns if the ratio between active and available connections is higher than 75%. |
| [MongoDB - sudden increase in connection count](checks/mongodb-connection-spike.md) | `mongodb_connection_sudden_spike` | MongoDB | Warns about any significant increase in the number of connections exceeding 50% of the recent or typical connection count. |
| [MongoDB High connections](checks/mongo-high-connections.md) | `mongodb_connections` | MongoDB | Returns the current number of connections as a warning when connection counts exceed 5000. |
| [MongoDB maxSessions](checks/mongodb-maxSessions.md) | `mongodb_maxsessions` | MongoDB | Warns if MongoDB is configured with a `maxSessions` value other than the default value of `1000000`. |
| [Check max connections usage](checks/mysql_configuration_max_connections_usage.md) | `mysql_configuration_max_connections_usage` | MySQL | Reports how close the highest number of connections came to the `max_connections` limit. |
| [PostgreSQL max_connections too high](checks/configuration-pg-high-max-connections.md) | `postgresql_max_connections` | PostgreSQL | Warns if `max_connections` is above 300. PostgreSQL handles many connections badly, even idle ones. |

## Durability

Use these checks to make sure that committed data survives a crash. They flag a disabled journal, redo log, binary log or fsync, and settings that weaken data integrity:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB Journal](checks/mongodb-journal.md) | `mongodb_journal` | MongoDB | Warns if the journal is disabled. |
| [Binlogs retention check](checks/binlog-retention-period.md) | `mysql_config_binlog_retention_period` | MySQL | Checks whether binlogs are being rotated too frequently, which is not recommended, except in very specific cases. |
| [Binlogs raw image is not set to FULL](checks/binlog-row-image.md) | `mysql_config_binlog_row_image` | MySQL | Advises when to set `binlog_row_image=FULL`. |
| [Binlogs are not checksummed](checks/binlog-checksum-value.md) | `mysql_config_binlogs_checksummed` | MySQL | Advises when to set `binlog_checksum=CRC32` to improve consistency and reliability. |
| [Redo log is disabled in this instance](checks/mysql-innodb-redolog-disabled.md) | `mysql_config_innodb_redolog_disabled` | MySQL | Warns if the InnoDB redo log is turned off, which puts data integrity at risk. |
| [Binary Log is disabled](checks/configuration-check-log-bin.md) | `mysql_config_log_bin` | MySQL | Checks whether the binlog is enabled or disabled. |
| [Server is not configured to enforce data integrity](checks/configuration-check-sql-mode.md) | `mysql_config_sql_mode` | MySQL | Checks whether the server has specific values configured in `sql_mode` to ensure maximum data integrity. |
| [Sync binlog is disabled](checks/configuration-check-sync-binlog.md) | `mysql_config_sync_binlog` | MySQL | Checks whether the binlog is synchronized before a transaction is committed. |
| [InnoDB strict mode](checks/innodb-strict-mode.md) | `mysql_configuration_innodb_strict_mode` | MySQL | Warns if `innodb_strict_mode` is disabled. |
| [MySQL time zone tables are not loaded](checks/time-zone-data-not-loaded.md) | `mysql_timezone` | MySQL | Verifies whether the time zone is correctly loaded. |
| [PostgreSQL Archiver is failing](checks/performance-pg-archiver-failing.md) | `postgresql_archiver_failing` | PostgreSQL | Verifies if the archiver has failed. |
| [PostgreSQL fsync is set to off](checks/configuration-pg-check-fsync-enabled.md) | `postgresql_fsync` | PostgreSQL | Returns an error if the `fsync` configuration option is `OFF`, as this can lead to database corruption. |

## Logging

Use these checks to make sure that your databases log enough to diagnose problems, without logging so much that the logs grow out of control:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB Non-Default Log Level](checks/mongodb-log-messages.md) | `mongodb_loglevel` | MongoDB | Warns if MongoDB is not using the default Log level. |
| [General Log is enabled](checks/configuration-check-general-log.md) | `mysql_config_general_log` | MySQL | Checks whether the general log is enabled. |
| [Check log verbosity](checks/mysql-warnings-not-being-logged.md) | `mysql_configuration_log_verbosity` | MySQL | Checks whether warnings are being printed on the log. |
| [PostgreSQL Autovacuum Logging Is Disabled](checks/configuration-pg-log-autovacuum-disabled.md) | `postgresql_log_autovacuum_min_duration` | PostgreSQL | Notifies if `log_autovacuum_min_duration` is set to `-1`, which turns off autovacuum logging. |
| [PostgreSQL Checkpoints Logging Disabled](checks/configuration-pg-log-checkpoints-disabled.md) | `postgresql_log_checkpoints` | PostgreSQL | Notifies if `log_checkpoints` is turned off. |
| [Check for minimal logging](checks/postgresql-logging-recommendations.md) | `postgresql_logging_recommendation_checks` | PostgreSQL | Verifies whether the recommended minimum logging features are enabled. |

## Maintenance

Use these checks to keep your databases healthy over time. They flag bloat, fragmentation, vacuum settings, an approaching transaction ID wraparound and settings that wait for a restart:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB Collections Fragmented](checks/mongodb-collections-fragmented.md) | `mongodb_collection_fragmented` | MongoDB | Warns if the storage size exceeds the data size of a collection, indicating potential fragmentation. To reclaim the disk space, compact the collection or run an initial sync. |
| [Configuration change requires restart/reload](checks/configuration-pg-settings-changed.md) | `postgresql_config_changes_need_restart` | PostgreSQL | Warns if there are any settings or configurations that have been changed and require a server restart or reload. |
| [Check whether there is any table level autovacuum settings](checks/postgresql-tables-per-table-vacuum-settings.md) | `postgresql_table_autovac_settings` | PostgreSQL | Returns tables where autovacuum parameters are specified along with the corresponding autovacuum settings. |
| [Check amount of bloat in tables if greater than 250MB](checks/postgresql-table-bloat-in-bytes.md) | `postgresql_table_bloat_bytes` | PostgreSQL | Verifies the size of the table bloat in bytes across all databases and raises alerts accordingly. |
| [PostgreSQL Table Bloat in percentage of the table size](checks/postgresql-table-bloat-in-percentage-of-table-size.md) | `postgresql_table_bloat_in_percentage` | PostgreSQL | Verifies the size of the table bloat in the percentage of the total table size and alerts accordingly. |
| [PostgreSQL Transaction ID Wraparound approaching](checks/postgresql-transaction-id-wraparound-is-approaching.md) | `postgresql_txid_wraparound_approaching` | PostgreSQL | Verifies the age of databases and alerts if the transaction ID wraparound issue is nearing. |
| [PostgreSQL vacuum setting quick check](checks/postgresql-vacuum-sanity-check.md) | `postgresql_vacuum_sanity_check` | PostgreSQL | Checks some key vacuum parameters. |

## Performance

Use these checks to find settings and workloads that slow down queries, such as ticket limits, a low cache hit ratio, temporary tables written to disk and InnoDB settings:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB Read Tickets](checks/mongodb-read-tickets.md) | `mongodb_read_tickets` | MongoDB | Warns if MongoDB is using more than 128 read tickets. |
| [MongoDB write Tickets](checks/mongodb-write-tickets.md) | `mongodb_write_tickets` | MongoDB | Warns if the MongoDB startup options set more than 128 write tickets or, starting with MongoDB 7.0, any fixed number of write tickets. |
| [MongoDB Configuration Write ticket Check](checks/mongodb-non-default-runtime-write-tickets.md) | `mongodb_write_tickets_runtime` | MongoDB | Warns if MongoDB is using more than 128 write tickets during runtime or, starting with MongoDB 7.0, any fixed number of write tickets. |
| [InnoDB Adaptive Hash Index (AHI) efficiency checker](checks/mysql-ahi-efficiency-performance-basic-check.md) | `mysql_ahi_efficiency_performance_basic_check` | MySQL | Checks the efficiency and effectiveness of InnoDB's Adaptive Hash Index (AHI). |
| [Temp table size is larger than Heap Table size](checks/configuration-check-tmp-table-size-limit.md) | `mysql_config_tmp_table_size_limit` | MySQL | Checks whether the size of temporary tables exceeds the size of heap tables. |
| [MySQL InnoDB file format](checks/configuration-innodb-file-format.md) | `mysql_configuration_innodb_file_format` | MySQL | Verifies whether InnoDB is configured with the recommended file format. |
| [MySQL InnoDB flush method](checks/configuration-innodb-flush-method.md) | `mysql_configuration_innodb_flush_method` | MySQL | Checks whether InnoDB is configured with the recommended flush method. |
| [Checks if InnoDB redo log size is not configured correctly](checks/innodb_redo_logs_not_sized_correctly.md) | `mysql_innodb_redo_logs_not_sized_correctly` | MySQL | Reviews the InnoDB redo log size and provides suggestions if it is configured too low. |
| [Too many on-disk temporary tables](checks/mysql_performance_temp_ondisk_table_high.md) | `mysql_performance_temp_ondisk_table_high` | MySQL | Warns if there are too many on-disk temporary tables being created due to unoptimized query execution. |
| [PostgreSQL cache hit ratio](checks/performance-pg-low-cache-hit-ratio.md) | `postgresql_cache_hit_ratio` | PostgreSQL | Checks the hit ratio of one or more databases and raises a complaint when they are too low. |
| [PostgreSQL temporary file statistics](checks/postgresql-tmpfiles-check.md) | `postgresql_tmpfiles_check` | PostgreSQL | Reports the number of temporary files and the number of bytes written to disk since the last statistics reset. |

## Replication

Use these checks to make sure that replicas stay in sync and can take over. They flag replication lag, risky replica settings, a short oplog window and fragile topologies:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB Balancer is disabled](checks/mongodb-chunk-imbalance.md) | `mongodb_balancer` | MongoDB | Warns if the balancer process is disabled. |
| [MongoDB Oplog Recovery Window Low](checks/mongodb-oplog-sizing.md) | `mongodb_oplog_size_recommendation` | MongoDB | Warns if the oplog window is below a 24-hour period and provides a recommended oplog size based on your instance. |
| [MongoDB PSA Architecture](checks/mongodb-psa-architecture.md) | `mongodb_psa_architecture_check` | MongoDB | Raises an error if the replica set is utilizing a PSA (Primary-Secondary-Arbiter) architecture. |
| [MongoDB Replica Set Topology](checks/mongo-check-the-replica-set-topology.md) | `mongodb_replicaset_topology` | MongoDB | Warns if the Replica Set has less than three data-bearing nodes. |
| [MongoDB Replication Lag](checks/mongo-replication-lag.md) | `mongodb_replication_lag` | MongoDB | Warns if the replica set member lags behind the primary by more than 10 seconds. |
| [Automatic relay log purging is off](checks/configuration-check-relay-log-purge.md) | `mysql_config_relay_log_purge` | MySQL | Identifies whether a replica node has relay-logs purge set. |
| [Checks for basic best practices when setting a replica node](checks/configuration-check-replica-not-read-only.md) | `mysql_config_replication_bp1` | MySQL | Identifies whether a replica node is in read-only mode and if `checksum` is enabled. |
| [Replication is single threaded](checks/sql-processing-not-multi-threaded.md) | `mysql_config_slave_parallel_workers` | MySQL | Identifies whether replication is single-threaded. |
| [Replicated transactions are not logged](checks/mysql-log-replica-updates.md) | `mysql_log_replica_updates` | MySQL | Checks if a replica is safely logging replicated transactions. |
| [Checks if replica configured is skipping errors or slave_exec_mode is idempotent](checks/replica_running_skipping_errors_or_idempotent_mode.md) | `mysql_replica_running_skipping_errors_or_idempotent_mode` | MySQL | Reviews replication status to check if it is configured to skip errors or if the `slave_exec_mode` is set to `IDEMPOTENT`. |
| [PostgreSQL Stale Replication Slot](checks/performance-pg-stale-replication-slot.md) | `postgresql_stale_replication_slot` | PostgreSQL | Warns if there is a stale replication slot. Stale replication slots will lead to WAL file accumulation and can result in a database server outage. |

## Resources

Use these checks to make sure that each server has the CPU, memory, swap, storage layout and file settings that the database needs:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [Mongo Storage Cache](checks/mongodb-high-wiredTiger-cache.md) | `mongodb_cache_size` | MongoDB | Warns when Mongo WiredTiger cache size is greater than the default 50%. |
| [MongoDB CPU cores](checks/mongodb-cpu-cores-check.md) | `mongodb_cpucores` | MongoDB | Warns if the number of CPU cores does not meet the minimum recommended requirements according to best practices. |
| [MongoDB separate mount point other than "/" partition for dbpath](checks/mongodb-dbpath-mount.md) | `mongodb_dbpath_mount` | MongoDB | Warns if `dbpath` does not have a dedicated mount point. |
| [MongoDB - Multiple mongod services](checks/multiple-mongod-running-in-a-node.md) | `mongodb_multiple_services` | MongoDB | Warns if multiple mongod services are detected running on a single node. |
| [MongoDB - allocate swap memory](checks/mongodb-swap-allocation.md) | `mongodb_swap_allocation` | MongoDB | Warns if there is no swap memory allocated to your instance. |
| [MongoDB TaskExecutorPoolSize High](checks/mongodb-task-executorPoolSize-high.md) | `mongodb_taskexecutor` | MongoDB | Warns if the count of MongoDB `TaskExecutorPoolSize` exceeds the number of available CPU cores. |
| [MongoDB - xfs](checks/mongodb-xfs.md) | `mongodb_xfs_ftype` | MongoDB | Warns if `dbpath` is not using the XFS filesystem type. |
| [Check if binaries are 32 bits](checks/platform-mysql-binaries-32-bit.md) | `mysql_32binary_on_64system` | MySQL | Notifies if `version_compile_machine` equals `i686`. |
| [InnoDB Tablespace size has a maximum limit](checks/configuration-check-innodb-data-file-path.md) | `mysql_configuration_innodb_file_maxlimit` | MySQL | Checks whether InnoDB is configured with the recommended auto-extend settings. |
| [innodb_file_per_table not enabled](checks/mysql_configuration_innodb_file_per_table_not_enabled.md) | `mysql_configuration_innodb_file_per_table_not_enabled` | MySQL | Warns when `innodb_file_per_table` is not enabled. |
| [Check for WAL file accumulation](checks/postgresql-wal-retention-check.md) | `postgresql_wal_retention_check` | PostgreSQL | Checks if there are too many WAL files retained in the WAL directory. |

## Schema & indexes

Use these checks to find unused, inconsistent or oversized indexes, tables without a primary key and excessive sequential scans:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB Inconsistent Indexes Across Shards](checks/mongodb-inconsistent-indexes-across-shards.md) | `mongodb_shard_collection_inconsistent_indexes` | MongoDB | Warns if there are inconsistent indexes across shards for sharded collections. Missing or inconsistent indexes across shards can have a negative impact on performance. |
| [MongoDB - Unused Indexes](checks/mongodb-unused-indexes.md) | `mongodb_unused_index` | MongoDB | Warns if there are unused indexes on any database collection in your instance. This requires enabling the `indexStats` collector. |
| [Are there tables with index sizes larger than data?](checks/datadesign-mysql-indexes-larger-than-data.md) | `mysql_indexes_larger` | MySQL | Checks for tables whose indexes are larger than their data, which suggests a schema to review. |
| [MySQL check for table without Primary Key](checks/tables-found-without-primary-keys.md) | `mysql_tables_without_pk` | MySQL | Checks tables without primary keys. |
| [Check for relations with a high number of indexes](checks/postgresql-high-number-of-indexes-check.md) | `postgresql_number_of_index_check` | PostgreSQL | Lists relations with more than ten indexes. |
| [PostgreSQL sequential scan check](checks/postgresql-sequential-scan-check.md) | `postgresql_sequential_scan_check` | PostgreSQL | Checks for tables with excessive sequential scans. |
| [Check for relations that have unused indexes](checks/postgresql-unused-index-check.md) | `postgresql_unused_index_check` | PostgreSQL | Lists relations with indexes that have not been used since the statistics were last reset. |

## Security

Use these checks to find weak access and password settings, risky users and privileges, and versions with known vulnerabilities:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB authentication](checks/mongodb-authentication.md) | `mongodb_auth` | MongoDB | Warns if MongoDB authentication is disabled. |
| [MongoDB Security AuthMech Check](checks/mongodb-scram-sha-256.md) | `mongodb_authmech_scramsha256` | MongoDB | Warns if MongoDB is not using the default SHA-256 hashing function as its SCRAM authentication method. |
| [MonogDB IP bindings](checks/mongodb-bindip.md) | `mongodb_bindip` | MongoDB | Warns if the MongoDB network binding is not set as Recommended. |
| [MongoDB Zlib Compression Heap Memory Vulnerability (CVE-2025-14847)](checks/mongodb-cve-2025-14847.md) | `mongodb_cve_2025_14847_zlib` | MongoDB | Checks whether MongoDB Server is vulnerable to CVE-2025-14847 with active Zlib compression. |
| [MongoDB CVE Version](checks/mongodb-cve-version.md) | `mongodb_cve_version` | MongoDB | Shows an error if the MongoDB or Percona Server for MongoDB version is older than the latest version with CVE fixes. |
| [MongoDB localhost authentication bypass enabled](checks/mongodb-localhost-bypass.md) | `mongodb_localhost_auth_bypass` | MongoDB | Warns if MongoDB localhost bypass is enabled. |
| [MySQL Automatic User Expired Password](checks/mysql-automatic-expired-password.md) | `mysql_automatic_expired_password` | MySQL | Warns if the MySQL parameter for automatic password expiry is not active. |
| [Checks if automatic_sp_privileges configuration is ON](checks/mysql_automatic_sp_privileges_enabled.md) | `mysql_automatic_sp_privileges_enabled` | MySQL | Checks if the `automatic_sp_privileges` configuration is `ON`. |
| [Load data in file active](checks/local-option-of-load-data-enabled.md) | `mysql_config_local_infile` | MySQL | Checks if the `LOAD DATA INFILE` functionality is active. |
| [secure_file_priv is empty](checks/mysql_configuration_secure_file_priv_empty.md) | `mysql_configuration_secure_file_priv_empty` | MySQL | Warns when `secure_file_priv` is empty as this enables users with the `FILE` privilege to create files at any location where MySQL server has Write permission. |
| [Check MySQL user password expiry](checks/mysql_password_expiry.md) | `mysql_password_expiry` | MySQL | Checks if MySQL user passwords are expired or expiring within the next 30 days. |
| [MySQL Users With Granted Public Networks Access](checks/mysql-private-networks-only.md) | `mysql_private_networks_only` | MySQL | Notifies about MySQL accounts that are allowed to connect from public networks. |
| [MySQL security check for replication user](checks/replica-without-replication-slave-account.md) | `mysql_replication_grants` | MySQL | Checks if replication is configured on a node without user grants. |
| [Secure transport is not required](checks/mysql-secure-transport-only.md) | `mysql_require_secure_transport` | MySQL | Checks whether `require_secure_transport` is enabled. |
| [Anonymous user (you must remove any anonymous user)](checks/security-anonymous-user.md) | `mysql_security_anonymous_user` | MySQL | Verifies if anonymous users are present, as this would contradict security best practices. |
| [User(s) has/have host definition '%' which is too open](checks/security-user-with-open-to-the-word-scope.md) | `mysql_security_open_to_world_host` | MySQL | Checks whether host definitions are set as `%` since this is overly permissive and could pose security risks. |
| [Password lifetime is not enforced](checks/default-password-lifetime.md) | `mysql_security_password_lifetime` | MySQL | Warns about password lifetime. |
| [MySQL security check for password](checks/validate-password-policy.md) | `mysql_security_password_policy` | MySQL | Checks for password policy. |
| [Replication privileges](checks/mysql-replication-privileges.md) | `mysql_security_replication_grants_mixed` | MySQL | Checks if replication privileges are mixed with more elevated privileges. |
| [Root user can connect from non local location](checks/root-account-can-login-remotely.md) | `mysql_security_root_not_local` | MySQL | Checks whether the root user has a host definition that is not set to `127.0.0.1` or `localhost`. |
| [User(s) not using secure SSL protocol to connect](checks/security-user-not-using-ssl-protocol.md) | `mysql_security_user_ssl` | MySQL | Reports users who are not using a secure SSL protocol to connect. |
| [Users have super privileges with remote or lack access restrictions](checks/security-non-root-accounts-super-privileges.md) | `mysql_security_user_super_not_local` | MySQL | Reports users with super privileges who can connect from a remote or loosely restricted host, such as `192.168.%`. |
| [User(s) without password](checks/security-user-without-password.md) | `mysql_security_user_without_password` | MySQL | Reports users without passwords. |
| [MySQL test Database](checks/mysql-test-database.md) | `mysql_test_database` | MySQL | Notifies if there are database named `test` or `test_%`. |
| [Check for security vulnerabilities](checks/postgresql-cve-check.md) | `postgresql_cve_check` | PostgreSQL | Checks if the currently installed version has reported security vulnerabilities. |
| [Check for password expiration](checks/postgresql-expiring-passwd.md) | `postgresql_expiring_passwd_check` | PostgreSQL | Checks for passwords that are expiring and displays the time left before they expire. |
| [PostgreSQL Super Role](checks/postgresql-super-role.md) | `postgresql_super_role` | PostgreSQL | Notifies if there are users with Superuser role. |

## Versions

Use these checks to find database versions that have reached end of life, aren't supported or have a newer release, and outdated extensions:

| Summary | Check name | Technology | What it checks |
| :------ | :--------- | :--------- | :------------- |
| [MongoDB version EOL](checks/mongodb-eol.md) | `mongodb_EOL` | MongoDB | Raises an error or a warning if your MongoDB or Percona Server for MongoDB version has reached or is nearing its end of life. |
| [MongoDB - FCV mismatch](checks/mongodb-fcv-mismatch.md) | `mongodb_fcv_check` | MongoDB | Warns if there is a mismatch between the MongoDB version and the internal FCV (Feature Compatibility Version) parameter setting. |
| [MongoDB Unsupported version check](checks/mongodb-unsupported-version.md) | `mongodb_unsupported_version` | MongoDB | Raises an error if your current PSMDB or MongoDB version is not supported. |
| [MongoDB version check](checks/mongodb-version.md) | `mongodb_version` | MongoDB | Shows the MongoDB or Percona Server for MongoDB versions in your environment, and the newer minor and major versions available. |
| [Checks mysql version for support](checks/mysql_unsupported_version_check.md) | `mysql_unsupported_version_check` | MySQL | Warns against an unsupported Mysql version. |
| [MySQL Version](checks/mysql-version.md) | `mysql_version` | MySQL | Warns if MySQL, Percona Server for MySQL, or MariaDB version is not the latest available one. |
| [End Of Life server version (5.7)](checks/mysql_version_eol_57.md) | `mysql_version_eol_57` | MySQL | Checks if the server version is EOL. |
| [Check if PostgreSQL version is EOL](checks/postgresql-eol-check.md) | `postgresql_eol_check` | PostgreSQL | Checks if the currently installed PostgreSQL version has reached its EOL and is no longer supported. |
| [Check for outdated extensions](checks/postgresql-extension-check.md) | `postgresql_extension_check` | PostgreSQL | Lists outdated extensions with newer versions available. |
| [Check for unsupported PostgreSQL](checks/postgresql-unsupported-check.md) | `postgresql_unsupported_check` | PostgreSQL | Verifies if the currently installed version is supported by Percona. |
| [Check for newer version of PostgreSQL](checks/postgresql-version-check.md) | `postgresql_version_check` | PostgreSQL | Checks if the currently installed version is outdated for its release level. |

## Related topics

- [Run Advisor checks and review results](advisors.md)
- [Schedule Advisor checks](schedule-advisor-checks.md)
- [Develop Advisor checks](develop-advisor-checks.md)
