# Advisor check: PostgreSQL excessive sequential scans 

## Description
Checks for relations with excessive sequential scans vs. index scans. Although the query planner will occasionally choose sequential scans when it is more efficient than using an index (typically low count for tuples), this check is based on at least 50,000 live tuples in the relation.

## Resolution
To fix this issue, follow the steps below:

- Make sure that the relations are analyzed regularly.  
- Check **pg_stat_user_tables** for statistics about vacuums and analyze. 
- Identify the queries using the relations noted in the check, and run EXPLAIN on them. This will help identify relations with missing indexes. 
- Rewrite a bad query to use indexed columns when possible.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

