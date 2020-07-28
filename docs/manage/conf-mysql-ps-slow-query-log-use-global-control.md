# `slow_query_log_use_global_control`

By default, slow query log settings apply only to new sessions.  If you want to
configure the slow query log during runtime and apply these settings to existing
connections, set the `slow_query_log_use_global_control` variable to `all`.

!!! seealso "See also"

    [MySQL Server 5.7 Documentation Setting variables](https://dev.mysql.com/doc/refman/5.7/en/set-variable.html)
