# MySQL relay log on the replica node, not automatically purged

## Description
Automatic relay log purging is off, relay logs can take up an unnecessary amount of disk space. 
 Disable or enable automatic purging of relay logs as soon as they are no longer needed. The default value is 1 (enabled). This is a global variable that can be changed dynamically with SET GLOBAL relay_log_purge = N. Disabling purging of relay logs when enabling the --relay-log-recovery option risks data consistency and is therefore not crash-safe.


## Rule
`SELECT @@global.relay_log_purge;`


## Resolution
Set relay_log_purge to 1. 


