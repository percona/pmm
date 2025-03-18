# QAN Filters panel

The Filters panel on the left hand side of the [QAN dashboard](../../qan/index.md) helps you narrow down query data to focus on specific metrics, database instances, or performance issues.


![!image](../../../images/PMM_Query_Analytics_Panels_Filters.jpg)

## Understanding filters

- The **Filter** panel lists the filters grouped by category. It also shows the percentage of the main metrics (explained below). If you select a different metric, the percentages on the left panel will change as per this metric. When you select a metric, it reduces the overview list as per the matching filter.
- The first five of each category are shown. If there are more, the list is expanded by clicking **Show all** beside the category name, and collapsed again with **Show top 5**.
- Applying a filter may make other filters inapplicable. These become grayed out and inactive.
- Click the chart symbol <i class="uil uil-graph-bar"></i> to navigate directly to an item's associated dashboard.
- Separately, the global **Time range** setting filters results by time, either your choice of **Absolute time range**, or one of the predefined **Relative time ranges**.

![!image](../../../images/PMM_Query_Analytics_Time_Range.jpg)

## Available filter groups
The available filter groups depend on the database type you're monitoring.

### Common filter groups
These filter groups are available for all database types:

- **Environment**
- **Cluster**
- **Replication Set**
- **Database**
- **Schema**
- **Node Name**
- **Service Name**
- **Client Host**
- **User Name**
- **Service Type**
- **Node Type**

### MySQL-specific filter groups
- **Command Class**: filters by SQL command class (SELECT, INSERT, UPDATE, etc.)
- **Fingerprint**: Filters by normalized query pattern

### MongoDB-specific filter groups
- **Plan Summary**: filters queries by execution plan type (COLLSCAN, IXSCAN, etc.) to easily identify inefficient full collection scans
- **Client Application Name**: filters queries by the application name that generated them

### PostgreSQL-specific filter groups
- **Application**
- **Command Type**
- **Tables**
- **Client Application Name**


## Custom filter groups

!!! caution alert alert-warning "Important/Caution"
    This feature is still in [Technical Preview](../../../reference/glossary.md#technical-preview) and is subject to change. We recommend that early adopters use this feature for testing purposes only.

Filter queries using custom key=value pairs from query comments. This feature is disabled by default.

### Supported technologies and agents

- MySQL (`perfschema`, `slowlog`),
- PostgreSQL (`pg_stat_statements`, `pg_stat_monitor`)

**Example**

![!image](../../../images/PMM_QAN_Custom_Filter.png)

In the image above we have tagged queries running databases on Windows using the following comment: 

```sh
comment: /* OperationSystem='windows' */. 
```
Queries from the database running on Linux are tagged with:

```sh
/* OperationSystem='linux' */. 
```

All types of comments and multicomments are supported `(/* */, --, # etc)`. 

So the queries are as follows:

```sh
SELECT * /* OperationSystem='windows' */ FROM city;
SELECT city /* OperationSystem='linux' */ FROM world;
```

In the output, you can see another custom group in the `OperationSystem` filter. Use this to easily filter by any custom key or value.

### Enabling custom filter groups

- **via CLI**: While adding a service through CLI use the flag `comments-parsing`. Possible values are `on/off`. 

    Example for adding MySQL with comments parsing on:

    ```sh
    pmm-admin add mysql --username=root --password=root-password --comments-parsing="on"
    ```

- **via UI**: While adding a service through the UI you will see new checkbox to `enable/disable` comments parsing for current service.

    ![!image](../../../images/PMM_QAN_Parsing.png)

!!! note alert alert-primary "MySQL CLI"
    - If you are using official MySQL CLI to trigger queries, start mysql with `--comments` flag. Otherwise comments will not be parsed.
    - In case of PGSM (`pg_stat_monitor`), set the DB variable `pgsm_extract_comments=yes`

