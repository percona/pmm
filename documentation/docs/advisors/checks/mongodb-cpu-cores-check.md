# MongoDB CPU cores check

## Description
This advisor warns if the number of CPU cores does not meet the minimum requirements according to best practices.

To avoid performance issues, see the following documentation to determine the number of CPU cores required for your environment: 

-  [MongoDB 4.0 Production notes](https://www.mongodb.com/docs/v4.0/administration/production-notes/#std-label-prod-notes-ram)
- [MongoDB 4.2 Production notes](https://www.mongodb.com/docs/v4.2/administration/production-notes/#std-label-prod-notes-ram)
- [MongoDB 4.4 Production notes](https://www.mongodb.com/docs/v4.4/administration/production-notes/#std-label-prod-notes-ram)
- [MongoDB 5.0 Production notes](https://www.mongodb.com/docs/v5.0/administration/production-notes/#std-label-prod-notes-ram)
- [MongoDB 6.0 Production notes](https://www.mongodb.com/docs/v6.0/administration/production-notes/#std-label-prod-notes-ram)

## Rule 
```MONGODB_GETDIAGNOSTICDATA
db.adminCommand({'getDiagnosticData':1}).data.systemMetrics.cpu.num_cpus


data = docs[0]["data"]
          print(repr(data))
          systemMetrics = data.get("systemMetrics", {})
          print(repr(systemMetrics))
          cpu = systemMetrics.get("cpu", {})
          numcpu = int(cpu.get("num_cpus", fail))

``` 

## Resolution
Increase the number of CPU cores so that it does not impact the performance.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
