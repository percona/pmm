# Develop Advisor checks
 
PMM offer sets of checks that can detect common security threats, performance degradation, data loss and data corruption.
 
As a developer, you can create custom checks to cover additional use cases, relevant to your specific database infrastructure.
 
## Check components
 
A check is a combination of:
 
- A query for extracting data from the database.
- Python script for converting extracted data into check results. This is actually a [Starlark](https://github.com/google/starlark-go) script, which is a Python dialect that adds more imperative features than Python. The script's execution environment is sandboxed, and no I/O can be done from it.
 
All checks are self-contained in the first phase, as well as in most of the planned phases.
 
This means that extracted data is processed on the PMM side and not sent back to the SaaS.


## Backend

1. pmm-managed checks that the installation is opted-in for checks.
2. pmm-managed downloads files with checks from SaaS.
3. pmm-managed verifies file signatures using a list of hard-coded public keys. At least one signature should be correct.
4. pmm-managed sends queries to pmm-agent and gathers results.
5. pmm-managed executes check scripts that produce alert information.
6. pmm-managed sends alerts to Alermanager.
   - Due to Alertmanager design, pmm-managed has to send and re-send alerts to it much more often than the frequency with which checks are executed. This expected behaviour is not important for using checks but is important for understanding how checks work.
   - Currently, Prometheus is not involved.

![!](../_images/BackendChecks.png)

## Frontend
PMM uses Aletmanager API to get information about failed checks and show them on the UI:

![!](../_images/FrontEndChecks.png)
 
 
## Check fields
Checks include the following fields:  

- **Version** (integer, required): defines what other properties are expected, what types are supported, what is expected from the script and what it can expect from the execution environment, etc.
- **Name** (string, required): defines machine-readable name (ID).
- **Summary** (string, required): defines short human-readable description.
- **Description** (string, required): defines long human-readable description.
- **Interval** (string/enum, optional): defines running interval. Can be one of the predefined intervals in the UI: Standard, Frequent, Rare.
- **Type** (string/enum, required): defines the query type and the PMM Service type for which the advisor runs. Check the list of available types in the table below.
- **Query** (string, can be absent if the type defines the whole query by itself): The query is executed on the PMM Client side and contains query specific for the target DBMS.
- **Script** (string, required): contains a small Starlark program that processes query results, and returns check results. It is executed on the PMM Server side.
 
## Checks script
 
The check script assumes that there is a function with a fixed name, that accepts a _list_ of _docs_ containing returned rows for SQL databases and documents for MongoDB. It returns zero, one, or several check results that are then converted to alerts.
 
PMM 2.12.0 and earlier function name is **check**, while newer versions use name **check_context**. Both have the same meaning.

### Function signature
 
The function signature should be **check_context** (docs, context), where **docs** is lists of docs (one doc represents one row for SQL DBMS and one document for MongoDB).

## Check severity levels
PMM can display failed checks as **Critical**, **Major** or **Trivial**. These three severity levels correspond to the following severity types in the check source:
 
 - **Critical**: emergency, alert, critical
 - **Major**: warning  
 - **Trivial**: notice, info, debug
 
## Check types

Expand the table below for the list of checks types that you can use to define your query type and the PMM Service type for which the check will run.
 
??? note alert alert-info "Check Types table (click to show/hide)"

    | Check type  |  Description | "query" required (must be empty if "No")   |  
    |---|---|---|
    | MYSQL_SHOW |Executes 'SHOW …' clause against MySQL database. This check is available  starting with PMM 2.6 |Yes|
    | MYSQL_SELECT    |     Executes 'SELECT …' clause against MySQL database. This check is available  starting with PMM 2.6        |Yes|
    | POSTGRESQL_SHOW     |    Executes 'SHOW ALL' command against PosgreSQL database. This check is available  starting with PMM 2.6.       |No|
    | POSTGRESQL_SELECT      | Executes 'SELECT …' clause against PosgreSQL database. This check is available  starting with PMM 2.6.    |Yes|
    | MONGODB_GETPARAMETER     | Executes db.adminCommand( { getParameter: "*" } ) against MongoDB's "admin" database. This check is available  starting with PMM 2.6. For more information, see [getParameter](https://docs.mongodb.com/manual/reference/command/getParameter/)| No|
    | MONGODB_BUILDINFO    | Executes db.adminCommand( { buildInfo:  1 } ) against MongoDB's "admin" database. This check is available  starting with PMM 2.6. For more information, see [buildInfo](https://docs.mongodb.com/manual/reference/command/buildInfo/) | No|
    | MONGODB_GETCMDLINEOPTS          |    Executes db.adminCommand( { getCmdLineOpts: 1 } ) against MongoDB's "admin" database. This check is available  starting with PMM 2.7. For more information, see [getCmdLineOpts](https://docs.mongodb.com/manual/reference/command/getCmdLineOpts/) |No|
    | MONGODB_REPLSETGETSTATUS     |   Executes db.adminCommand( { replSetGetStatus: 1 } ) against MongoDB's "admin" database. This check is available  starting with PMM 2.27. For more information, see  [replSetGetStatus](https://docs.mongodb.com/manual/reference/command/replSetGetStatus/) |No|
    | MONGODB_GETDIAGNOSTICDATA |Executes db.adminCommand( { getDiagnosticData: 1 } ) against MongoDB's "admin" database. This check is available  starting with PMM 2.27. For more information, see [MongoDB Performance](https://docs.mongodb.com/manual/administration/analyzing-mongodb-performance/#full-time-diagnostic-data-capture)| No|
    
## Develop custom checks
 
1. Install the latest PMM Server and PMM Client builds following the [installation instructions](https://www.percona.com/software/pmm/quickstart#). 
2. Run PMM Server with special environment variables:
 
    - _PMM_DEBUG=1_ to enable debug output that would be useful later;
    - _PERCONA_TEST_CHECKS_FILE=/srv/custom-checks.yml_ to use checks from the local files instead of downloading them from the SaaS.
    - _PERCONA_TEST_CHECKS_DISABLE_START_DELAY=true_ to disable the default check execution start delay. This is currently set to one minute, so that checks run upon system start.
    - _PERCONA_TEST_CHECKS_RESEND_INTERVAL=2s_ to define the frequency for sending the SA-based alerts to Alertmanager.
 
    ```
    docker run -p 80:80 -p 443:443 --name pmm-server \
    -e PMM_DEBUG=1 \
    -e PERCONA_TEST_CHECKS_FILE=/srv/custom-checks.yml \
    -e PERCONA_TEST_CHECKS_DISABLE_START_DELAY=true \
    -e PERCONA_TEST_CHECKS_RESEND_INTERVAL=2s \
    perconalab/pmm-server:dev-latest
    ```
 
3.  Log in to Grafana with credentials **admin/admin**.
 
4. Go to **Configuration > Settings > Advanced Settings** and enable **Advisors**. For PMM 2.26 and older this option is called **Security Threat Tool**.
 
4.  Create _/srv/custom-checks.yml_ inside a Docker container with the content of your check.
 
5.  The checks will run according to the time interval defined on the UI. You can see the result of running the check on the home dashboard:
 
    ![!](../_images/HomeDashboard.png)
 
6.  Click on the number of failed checks to open the Failed Checks dashboard:
 
    ![!](../_images/FailedChecks.png)
 
7.  Go into Docker container to output the logs of pmm-managed and read check logs:
 
```
# get inside the container
docker exec -it pmm-server bash
# print and watch the logs
supervisorctl tail -f pmm-managed
 
```

### Format
To create checks use the following format:

=== "Check format"
            ---
        checks:
        - version: 1
            name: example
            summary: Example check
            description: This check is just an example.
            type: MONGODB_BUILDINFO
            script: |
            def check(docs):
                # for compatibility with PMM Server < 2.12
                context = {
                    "format_version_num": format_version_num,
                    "parse_version": parse_version,
                }
                return check_context(docs, context)


            def check_context(docs, context):
                # `docs` is a frozen (deeply immutable) list of dicts where each dict represents a single document in result set.
                # `context` is a dict with additional functions.
                #
                # Global `print` and `fail` functions are available.
                #
                # `check_context` function is expected to return a list of dicts that are then converted to alerts;
                # in particular, that list can be empty.
                # Any other value (for example, string) is treated as script execution failure
                # (Starlark does not support Python exceptions);
                # it is recommended to use global function `fail` for that instead.

                format_version_num = context.get("format_version_num", fail)
                parse_version = context.get("parse_version", fail)

                print("first doc =", repr(docs[0]))

                return [{
                    "summary": "Example summary",
                    "description": "Example description",
                    "severity": "warning",
                    "labels": {
                        "version": format_version_num(10203),
                    }
                }]

=== "Realistic example"
            ---
        checks:
        - version: 1
            name: mongodb_version
            summary: MongoDB Version
            description: This check returns warnings if MongoDB/PSMDB version is not the latest one.
            type: MONGODB_BUILDINFO
            script: |
            LATEST_VERSIONS = {
                "mongodb": {
                    "3.6": 30620,  # https://docs.mongodb.com/manual/release-notes/3.6/
                    "4.0": 40020,  # https://docs.mongodb.com/manual/release-notes/4.0/
                    "4.2": 40210,  # https://docs.mongodb.com/manual/release-notes/4.2/
                    "4.4": 40401,  # https://docs.mongodb.com/manual/release-notes/4.4/
                },
                "percona": {
                    "3.6": 30620,  # https://www.percona.com/downloads/percona-server-mongodb-3.6/
                    "4.0": 40020,  # https://www.percona.com/downloads/percona-server-mongodb-4.0/
                    "4.2": 40209,  # https://www.percona.com/downloads/percona-server-mongodb-4.2/
                    "4.4": 40401,  # https://www.percona.com/downloads/percona-server-mongodb-4.4/
                },
            }


            def check(docs):
                # for compatibility with PMM Server < 2.12
                context = {
                    "format_version_num": format_version_num,
                    "parse_version": parse_version,
                }
                return check_context(docs, context)


            def check_context(docs, context):
                # `docs` is a frozen (deeply immutable) list of dicts where each dict represents a single document in result set.
                # `context` is a dict with additional functions.
                #
                # Global `print` and `fail` functions are available.
                #
                # `check_context` function is expected to return a list of dicts that are then converted to alerts;
                # in particular, that list can be empty.
                # Any other value (for example, string) is treated as script execution failure
                # (Starlark does not support Python exceptions);
                # it is recommended to use global function `fail` for that instead.

                """
                This check returns warnings if MongoDB/PSMDB version is not the latest one.
                """

                format_version_num = context.get("format_version_num", fail)
                parse_version = context.get("parse_version", fail)

                if len(docs) != 1:
                    return "Unexpected number of documents"

                info = docs[0]

                # extract information
                is_percona = 'psmdbVersion' in info

                # parse_version returns a dict with keys: major, minor, patch, rest, num
                version = parse_version(info["version"])
                print("version =", repr(version))
                num = version["num"]
                mm = "{}.{}".format(version["major"], version["minor"])

                results = []

                if is_percona:
                    latest = LATEST_VERSIONS["percona"][mm]
                    if latest > num:
                        results.append({
                            "summary": "Newer version of Percona Server for MongoDB is available",
                            "description": "Current version is {}, latest available version is {}.".format(format_version_num(num), format_version_num(latest)),
                            "severity": "warning",
                            "labels": {
                                "current": format_version_num(num),
                                "latest":  format_version_num(latest),
                            },
                        })

                    return results

                if True:  # MongoDB
                    latest = LATEST_VERSIONS["mongodb"][mm]
                    if latest > num:
                        results.append({
                            "summary": "Newer version of MongoDB is available",
                            "description": "Current version is {}, latest available version is {}.".format(format_version_num(num), format_version_num(latest)),
                            "severity": "warning",
                            "labels": {
                                "current": format_version_num(num),
                                "latest":  format_version_num(latest),
                            },
                        })

                    return results
## Security checks in PMM 2.26 and older
PMM 2.26 and older included a set of security checks grouped under the **Security Threat Tool** option.
 
With the 2.27 release, security checks have been renamed to Advisor checks, and the **Security Threat Tool** option in the PMM Settings was renamed to **Advisors**.


## Submit feedback
 We welcome your feedback on the current process for developing and debugging checks. Send us your comments over [Slack](https://percona.slack.com) or post a question on the [Percona Forums](https://forums.percona.com/).

