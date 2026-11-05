# Alert rules and alert templates

Alert rules describe the circumstances under which you want to be alerted. The evaluation criteria that you define determine whether an alert will fire. 

An alert rule consists of one or more queries and expressions, a condition, the frequency of evaluation, and the duration over which the condition is met. For example, you might configure an alert to fire and trigger a notification when MongoDB is down.

During evaluation, an alert rule moves through the following states:

- **Normal**: Everything is working correctly, and the conditions specified in the rule have not been met. This is the default state for newly created rules.
- **Pending**: The conditions specified in the alert rule have been met, but for less time than the configured duration.
- **Firing**: Both the alert condition and the configured duration have been met.
- **Recovering**: The alert conditions are no longer met, but the alert remains active until the metric is stable below the threshold. This prevents alerts from repeatedly flipping between **Firing** and **Normal** when metrics briefly cross a threshold.

It takes at least one evaluation cycle for an alert rule to transition from one state to another (e.g., from `Normal` to `Pending`).

## How alert templates work

PMM provides a set of alert rule templates with common events and expressions for alerting. You can use them as a basis for alert rules, or create your own templates if you need custom expressions.

To see the templates available in your PMM instance, go to **Alerts > Templates**. PMM lists two types of templates there:

- Built-in templates, available out of the box. The [list of available alert templates](templates_list.md) describes each one.
- Custom templates that you create or upload on this page. You can also store your custom template files in the `/srv/alerting/templates` directory, and PMM loads them at startup.

### What alert templates monitor

Alert templates are YAML files that provide the source framework for alert rules. Each template contains general template details and the queries that PMM runs to decide whether to alert. A template alerts on one of two kinds of data:

