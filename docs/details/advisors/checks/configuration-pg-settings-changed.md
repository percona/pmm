# Settings changed on a instance that requires a restart

## Description

There is one or more parameter setting that requires a server restart/reload that was changed without a following reload/restart.

## Resolution

The parameters can be checked with the below query:
```
SELECT name, setting, short_desc, reset_val FROM pg_settings WHERE pending_restart IS true;
```

The PostgreSQL server needs to be restarted for the new value to be applied. 
