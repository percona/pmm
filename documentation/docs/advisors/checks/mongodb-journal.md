# MongoDB journal enabled

## Description
This advisor warns if the journal is not enabled. 
Disabled journal is dangerous because you could have a serious issue for data durability in case of a failure.

For Production systems, enable journal to ensure that data files are valid/recoverable.

It is always recommended to enable the journal. 

In recent versions (starting with versions 4.0 +), MongoDB enables journaling by default and doesn't allow turning it off.

For more information, see the [Journaling section](https://docs.mongodb.com/manual/core/journaling/) in the MongoDB documentation.



## Rule
```
 storage_journal = parsed.get("storage.journal", {})
 journal_enabled = (storage_journal.get("enabled") == "true")
```


## Resolution

Follow the steps below to enable journaling: 

1. Enable journal. 
2. Edit **mongod.conf** and set the following parameter:
```
storage:
  journal:
	enabled: true
```

3. Roll-restart your `mongod` (data bearing) nodes.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
