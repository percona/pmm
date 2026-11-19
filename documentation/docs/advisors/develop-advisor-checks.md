# Develop Advisor checks

When the built-in checks don't cover a rule that matters for your databases, for example an internal naming convention or a setting that your team requires, write your own check.

You create custom checks in the PMM UI, and PMM runs them on your services together with the [built-in checks](advisor-details.md).

A check queries your database, PMM metrics or Query Analytics, and turns the results into findings with a [Starlark](https://github.com/google/starlark-go) script, which uses Python syntax.

## Prerequisites

Before you create a check:
{.power-number}

1. Check that you have the **Admin** role. Only Admins can create, edit, clone, test and delete checks, because a check's queries run on your databases with the privileges of the PMM monitoring user. Editors can only view, run, enable and disable checks. Before you give other users access, review [the permissions of every role](../admin/roles/index.md).
2. Check that the **Advisors** option is turned on under **Configuration > Settings > Advisors**.
3. Check that PMM monitors at least one service of the technology that your check targets, so that you can test the check on it.

## Built-in and custom checks

Before you write a check from scratch, look for a built-in check that already does most of what you need. In **Advisors > Catalog**, the **Source** column shows **Built-in** for the checks that ship with PMM and **Custom** for the checks that you create.

You can run, test and turn off checks of both kinds, and open, copy or clone their scripts. Only custom checks can be edited or deleted, so to adapt a built-in check, clone it and edit the copy. The scripts of the built-in checks are good examples to start from.

## Create a check

Create a check from scratch, or clone a built-in check that is close to what you need and adapt the copy.

To create a check:
{.power-number}

1. Go to **Advisors > Catalog** and open the check form:

    - click the :material-plus: **Add advisor check** icon to start from scratch
    - double-click a check's row and click **Clone** in the details pane to start from that check. The clone's name starts as `custom_` followed by the name of the source check.

2. In **Name**, complete the ID after `custom_`, for example `custom_mysql_test_database`. You can't change it later.
3. Enter a **Summary**, a short title of the check, and a **Description** of what the check verifies.
4. In **Category**, select an existing category, such as **Security** or **Performance**, or type a new one.
5. In **Technology**, select the type of database that the check targets: **MySQL**, **PostgreSQL** or **MongoDB**. The check runs only on services of that type, and the query types you can choose depend on it.
6. Under **Queries**, [add the queries that collect the data](#collect-the-data-that-the-check-needs) that your check needs.
7. In **Script**, [write the script](#write-the-check-script) that turns the query results into findings.
8. To try the check before you save it, select a service in **Test on service** and click **Test**:

    - **Test results** shows the findings, or the error if a query or the script failed.
    - **Script output** shows what the script printed with `print()`.

    The test doesn't save the check and doesn't record any insight.

9. Click **Save check**. The check appears in the catalog as enabled, with **Custom** in the **Source** column.

## Collect the data that the check needs

Decide which data your check needs, and add one query for each set of data. A query can read your database, through the PMM Client that monitors it, or the metrics and Query Analytics data that PMM Server stores.

To add a query to a check:
{.power-number}

1. Under **Queries**, click **Add query**.
2. In **Type**, select the type that reads the data your check needs. The list offers only the types that fit the check's database:

    - `MYSQL_SHOW` or `MYSQL_SELECT` to run a `SHOW …` or `SELECT …` statement on a MySQL service
    - `POSTGRESQL_SELECT` to run a `SELECT …` statement on a PostgreSQL service, or `POSTGRESQL_SHOW` to read all its settings with `SHOW ALL`
    - a `MONGODB_*` type to run an admin command on the `admin` database of a MongoDB service:
        - `MONGODB_GETPARAMETER` for the server parameters, with the [getParameter command](https://docs.mongodb.com/manual/reference/command/getParameter/)
        - `MONGODB_BUILDINFO` for the version and build details, with the [buildInfo command](https://docs.mongodb.com/manual/reference/command/buildInfo/)
        - `MONGODB_GETCMDLINEOPTS` for the startup options, with the [getCmdLineOpts command](https://docs.mongodb.com/manual/reference/command/getCmdLineOpts/)
        - `MONGODB_REPLSETGETSTATUS` for the replica set status, with the [replSetGetStatus command](https://docs.mongodb.com/manual/reference/command/replSetGetStatus/)
        - `MONGODB_GETDIAGNOSTICDATA` for the [full-time diagnostic data](https://docs.mongodb.com/manual/administration/analyzing-mongodb-performance/#full-time-diagnostic-data-capture)
    - `METRICS_INSTANT` to read the current value of a metric that PMM collects, or `METRICS_RANGE` to read its values over a time range, with a [MetricsQL](https://docs.victoriametrics.com/MetricsQL.html) query
    - `CLICKHOUSE_SELECT` to run a `SELECT …` statement on the [Query Analytics](../use/qan/index.md) data

3. In **Query**, enter the query text. Leave it empty for `POSTGRESQL_SHOW` and the `MONGODB_*` types, which take none. To query only the service that the check runs on, use these placeholders:

    - {% raw %}`{{.ServiceName}}`{% endraw %} and {% raw %}`{{.NodeName}}`{% endraw %} in `METRICS_INSTANT` and `METRICS_RANGE` queries, for the names of the service and its node
    - {% raw %}`{{.ServiceName}}`{% endraw %} and {% raw %}`{{.ServiceID}}`{% endraw %} in `CLICKHOUSE_SELECT` queries, for the name and ID of the service

4. If the query type accepts parameters, click **Add parameter** under the query, select or type the parameter name in **Parameter**, and enter its **Value**. Durations take a number with a unit, such as `30s`, `5m` or `8h`. You can set each parameter only once per query:

    - **lookback** for `METRICS_INSTANT` and `METRICS_RANGE`, optional: how far back in the metrics history the query looks. Without it, the query reads the latest data.
    - **range** for `METRICS_RANGE`, required: the time window of the query, as in the [Prometheus range query API](https://prometheus.io/docs/prometheus/latest/querying/api/#range-queries)
    - **step** for `METRICS_RANGE`, required: the resolution of the query, as in the Prometheus range query API
    - **all_dbs** for `POSTGRESQL_SELECT`, optional: set it to `true` to run the query on every database of the PostgreSQL instance. Without it, the query runs on the database that was specified when the service was added to PMM.

The script receives the results in the order of the queries. To remove a query or a parameter, click its :material-trash-can-outline: **Remove query** or **Remove parameter** icon.

## Write the check script

Write the script that decides, from the query results, whether a service has an issue. Keep it fast: the script has 5 seconds to finish, and the whole check, including its queries, has 5 minutes.

The script runs in a sandbox on PMM Server, so it can't read or write files or access the network.

In the script, define a `check_context(docs, context)` function:

- `docs` is a read-only list with one item for each query, in the order of the queries. Each item holds the result of that query: the returned rows for SQL databases and the returned documents for MongoDB.
- `context` is a dictionary with [helper functions](#use-the-helper-functions).

Indent the script with spaces, not tabs.

### Return findings

To tell PMM what the check found, return a list of findings, one for each issue. Each finding becomes a **Failed** insight.

An empty list means that the check passed, and the service gets an **OK** insight. Learn [what each insight status means](advisors.md#understand-the-status-of-an-insight).

Each finding is a dictionary with these keys:

- `summary` (required): a short description of the issue
- `severity` (required): `critical`, `error`, `warning` or `info`, in lowercase
- `description` (optional): details of the issue
- `read_more_url` (optional): a valid URL that links to the steps that fix the issue
- `labels` (optional): a dictionary of string keys and values that describe the finding

PMM rejects any other severity, including the `emergency`, `alert`, `notice` and `debug` levels that earlier versions accepted, and the check gets the **Error** status.

If the check can't evaluate the data, return a string or call `fail()` instead. The check then gets the **Error** status with your message.

### Report one finding per service

To keep **Advisors > Insights** easy to read, return one finding per service, or one per database for checks that examine per-database objects, rather than one finding per affected object.

The list tells findings apart by their summary and labels, so per-object findings with the same summary fill the list with rows that look identical.

To report several objects in one finding:

- put the count in the summary, for example `3 relation(s) with unused indexes in database app`
- list the objects in the description
- set labels such as `database` and `count`

If the check finds issues at different severity levels, return one finding per severity level.

### Use the helper functions

Use the helper functions to debug your script and to handle versions and IP addresses without writing that logic yourself.

To debug the script, call `print`: when you test the check, **Script output** shows what it wrote. To stop the check with an error, call `fail`.

The `context` dictionary provides these functions:

- `parse_version(version)` parses a version string and returns a dictionary with the `major`, `minor`, `patch`, `rest`, `num` and `numrest` keys.
- `format_version_num(num)` formats a version number, such as `80036`, as a version string, such as `8.0.36`.
- `current_date()` returns today's date in UTC as a `YYYY-MM-DD` string.
- `days_until(date)` returns the number of whole days from today (UTC) until a `YYYY-MM-DD` date. The number is negative for a date in the past.
- `ip_is_private(address)` returns `True` if the IP address is private, or if the network in CIDR notation, such as `10.0.0.0/8`, lies fully inside a private address block. It returns `False` if it isn't, and `None` if the argument isn't a valid address or network.

Get a function from `context` before you call it, for example `parse_version = context.get("parse_version", fail)`.

### Try an example check

To see how queries and a script work together, create a check that reports the MySQL version, the uptime and the recent load average of each MySQL service.

In the check form, set **Technology** to **MySQL**, then:
{.power-number}

1. Add a `MYSQL_SHOW` query with the text `VARIABLES`.
2. Add a `METRICS_INSTANT` query with the text {% raw %}`mysql_global_status_uptime{service_name=~"{{.ServiceName}}"}`{% endraw %}.
3. Add a `METRICS_RANGE` query with the text {% raw %}`avg by (node_name) (avg_over_time(node_load1{node_name=~"{{.NodeName}}"}[5m]))`{% endraw %}, and set its `range` parameter to `15m` and its `step` parameter to `5m`.
4. In **Script**, paste this script. It reads the results of the three queries from `docs[0]`, `docs[1]` and `docs[2]`, in the order that you added them. Each series of a `METRICS_RANGE` result holds its data points in `values`, as `[timestamp, value]` pairs:

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
                    "severity": "info",
                    "labels": {},
                })

        uptimeNow = int(int(docs[1][0]["value"][1])/60)
        results.append({
            "summary": "MySQL uptime {} min".format(uptimeNow),
            "description": "Current uptime is {} min".format(uptimeNow),
            "read_more_url": "",
            "severity": "info",
            "labels": {},
        })

        dataPoints = []
        for row in docs[2][0]["values"]:
            dataPoints.append(row[1])

        results.append({
            "summary": "Node load average for the last 15 minutes {}".format(dataPoints),
            "description": "Data points {}".format(dataPoints),
            "read_more_url": "",
            "severity": "info",
            "labels": {},
        })

        return results
    ```

5. Select a MySQL service in **Test on service** and click **Test**. **Test results** shows 3 findings: the version, the uptime and the load average.

## Run the check and review its results

Your check runs on every [schedule](schedule-advisor-checks.md) set to **All checks**, together with the built-in checks. To see its results before the next scheduled run:
{.power-number}

1. Go to **Advisors > Catalog**, find your check and click the :material-play-outline: **Run** icon in its row.
2. In the confirmation message, click **View results**. **Advisors > Insights** shows the results of that run for each service.
3. To follow your check across all runs, click the :material-dots-vertical: icon in one of its insights and select **Filter by name**.

To make sure that the run reached all your services, check [the coverage numbers of the run](advisors.md#understand-the-coverage-numbers-of-a-run) in **Advisors > Run history**.

## Change or delete a check

Change a custom check when your rules change, and delete it when you no longer need it.

Built-in checks can't be changed or deleted: to adapt one, [clone it](#create-a-check), and turn off the original with its **Status** switch.

In **Advisors > Catalog**:

- click the :material-pencil-outline: **Edit** icon in a check's row, or **Edit** in its details pane, to change any field except **Name**. Test your changes before you click **Save check**.
- click the :material-trash-can-outline: **Delete** icon in a check's row and confirm to delete the check. You can't undo the deletion.

## Troubleshooting

Fix the most common problems with custom checks: no services to test on, tests that fail, and checks that you need to debug in the logs.

### Test on service lists no services

If **Test on service** lists no services, PMM doesn't monitor any service of the check's database type through a PMM Client. [Add a service](../install-pmm/install-pmm-client/connect-database/index.md) of that type to PMM, or change the check's **Technology**. PMM Server's internal PostgreSQL database is never listed.

### The test fails

If a test fails, find the cause:
{.power-number}

1. Read the error message. It shows the failed query or the script error, followed by the script's output.
2. Check that each parameter fits its query type and has a valid value, and that every `METRICS_RANGE` query has `range` and `step`. These problems also stop you from saving the check.
3. Check that the **Advisors** option is turned on under **Configuration > Settings > Advisors**.
4. Check that the PMM Client of the service is connected. If it's too old for the check, [upgrade PMM Client](../pmm-upgrade/upgrade_client.md) on that node, or test on another service.

### Find the check logs

To find out why a check fails, read the `pmm-managed` logs on PMM Server. Every log line of the checks includes `component=checks`:
{.power-number}

1. To follow the logs of a PMM Server that runs in Docker, run:

    ```sh
    docker exec -it pmm-server supervisorctl tail -f pmm-managed
    ```

2. To list only the check logs, filter the log file:

    ```sh
    docker exec pmm-server grep "component=checks" /srv/logs/pmm-managed.log
    ```

3. If the logs don't show enough detail, set `PMM_DEBUG=1` on the PMM Server container to turn on verbose logging. Turn it off again when you're done, because verbose logs grow quickly and hide important messages.

## Submit feedback

Tell us how we can improve developing and debugging checks: post your comments or questions on the [Percona Forums](https://forums.percona.com/c/percona-monitoring-and-management-pmm/pmm-3/84).

## Related topics

- [Run Advisor checks and review results](advisors.md)
- [Schedule Advisor checks](schedule-advisor-checks.md)
- [List of advisors and checks](advisor-details.md)
