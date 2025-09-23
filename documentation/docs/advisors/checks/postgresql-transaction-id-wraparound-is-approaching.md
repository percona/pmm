 # Transaction ID wraparound is approaching
 
## Description
This advisor check verifies the age of the database's transaction IDs, and notifies if any is approaching the wraparound limit by 20% or more.
To understand the PostgreSQL wraparound concept, here's an excerpt from Robert Bernier blog post:

*PostgreSQL at any one time has a number of transactions that are tracked by a unique ID.* 

*Every so often that number reaches the upper limit that can be registered, for example, 200 million transactions which is the default and is then renumbered.*

*But if the number of unique transaction IDs goes to its maximum transactions limit, known as TXID Wraparound, Postgres will force a shutdown in order to protect the data.*

*Here’s how it works:*

*- 4 billion transactions, 2^32, is the integer upper limit for the datatype used in Postgres.*
*- 2 billion transactions, 2^31, is the upper limit that PostgreSQL permits before forcing a shutdown.*
*- **10 million** transactions before the upper limit is reached, WARNING messages consisting of a countdown will be logged.*
*- <b style="color:#e02f44;">1 million</b>  transactions before the upper limit is reached, PostgreSQL goes to READ-ONLY mode.*

There are different causes for a database approaching the wraparound scenario:

*Transaction ID Wraparound can be caused by a combination of one or more of the following circumstances:*

*- Autovacuum is turned off*
*-* Long-lived transactions*
*- Database logical dumps (on a REPLICA using streaming replication)*
*- Many session connections with locks extending across large swaths of the data cluster*
*- Intense DML operations forcing the cancellation of autovacuum worker processes*
*- A well-configured AUTOVACUUM process should perform the corresponding FREEZE operations over the tables so that the transactions ID can be reused and the wraparound stays far away. But as listed above, there are some situations where the AUTOVACUUM might not complete the operation successfully.*


## Resolution
We have three main stages when dealing with the wraparound issue:

- Actions to prevent the wraparound.
- Actions when PostgreSQL approaches the wraparound, less than 10M transactions are left.
- Actions when PostgreSQL has shut down as the effect of reaching the wraparound limit. 

### Prevent wraparound

As stated before, the AUTOVACUUM should be sufficient to keep the wraparound issue away. 

But if for some reason, the system cannot catch up with the workload, the recommendation is to schedule some manual VACUUM job:

```vacuumdb -F -z -j 10 -v --host=<BD_HOST> --username=<DB_USER> --dbname=<DB_TO_VACUUM>```


Here:
**-F|--freeze** - flag to freeze row transaction information.
**-z|--analyze** - flag to update optimizer statistics.
**-j|--jobs=N** - flag to set a number of concurrent connections to vacuum. It can vary from a couple to a value equal to the number of CPUs on the host.


### Approaching wraparound (less than 10M transactions left) 

When the database is near the wraparound by 10M transactions or less, the DB log will show WARNING messages, and we are in a timed race to avoid the instance shutdown. 

An effective way to quickly start moving backward the wraparound is to identify the specific tables with the oldest transaction ID age.
Then, vacuum (FREEZE) those specific tables. 

In his blogpost, Rober Bernier recommends using the following pair of scripts:

1. Identify the database with the oldest TXID.
2. Generate a list of tables in order of the oldest TXID age to the youngest.
3. Feed this list of tables into a script that invokes vacuumdb and VACUUM one table per invocation.

Script one generates a list of tables in a selected database and calls script two, which executes the VACUUM on each of those tables individually.

**SCRIPT ONE**  (go1_highspeed_vacuum.sh)
```yaml #!/bin/bash
#
# INVOCATION
# EX: ./go1_highspeed_vacuum.sh
#

########################################################
# EDIT AS REQUIRED
export CPU=4
export PAGER=less PGUSER=postgres PGPASSWORD=mypassword PGDATABASE=db01 PGOPTIONS='-c statement_timeout=0'
########################################################

SQL1="
with a as (
  select  c.oid::regclass as table_name,
          greatest(age(c.relfrozenxid),age(t.relfrozenxid))
  from pg_class c
  left join pg_class t on c.reltoastrelid = t.oid
  where c.relkind in ('r', 'm')
  order by 2 desc
)
select table_name from a
"

LIST="$(echo "$SQL1" | psql -t)"

# the 'P' sets the number of CPU to use simultaneously
xargs -t -n 1 -P $CPU ./go2_highspeed_vacuum.sh $PGDATABASE<<<$LIST

echo "$(date): DONE"
```

**SCRIPT TWO** (go2_highspeed_vacuum.sh)

``` yaml
#!/bin/bash

########################################################
# EDIT AS REQUIRED
export PAGER=less PGUSER=postgres PGPASSWORD=mypassword PGOPTIONS='-c statement_timeout=0'
export DB=$1

########################################################

vacuumdb --verbose ${DB} > ${DB}.log 2>&1
``` 


### PostgreSQL has shut down due to wraparound

If the service is already down, as the effect of a data protective measure, the only option is to perform the vacuum in single-user mode:

``` yaml
# variable PGDATA points to the data cluster
postgres --single -D $PGDATA postgres <<< 'vacuum analyze'
```

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }