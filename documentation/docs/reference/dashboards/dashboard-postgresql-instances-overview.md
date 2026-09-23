# PostgreSQL Instances Overview

This dashboard shows a high-level overview of all your PostgreSQL instances to help you quickly identify performance trends across your entire database infrastructure.

Start here when you need to compare performance across all your databases, identify slow queries affecting multiple services, or get a bird's-eye view of your PostgreSQL infrastructure health.

![!image](../../images/PMM_PostgreSQL_Instances_Overview.jpg)

## Databases Monitored
Shows a summary of your PostgreSQL monitoring coverage including the number of clusters, total nodes, and nodes without cluster configuration. The **Databases monitored** link takes you to the full inventory of monitored services for detailed management.

## Executed Queries  
Shows query execution trends for your top 5 busiest PostgreSQL services over time. Sudden spikes may indicate performance issues or unexpected workload changes that need investigation.

## Slow Queries
Shows the total count of queries that exceeded your configured slow query threshold across all monitored services over the selected time range. 

Focus on reducing this number across your infrastructure. High slow query counts indicate widespread performance problems that need systematic optimization.

## Transactions per Second
Shows the current transaction rate across all your PostgreSQL services combined. Use this as a high-level health indicator. Dramatic changes in TPS may indicate infrastructure issues, application problems, or significant workload shifts.

## Execution Time
Shows average query execution time trends for your PostgreSQL services over time. 

Watch for services with increasing execution times. Rising trends indicate performance degradation that needs investigation and optimization.

## Top slow queries

Shows the slowest-performing queries across all your PostgreSQL services, ranked by average execution time. Each row represents a single query aggregated over the selected time range rather than one row per collection interval, so the table stays short enough to scan.

Use this to identify queries that need optimization. Focus on queries with the highest execution times, then use **Calls** to prioritize by actual workload impact, since a query with a high average but a single call is usually less urgent than one running thousands of times.

For each query, you can see:

- **Slowest at**: when the query reached its peak execution time within the time range. Use this to correlate a spike with a deployment or a load event.
- **Service**: the PostgreSQL service the query ran on.
- **Username**: the database user that executed the query.
- **Query**: the query fingerprint. Click to expand the full query text.
- **Calls**: how many times the query ran during the time range.
- **Execution Time**: average execution time per call.
