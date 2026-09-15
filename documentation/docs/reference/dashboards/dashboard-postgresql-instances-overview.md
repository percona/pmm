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
Shows the slowest queries across all your PostgreSQL services, ranked by average execution time. Each row is a single query aggregated over the whole selected time range, rather than one row per time interval, so the table stays short enough to scan.

Alongside the query text, the table shows:

- **Slowest at** — when that query was at its slowest within the time range, which helps you correlate a spike with a deployment or a load event.
- **Calls** — how many times the query ran. This separates a one-off outlier from sustained load: a query with a high average and a single call is usually less urgent than a slightly faster one that runs thousands of times.
- **Execution Time** — the average time per call, weighted by the number of calls.

Focus optimization efforts on queries with the highest execution times, then use **Calls** to judge which of them actually affect your workload. This cross-service view helps you identify the most impactful slow queries across your entire infrastructure.
