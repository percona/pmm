# Version 1 checks for PMM 2.27 and older
 Advisor checks created for PMM 2.27 and older use a slightly different structure than checks for 2.28.
 This is because, compared to 2.28 checks, 2.27 checks do not support:

- Multiple queries
- Victoria Metrics as a data source
- No database **Family** field

## Format for v.1 checks
Checks for PMM 2.27 and older use the following format:

??? note alert alert-info "Version 1 Checks Format"

    ```yaml
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
    ```

??? note alert alert-info "Realistic Example of Check in v.1 Format"

    ```yaml
    ---
    checks:
      - version: 1
        name: mongodb_version
        summary: MongoDB Version
        description: This check returns warnings if MongoDB/PSMDB version is not the latest one.
        type: MONGODB_BUILDINFO
        script: |-
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
    ```


## Security checks in PMM 2.26 and older
PMM 2.26 and older included a set of security checks grouped under the **Security Threat Tool** option.

With the 2.27 release, security checks have been renamed to Advisor checks, and the **Security Threat Tool** option in the PMM Settings was renamed to **Advisors**.

## Checks script

The check script assumes that there is a function with a fixed name, that accepts a _list_ of _docs_ containing returned rows for SQL databases and documents for MongoDB. It returns zero, one, or several check results that are then converted to alerts.

PMM 2.12.0 and earlier function name is `check`, while newer versions use name `check_context`. Both have the same meaning.

### Function signature

The function signature should be `check_context(docs, context)`, where `docs` is lists of docs (one doc represents one row for SQL DBMS and one document for MongoDB).

## Check severity levels
PMM can display failed checks as **Critical**, **Major** or **Trivial**. These three severity levels correspond to the following severity types in the check source:

 - **Critical**: emergency, alert, critical
 - **Major**: warning
 - **Trivial**: notice, info, debug

## Check fields

Checks can include the following fields:

- **Version** (integer, required): defines what other properties are expected, what types are supported, what is expected from the script and what it can expect from the execution environment, etc.
- **Name** (string, required): defines machine-readable name (ID).
- **Summary** (string, required): defines short human-readable description.
- **Description** (string, required): defines long human-readable description.
- **Interval** (string/enum, optional): defines running interval. Can be one of the predefined intervals in the UI: Standard, Frequent, Rare.
- **Type** (string/enum, required): defines the query type and the PMM Service type for which the advisor runs. Check the list of available types for version 1 checks in the table below.
- **Script** (string, required): contains a small Starlark program that processes query results, and returns check results. It is executed on the PMM Server side.
- **Category** (string, required): specifies a custom or a default advisor check category. For example: Performance, Security.
- **Query** (string, can be absent if the type defines the whole query by itself): The query is executed on the PMM Client side and contains query specific for the target DBMS.

## Check types

Expand the table below for the list of checks types that you can use to define your query type and the PMM Service type for which the check will run.

??? note alert alert-info "Check Types table"

    | Check type             |  Description                                                                                                                                                                                      | "query" required (must be empty if "No") |
    |------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------|
    | `MYSQL_SHOW`           | Executes `SHOW …` clause against MySQL database.                                                                                                                                                  | Yes                                      |
    | `MYSQL_SELECT`         | Executes `SELECT …` clause against MySQL database.                                                                                                                                                | Yes                                      |
    | `POSTGRESQL_SHOW`      | Executes `SHOW ALL` command against PosgreSQL database.                                                                                                                                           | No                                       |
    | `POSTGRESQL_SELECT`    | Executes `SELECT …` clause against PosgreSQL database.                                                                                                                                            | Yes                                      |
    | `MONGODB_GETPARAMETER` | Executes `db.adminCommand( { getParameter: "*" } )` against MongoDB's "admin" database. For more information, see [getParameter](https://docs.mongodb.com/manual/reference/command/getParameter/) | No                                       |
    | `MONGODB_BUILDINFO`    | Executes `db.adminCommand( { buildInfo:  1 } )` against MongoDB's "admin" database.  For more information, see [buildInfo](https://docs.mongodb.com/manual/reference/command/buildInfo/)          | No                                       |

## Develop version 1 checks
To develop custom checks for PMM 2.26 and 2.27:


1. Install the latest PMM Server and PMM Client builds following the [installation instructions](https://www.percona.com/software/pmm/quickstart#).
2. Run PMM Server with special environment variables:

    - `PMM_DEBUG=1` to enable debug output that would be useful later;
    - `PERCONA_TEST_CHECKS_FILE=/srv/custom-checks.yml` to use checks from the local files instead of downloading them from Percona Platform.
    - `PERCONA_TEST_CHECKS_DISABLE_START_DELAY=true` to disable the default check execution start delay. This is currently set to one minute, so that checks run upon system start.
    - `PERCONA_TEST_CHECKS_RESEND_INTERVAL=2s` to define the frequency for sending the SA-based alerts to Alertmanager.

    ```sh
    docker run -p 80:80 -p 443:443 --name pmm-server \
    -e PMM_DEBUG=1 \
    -e PERCONA_TEST_CHECKS_FILE=/srv/custom-checks.yml \
    -e PERCONA_TEST_CHECKS_DISABLE_START_DELAY=true \
    -e PERCONA_TEST_CHECKS_RESEND_INTERVAL=2s \
    perconalab/pmm-server:dev-latest
    ```

3.  Log in to Grafana with credentials **admin/admin**.

4. Go to **Configuration > Settings > Advanced Settings** and enable the **Security Threat Tool** option.

5.  Create `/srv/custom-checks.yml` inside the `pmm-server` container with the content of your check.

6.  The checks will run according to the time interval defined on the UI. You can see the result of running the check on the home dashboard:

    ![!](../../_images/HomeDashboard.png)

7.  Click on the number of failed checks to open the Failed Checks dashboard:

    ![!](../../_images/FailedChecks.png)

8.  Go into Docker container to output the logs of pmm-managed and read check logs:

```
# get inside the container
docker exec -it pmm-server bash
# print and watch the logs
supervisorctl tail -f pmm-managed

```