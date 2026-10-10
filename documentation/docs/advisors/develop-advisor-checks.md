# Developing Advisor checks

PMM ships with built-in Advisor checks that detect common security threats, performance degradation, data loss and data corruption. You can also create custom checks in the PMM UI to cover use cases that are specific to your database infrastructure. PMM stores custom checks in its database and runs them together with the built-in checks.

## Check components

A check is a combination of:

- One or more queries that collect data from the monitored database, from metrics, or from Query Analytics
- A [Starlark](https://github.com/google/starlark-go) script that turns the collected data into findings

Starlark is a dialect of Python. The script runs in a sandbox on PMM Server and can't perform any I/O.

## How checks run

When a run reaches a check, PMM processes each service of the check's technology as follows:
{.power-number}

1. PMM runs the check's queries. The `pmm-agent` that monitors the service runs the database queries. PMM Server runs the metrics queries against its VictoriaMetrics database and the `CLICKHOUSE_SELECT` queries against its Query Analytics database.
2. PMM Server passes the query results to the check script, which runs in a separate sandboxed process.
3. PMM records the outcome as an insight. Each finding that the script returns becomes a **Failed** insight. If the script returns no findings, the check passed and gets an **OK** insight. If a query or the script fails, the insight gets the **Error** status and shows the reason.

The script must finish within 5 seconds, and the whole check, including its queries, within 5 minutes. For all insight statuses, see [Insight status](advisors.md#insight-status).

## Built-in and custom checks

The **Advisors > Catalog** page lists both kinds of checks. The **Source** column shows **Builtin** for the checks that ship with PMM and **Custom** for the checks that you create. The following table shows what you can do with each kind:

| Action | Built-in | Custom |
|--------|:--------:|:------:|
| Run the check right away | ✓ | ✓ |
| Change the interval in the **Interval** column | ✓ | ✓ |
| Enable or disable the check with the **Status** switch | ✓ | ✓ |
| Disable the check for specific services | ✓ | ✓ |
| View the details and the script, copy the script, test or clone the check | ✓ | ✓ |
| Edit the check | ✗ | ✓ |
| Delete the check | ✗ | ✓ |

To change what a built-in check does, clone it and edit the copy.

## Create a check

### Prerequisites

Before you start, make sure that:

- The **Advisors** option is enabled under **Configuration > Settings > Advisors**.
- PMM monitors at least one service of the technology that the check targets, so that you can test the check.

### Add the check

To create a custom check:
{.power-number}

1. Go to **Advisors > Catalog** and click the :material-plus: **Add advisor** icon. To start from an existing check instead, double-click the check's row and click **Clone** in the details pane. The name of the clone starts as `custom_` followed by the name of the source check.
2. Fill in the fields of the form. For details, see [Check fields](#check-fields).
3. To try the check before you save it, select a service in **Test on service** and click **Test**. **Test results** shows the findings, or the error if a query or the script failed. **Script output** shows what the script printed with `print()`. The test doesn't save the check and doesn't record any insight.
4. Click **Save**. The check appears in the catalog as enabled, with the **Custom** source.

### Run the check and review the results

To run your check and review its results:
{.power-number}

1. Custom checks run according to their interval, like built-in checks. To run a check right away, find it in **Advisors > Catalog** and click the :material-play-outline: **Run** icon in its row.
2. In the confirmation message, click **View results**. **Advisors > Insights** opens, filtered by the ID of the run, and shows the check's results for each service. From there:

    - To see only the insights of your check, across all runs, click the :material-dots-vertical: icon in one of its rows and select **Filter by name**.
    - To see how much of its plan the run covered, go to **Advisors > Run history**. For details, see [Run coverage](advisors.md#run-coverage).

## Edit or delete a check

You can edit and delete only custom checks:

- To edit a check, click the :material-pencil-outline: **Edit** icon in its row, or click **Edit** in its details pane. You can change every field except **Name**. Test your changes before you click **Save**.
- To delete a check, click the :material-trash-can-outline: **Delete** icon in its row and confirm. You can't undo the deletion.

## Check fields

The check form has the following fields, and all of them are required:

- **Name**: The unique ID of the check. It can contain letters, digits and underscores, must start with `custom_`, and can be up to 128 characters long. You can't change the name after you create the check.
- **Summary**: A short, human-readable title of the check.
- **Description**: A longer description of what the check verifies.
- **Category**: Groups the check with related checks, for example **Security** or **Performance**. Select an existing category or type a new one.
- **Technology**: **MySQL**, **PostgreSQL** or **MongoDB**. The check runs on services of this technology, and the technology determines which query types you can use.
- **Interval**: **Standard**, **Rare** or **Frequent**. This sets how often the check runs automatically. For the length of each interval, see [Change run interval for automatic advisors](advisors.md#change-run-interval-for-automatic-advisors).
- **Queries**: One or more queries, each with a **Type**, a **Query** and optional parameters. For the available types, see [Query types](#query-types). For the parameters, see [Query parameters](#query-parameters).
- **Script**: The Starlark script that processes the query results. For details, see [Check script](#check-script).

## Check script

The script must define a `check_context(docs, context)` function:

- `docs` is a read-only list with one item for each query, in the order of the queries. Each item holds the result of that query: the returned rows for SQL databases and the returned documents for MongoDB.
- `context` is a dictionary with helper functions. For details, see [Script functions](#script-functions).

The function returns a list of findings, which can be empty. To report that the check couldn't evaluate the data, return a string or call `fail()`. The check then gets the **Error** status with your message.

Each finding is a dictionary with the following keys:

- `summary` (required): A short description of the issue.
- `severity` (required): One of the levels listed in [Check severity levels](#check-severity-levels), in lowercase.
- `description` (optional): Details of the issue.
- `read_more_url` (optional): A link to remediation guidance. It must be a valid URL.
- `labels` (optional): A dictionary of string keys and values that describe the finding.

Indent the script with spaces, not tabs.

### Script functions

The global functions `print` and `fail` are available. When you test a check, **Script output** shows what `print` wrote.

The `context` dictionary provides the following functions:

- `parse_version(version)`: Parses a version string and returns a dictionary with the `major`, `minor`, `patch`, `rest`, `num` and `numrest` keys.
- `format_version_num(num)`: Formats a version number, such as `80036`, as a version string, such as `8.0.36`.
- `ip_is_private(address)`: Returns `True` if the IP address is private, or if the network in CIDR notation, such as `10.0.0.0/8`, lies fully inside a private address block. Returns `False` if it isn't, and `None` if the argument isn't a valid address or network.

Get a function from `context` before you call it, for example `parse_version = context.get("parse_version", fail)`.

### Finding granularity

Return one finding per service — or one finding per database for checks that examine per-database objects — rather than one finding per offending object. Aggregate the objects into the finding: put the count in the summary (for example, `3 relation(s) with unused indexes in database app`), list the objects in the description, and set labels such as `database` and `count`. The Advisors Insights list distinguishes findings by their summary and labels, so per-object findings with a shared summary flood the history with rows that look identical. If your check reports issues at different severity levels, return one aggregated finding per severity level.

### Example

The following check reports the MySQL version, the uptime and the recent load average of each MySQL service. It uses these fields:

- **Technology**: **MySQL**
- **Queries**, in the following order:
    - A `MYSQL_SHOW` query with the text `VARIABLES`
    - A `METRICS_INSTANT` query with the text {% raw %}`mysql_global_status_uptime{service_name=~"{{.ServiceName}}"}`{% endraw %}
    - A `METRICS_RANGE` query with the text {% raw %}`avg by (node_name) (avg_over_time(node_load1{node_name=~"{{.NodeName}}"}[5m]))`{% endraw %}, and two parameters: `range` set to `15m` and `step` set to `5m`

The script reads the results of the queries from `docs[0]`, `docs[1]` and `docs[2]`. Each series of a `METRICS_RANGE` result holds its data points in `values`, as `[timestamp, value]` pairs:

```python
def check_context(docs, context):
    results = []

    for row in docs[0]:
        name, value = row["Variable_name"], row["Value"]
        if name == "version":
            results.append({
                "summary": "MySQL has version {}".format(value),
                "description": "Current version is {}".format(value),
                "read_more_url": "",
                "severity": "warning",
                "labels": {},
            })

    uptimeNow = int(int(docs[1][0]["value"][1])/60)
    results.append({
        "summary": "MySQL uptime {} min".format(uptimeNow),
        "description": "Current uptime is {} min".format(uptimeNow),
        "read_more_url": "",
        "severity": "warning",
        "labels": {},
    })

    dataPoints = []
    for row in docs[2][0]["values"]:
        dataPoints.append(row[1])

    results.append({
        "summary": "Node load average for the last 15 minutes {}".format(dataPoints),
        "description": "Data points {}".format(dataPoints),
        "read_more_url": "",
        "severity": "warning",
        "labels": {},
    })

    return results
```

## Check severity levels

You can label your advisor checks with one of the following available severity levels:

- Critical
- Error
- Warning
- Info

Findings with any other severity level (previously: Emergency, Alert, Notice, Debug) are rejected, and the check run fails with an error.

PMM groups failed checks by their severity and displays them under **Advisors > Insights**.

## Query types

Each check technology supports its own database query types. MySQL checks can use the `MYSQL_*` types, PostgreSQL checks the `POSTGRESQL_*` types, and MongoDB checks the `MONGODB_*` types. Checks of every technology can use `METRICS_INSTANT`, `METRICS_RANGE` and `CLICKHOUSE_SELECT`.

Expand the following table for the list of query types:

??? note alert alert-info "Query types"

    | Query type  |  Description | "query" required (must be empty if "No")   |
    |---|---|---|
    | MYSQL_SHOW |Executes 'SHOW …' clause against MySQL database. |Yes|
    | MYSQL_SELECT    |     Executes 'SELECT …' clause against MySQL database.  |Yes|
    | POSTGRESQL_SHOW     |    Executes 'SHOW ALL' command against PosgreSQL database.    |No|
    | POSTGRESQL_SELECT      | Executes 'SELECT …' clause against PosgreSQL database.  |Yes|
    | MONGODB_GETPARAMETER     | Executes db.adminCommand( { getParameter: "*" } ) against MongoDB's "admin" database. For more information, see [getParameter](https://docs.mongodb.com/manual/reference/command/getParameter/)| No|
    | MONGODB_BUILDINFO    | Executes db.adminCommand( { buildInfo:  1 } ) against MongoDB's "admin" database. For more information, see [buildInfo](https://docs.mongodb.com/manual/reference/command/buildInfo/) | No|
    | MONGODB_GETCMDLINEOPTS          |    Executes db.adminCommand( { getCmdLineOpts: 1 } ) against MongoDB's "admin" database. For more information, see [getCmdLineOpts](https://docs.mongodb.com/manual/reference/command/getCmdLineOpts/) |No|
    | MONGODB_REPLSETGETSTATUS     |   Executes db.adminCommand( { replSetGetStatus: 1 } ) against MongoDB's "admin" database. For more information, see  [replSetGetStatus](https://docs.mongodb.com/manual/reference/command/replSetGetStatus/) |No|
    | MONGODB_GETDIAGNOSTICDATA |Executes db.adminCommand( { getDiagnosticData: 1 } ) against MongoDB's "admin" database. For more information, see [MongoDB Performance](https://docs.mongodb.com/manual/administration/analyzing-mongodb-performance/#full-time-diagnostic-data-capture)| No|
    | METRICS_INSTANT |Executes instant [MetricsQL](https://docs.victoriametrics.com/MetricsQL.html) query. Query can use placeholders in query string {% raw %} **{{.NodeName**}} and **{{.ServiceName}}**  {% endraw %}. Both match target service/node names. To read more about instant queries, check out the [Prometheus docs](https://prometheus.io/docs/prometheus/latest/querying/api/#instant-queries).|Yes|
    | METRICS_RANGE |Executes range [MetricsQL](https://docs.victoriametrics.com/MetricsQL.html) query. Query can use placeholders in query string {% raw %} **{{.NodeName**}} and **{{.ServiceName}}**  {% endraw %}. Both match target service/node names. To read more about range queries, check out the [Prometheus docs](https://prometheus.io/docs/prometheus/latest/querying/api/#range-queries).|Yes|
    | CLICKHOUSE_SELECT |Executes 'SELECT ...' statements against PMM's [Query Analytics](../use/qan/index.md) ClickHouse database. Queries can use the {% raw %} **{{.ServiceName**}} and **{{.ServiceID}}**  {% endraw %} placeholders in query string. They match the target service name and service ID respectively.|Yes|

## Query parameters

Some query types accept parameters:

- `METRICS_INSTANT`
    - **lookback** (duration, optional): specifies how far in past to look back to metrics history. If this parameter is not specified, then query executed on the latest data. Example values: `30s`, `5m`, `8h`.
- `METRICS_RANGE`
    - **lookback** (duration, optional): specifies how far in past to look back to metrics history. If this parameter is not specified, then query executed on the latest data. Example values: `30s`, `5m`, `8h`.
    - **range** (duration, required): specifies time window of the query. This parameter is equal to [Prometheus API](https://prometheus.io/docs/prometheus/latest/querying/api/#range-queries).
    - **step** (duration, required): query resolution. This parameter is equal to [Prometheus API](https://prometheus.io/docs/prometheus/latest/querying/api/#range-queries).
- `POSTGRESQL_SELECT`
    - **all_dbs** (boolean, optional): execute query on all available databases in PostgreSQL instance. If this parameter is not specified, then query executed on the default database (the one that was specified when service was added to PMM).

To set the parameters of a query in the check form:
{.power-number}

1. Under the query, click **Add parameter**. A row with the **Parameter** and **Value** fields appears.
2. In **Parameter**, select or type the name of the parameter. The field suggests the names that the query type accepts: `lookback` for `METRICS_INSTANT`, `range`, `step` and `lookback` for `METRICS_RANGE`, and `all_dbs` for `POSTGRESQL_SELECT`.
3. In **Value**, enter the value of the parameter. Durations take a number with a unit such as `s`, `m` or `h`, for example `30s`, `5m` or `8h`. `all_dbs` takes `true` or `false`.

To remove a parameter, click the :material-trash-can-outline: **Remove parameter** icon in its row. You can set each parameter only once per query. PMM validates the parameters when you test or save the check, and rejects parameters that the query type doesn't accept, invalid values, and `METRICS_RANGE` queries without `range` or `step`.

## Troubleshooting and tips

When developing checks for PMM, you may encounter various issues. Here are solutions for common problems:

### Test fails or lists no services

**Test on service** lists only the services of the check's technology that have a `pmm-agent`. PMM Server's internal PostgreSQL database is never listed.

If a test fails, the error message shows the failed query or the script error, followed by the script's output. A test also fails if the **Advisors** option is disabled, or if the `pmm-agent` of the service is disconnected or too old for the check.

### Managing debug output
Debug mode generates excessive information in log files that can obscure important data. To disable debug logging, use `PMM_DEBUG=0`.

### Filtering logs
All check subsystem logs include the component=checks tag. Filter relevant logs with: `grep "component=checks" /path/to/pmm-managed.log`.

To follow the `pmm-managed` logs of a PMM Server that runs in Docker, run:

```sh
docker exec -it pmm-server supervisorctl tail -f pmm-managed
```

## Submit feedback
We welcome your feedback on the current process for developing and debugging checks. Send us your comments or post a question on the [Percona Forums](https://forums.percona.com/c/percona-monitoring-and-management-pmm/pmm-3/84).
