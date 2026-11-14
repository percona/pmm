# MongoDB journal enabled

## Description
This advisor warns if the journal is explicitly disabled (`storage.journal.enabled: false`) on MongoDB versions before 7.0. A `mongod` that does not set the option is not reported, and MongoDB 7.0 and later are never reported because they always journal.
Disabled journal is dangerous because you could have a serious issue for data durability in case of a failure.

For Production systems, enable journal to ensure that data files are valid/recoverable.

It is always recommended to enable the journal. 

MongoDB enables journaling by default. Before 7.0, you can turn it off only by setting `storage.journal.enabled: false` explicitly, which is the only case this advisor reports. MongoDB 7.0 and later always journal and ignore the option.

For more information, see the [Journaling section](https://docs.mongodb.com/manual/core/journaling/) in the MongoDB documentation.

## Rule
```
 storage_journal = parsed.get("storage", {}).get("journal", parsed.get("storage.journal", {}))
 journal_disabled = (str(storage_journal.get("enabled")).lower() == "false")
```

## Resolution

To enable journaling: 
{.power-number}

1. Enable journal. 
2. Edit **mongod.conf** and set the following parameter:
```
storage:
  journal:
	enabled: true
```
3. Roll-restart your `mongod` (data bearing) nodes.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>