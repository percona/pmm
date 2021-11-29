# MySQL enforced data integrity checking is disabled

## Description
In order for the server to check data integrity, the sql_mode should have TRADITIONAL, STRICT_ALL_TABLES and STRICT_TRANS_TABLES set in sql_mode. The advisors raise an alert if one or more of them are missing.



## Rule
`SELECT @@sql_mode;`
Make sure TRADITIONAL, STRICT_ALL_TABLES, and STRICT_TRANS_TABLES.


## Resolution
Set sql_mode in a way that it contains TRADITIONAL, STRICT_ALL_TABLES and STRICT_TRANS_TABLES.