- **Metrics**: the health data that PMM collects from your nodes and services, such as CPU load or connections. Most templates [alert on server and database health](#alert-on-server-and-database-health), using [MetricsQL](https://docs.victoriametrics.com/MetricsQL.html) queries. MetricsQL is backward compatible with PromQL.
- **Query Analytics data**: how individual queries perform, as you see it in Query Analytics (QAN). Starting with PMM 3.11.0, templates can [alert on query performance](#alert-on-query-performance), using SQL queries.

## Prepare to create alert rules

Before creating PMM alert rules, configure the required alert resources:
{.power-number}

1. Go to **Configuration > Settings > Advanced settings** and ensure that the **Percona Alerting** option is enabled. When this is disabled, the **Alerting** page displays only Grafana-managed alert rules. This means that you will not be able to create alerts based on PMM templates.
2. Go to **All dashboards > Browse all dashboards** and check the folders available for storing alert rules. If none of the available folders are relevant for your future alert rules, click **New > New folder** and create a custom one.
3. Go to **Alerts > Templates** and check the default PMM templates. If none of the templates include a relevant expression for the type of alerts that you want to create, click **Add template** to create a custom template instead.

## Create an alert rule from a template

After provisioning the resources required for creating Percona templated alerts, you are now ready to create your alert rule based on a Percona template. 

If you want to learn about creating Grafana alerts instead, check out [Grafana's documentation](https://grafana.com/docs/grafana/latest/alerting/):
{.power-number}

1. Go to **Alerts > Alert rules**, and click **New alert rule from template**.
2. On the **Create alert rule from template** page, choose the template on which you want to base the new alert rule. This automatically populates the **Name**, **Duration**, and **Severity** fields with information from the template. You can change these values if you want to override the default specifications in the template.
3. From the **Folder** drop-down menu, select the location where you want to store the rule.
4. In the **Filters** section, specify if you want the alert rule to apply only to specific services or nodes. For example: `service_name=ps5.7`. When creating alert rule filters, consider the following:
   
    - Filters use conjunction semantics. This means that if you add more than one filter, PMM will combine their conditions to search for matches: filter 1 AND filter 2 AND filter 3.
    - **Label** must be an exact match. You can find a complete list of labels using the :material-compass-outline: **Explore** menu in PMM, or see [Labels reference](../reference/labels-reference.md).
    - For query performance templates, filters apply to the [columns that the query returns](#filter-and-route-query-performance-alerts).

5. Click **Save and Exit** to close the page and go to the **Alert Rules** tab where you can review, edit and silence your new alert.

If the template has a **Dynamic** badge, you can later [change the threshold of the new alert rule for individual nodes](alert-thresholds.md).

## Alert on server and database health

Most PMM alerts are based on metrics, the health data that PMM collects from your nodes and services, such as CPU load, memory, or connections.

### Use a built-in metrics template

PMM includes built-in metrics templates for operating systems, databases, and PMM itself, such as **Node high CPU load** or **MySQL restarted**. To use one, [create an alert rule](#create-an-alert-rule-from-a-template) from it. The [list of available alert templates](templates_list.md) describes each one.

### Write a metrics template

If no built-in template fits, write your own template with a MetricsQL query, in `expr` or in `queries`, using the [custom template fields](#define-a-custom-template). Before you add the query to the template, test it in the **Explore** section of PMM, where you can also query any PMM internal database.

To test a MetricsQL query:
{.power-number}

1. On the main menu in PMM, choose **Explore > PromQL builder**.
2. Enter your expression in the **Metrics** field and click **Run query**.

For example, to check the CPU usage, Go to **Explore > PromQL builder** and run the query expression below:
```
(1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
```
![!](../images/alerting-sample-query.png)

Note that to paste the query above, **Explore** must be in `Code` mode, and not in `Builder` mode.

When the template is ready, add it on the **Alerts > Templates** page by clicking **Add template**, then [create an alert rule from it](#create-an-alert-rule-from-a-template).

## Alert on query performance

Starting with PMM 3.11.0, you can get alerts when queries on your services slow down. Unlike metrics alerts, these alerts are based on the query data that you see in Query Analytics.

### Use the built-in query performance template

To alert on slow queries, [create an alert rule](#create-an-alert-rule-from-a-template) from the built-in [**QAN slow queries**](templates_list.md#qan_alerts) template and set the threshold that fits your services.

### What to expect from query performance alerts

Query Analytics data reaches PMM later than metrics, so these alerts react more slowly than metric alerts:

- Each evaluation reads the last 5 minutes of Query Analytics data. The data arrives in one-minute buckets, about one to two minutes late, so these alerts fire and resolve a few minutes later than metric alerts.
- A single slow query keeps the alert active until the slow query leaves the 5-minute window.
- If ClickHouse is unavailable, rules based on Query Analytics data report a data source error in addition to the **PMM ClickHouse is down** alert.

### Filter and route query performance alerts

When you create a rule from a query performance template, filters apply to the columns that the query returns:

- A label that the query doesn't return counts as empty, as with metrics. A match filter on that label keeps every row. A mismatch filter on that label keeps every row, unless its regular expression matches an empty value.
- Filters apply after any `LIMIT` in the query.
- Custom labels that you add to nodes and services aren't available to these queries, so a filter on a custom label doesn't narrow Query Analytics alerts.

Each rule queries ClickHouse separately. To alert on many services, create one rule and route the alerts by the `service_name` label in your notification policies.

### Write a query performance template

If the built-in template doesn't fit, write your own template with an SQL query on Query Analytics data, using the [custom template fields](#define-a-custom-template).

#### Write the query

Query Analytics stores its data in the `pmm.metrics` table of the ClickHouse database. Write an SQL query on this table that:

- returns a `DateTime` time column and exactly one numeric value column
- returns the label columns that you want to filter on, route by, or show on the alert, such as `service_name`, `node_name`, `region`, and `environment`
- works in the **Time Series** query format
- uses only `float` parameters
- doesn't contain `[[`, for example in nested array literals, because `[[` starts a template parameter

Every string column that the query returns becomes an alert label, and each combination of label values becomes a separate alert. To avoid an alert for every single query, don't return high-cardinality columns such as `queryid` or `fingerprint`.

To cover every database type, don't rely on `m_query_time_max` alone, because only MySQL slow log and MongoDB report it. For PostgreSQL and MySQL Performance Schema, use the average query time, as the built-in `pmm_qan_slow_queries` template does.

#### Test the query

Before you add the query to a template, check its results in **Explore**:
{.power-number}

1. In **Explore**, select the **ClickHouse** data source.
2. Select **SQL Editor**, and set **Query Type** to **Time Series**.
3. Set the time range to **Last 5 minutes** to match what the alert rule reads.
4. Enter the query and click **Run query**.

#### Build the template

Alerts need one value for each series, so a template on Query Analytics data has three steps: the query, a `reduce` expression that turns its results into one value, and a `math` expression that compares that value with the threshold. In the template:

- set `datasource: clickhouse` on the query
- add a `reduce` expression over the query, for example `reducer: max` with `expression: $A`
- compare the reduce expression, not the query, with the threshold in a `math` expression, for example `$B > [[ .threshold ]]`, and set that expression as the `condition`
- read values in annotations and labels from the reduce expression, for example `$values.B.Value`

??? info "Example: alert on slow queries"

    ```yaml
    {% raw %}
    ---
    templates:
      - name: custom_qan_slow_queries
        version: 1
        summary: Slow queries
        queries:
          - ref_id: A
            datasource: clickhouse
            expr: |-
              SELECT $__toTime AS time, service_name, node_name, max(m_query_time_max) AS query_time
              FROM pmm.metrics
              WHERE period_start >= $__fromTime AND period_start <= $__toTime
              GROUP BY service_name, node_name
        expressions:
          - ref_id: B
            type: reduce
            reducer: max
            expression: $A
          - ref_id: C
            type: math
            expression: "$B > [[ .threshold ]]"
        condition: C
        params:
          - name: threshold
            summary: Query execution time
            unit: s
            type: float
            value: 10
        for: 1m
        severity: warning
        annotations:
          summary: Slow queries on {{ $labels.service_name }}
          description: |-
            The slowest query on {{ $labels.service_name }} took {{ printf "%.2f" $values.B.Value }}s. The alert threshold is [[ .threshold ]]s.
    {% endraw %}
    ```

When the template is ready, add it on the **Alerts > Templates** page by clicking **Add template**, then [create an alert rule from it](#create-an-alert-rule-from-a-template).

## Define a custom template

Custom templates are YAML files that use the fields below, for both metrics and query performance alerts. You can base multiple alert rules on the same template. For example, one `pmm_node_high_cpu_load` template can be the source for separate rules for production and staging, or for warning and critical severities.

- **name** (required): uniquely identifies template. Spaces and special characters are not allowed.
- **version** (required): defines the template format version.
- **summary** (required): a template description.
- **expr** (required unless you use **queries** and **expressions**): a MetricsQL query string with parameter placeholders.
- **queries** (optional): a list of queries. Use them with **expressions** and **condition** instead of **expr**. Each query has these fields:
    - **ref_id** (required): the step ID. It must start with a letter or underscore and contain only letters, digits, and underscores.
    - **expr** (required): a MetricsQL query, or SQL for a ClickHouse query.
    - **datasource** (optional): `metrics` for a MetricsQL query, which is the default, or `clickhouse` for an SQL query on Query Analytics data.
- **expressions** (optional): a list of expressions over the queries, each with a `ref_id`, a `type`, and an `expression` that references other steps by `$ref_id`. Each `ref_id` follows the same rules as for queries. The supported types are:
    - **type: math**: a calculation or comparison, for example `$A > [[ .threshold ]]`.
    - **type: reduce**: reduces one query or expression to a single value for each series. Set `expression` to that step, for example `$A`, and `reducer` to `sum`, `mean`, `min`, `max`, `count`, `last`, or `median`.
- **condition** (required with **queries**): the `ref_id` of the query or expression that fires the alert. The condition can't be a ClickHouse query.
- **params**: contains parameter definitions required for the query. Each parameter has a name, type, and summary. It also may have a unit, available range, and default value.
    - **name** (required): the name of the parameter. Spaces and special characters are not allowed.
    - **summary** (required): a short description of what this parameter represents.
    - **unit** (optional): PMM currently supports either s (seconds) or % (percentage).
    - **type** (required): PMM currently supports the `float` type. `string`, `bool`, and other types will be available in a future release.
    - **range** (optional): defines the boundaries for the value of a  float parameter
    - **value** (optional): default parameter value. Value strings must not include any of these special characters: `< > ! @ # $ % ^ & * ( ) _ / \ ' + - = (space)`
    - **overridable** (optional): set to `true` to enable per-node overrides of this threshold without editing the alert rule. The template must use **queries** and **expressions**, and the parameter must be the whole right-hand side of a comparison, for example `$A > [[ .threshold ]]`. For details, see [per-node threshold overrides](alert-thresholds.md).
    - **override_scopes** (optional): the scopes at which you can override this parameter. Defaults to `[node]`. Only `node` is currently supported. Requires `overridable: true`.
- **for** (required): specifies the duration of time that the expression must be met before the alert will be fired
- **severity** (required): specifies default alert severity level

    The **Alerts > Status** page groups severities into four bands: `emergency`, `alert` and `critical` show as **Critical**; `notice`, `info`, `informational` and `debug` show as **Info**. `error` and `warning` keep their names. The alert label keeps the level you set. An alert with no severity, or with a value outside these levels, shows that value without an icon and is listed under **Unknown** in the **Severity** filter.

- **labels** (optional): are additional labels to be added to generated alerts
- **annotations** (optional): are additional annotations to be added to generated alerts.

??? info "Template example"

    ```yaml
    {% raw %}
    ---
    templates:
      - name: pmm_node_high_cpu_load
        version: 1
        summary: Node high CPU load
        queries:
          - ref_id: A
            expr: |-
              (1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
        expressions:
          - ref_id: C
            type: math
            expression: "$A > [[ .threshold ]]"
        condition: C
        params:
          - name: threshold
            summary: A percentage from configured maximum
            unit: "%"
            type: float
            range: [0, 100]
            value: 80
            overridable: true
        for: 5m
        severity: warning
        annotations:
          summary: Node high CPU load ({{ $labels.node_name }})
          description: |-
            {{ $labels.node_name }} CPU load is more than [[ .threshold ]]%.
    {% endraw %}
    ```

## Create recording rules

Record an alert rules to pre-compute a query on a schedule and store the result as a new metric, which you can then reference in alert rules. This is useful when the same query powers multiple alert rules, for example, average CPU load across all nodes or replication lag across a cluster.

To create a recording rule, click **Alerts > Alert rules** + New recording rule**. For full instructions, see [Create recording rules](https://grafana.com/docs/grafana/latest/alerting/alerting-rules/create-recording-rules/) in the Grafana documentation.
