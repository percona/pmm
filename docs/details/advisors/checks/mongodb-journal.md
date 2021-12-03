# MongoDB journal enabled

## Description
This check returns a warning if the journal is not enabled. 
This is dangerous because you could have a serious issue for data durability in case of a failure.
For Production systems - Enable journal to ensure that data files are valid/recoverable.

It is always recommended to enable the journal. More recent versions of MongoDB don’t permit the journal to be turned off.  MongoDB enables journaling by default in recent versions (4.0 +).
[https://docs.mongodb.com/manual/core/journaling/](https://docs.mongodb.com/manual/core/journaling/)



## Rule
```
 storage_journal = parsed.get("storage.journal", {})
 journal_enabled = (storage_journal.get("enabled") == "true")
```



## Resolution

Please Perform the steps mentioned below to enable journaling

Enable journal. 
Edit mongod.conf and set the below parameter.
```
storage:
  journal:
	enabled: true
```
Perform a rolling restart of your mongod (data bearing) nodes
