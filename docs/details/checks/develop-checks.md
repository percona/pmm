# Develop security checks

The Security Threat Tool in PMM offers a set of checks that can detect common security threats, performance degradation, data loss and data corruption.

As a developer, you can create custom checks to cover additional use cases that are relevant to your specific database infrastructure.

## Check components

A check is a combination of:

- SQL query or MongoDB query document for extracting data from the database.
- Python script for converting extracted data into check results. This is actually a [Starlark](https://github.com/google/starlark-go) script – a Python dialect that adds more imperative features from Python. Script's execution environment is sandboxed so no I/O can be done from it.

All checks in the first phase (and most of the planned ones) are self-contained. This means that extracted data is processed on the PMM side and not sent back to the SaaS.

On the other hand, checks results and other metadata can be sent to SaaS to implement a history feature if the user opted-in and is entitled.

For example, below is a single check that returns the static result:

```yaml
---
checks:
  - version: 1
    name: example
    summary: Example check
    description: This check is just an example.
    tiers: [anonymous, registered]
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

Here is a much more realistic example:

```yaml
---
checks:
  - version: 1
    name: mongodb_version
    summary: MongoDB Version
    description: This check returns warnings if MongoDB/PSMDB version is not the latest one.
    tiers: [anonymous, registered]
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
```

## Check fields

- **Version** (integer, required) defines what other check properties are expected, what types are supported, what is expected from the script and what it can expect from the execution environment, etc.
- **Name** (string, required) defined machine-readable name (ID).
- **Summary** (string, required) defines short human-readable summary.
- **Description** (string, required) defines long human-readable description.
- **Type** (string/enum, required) defines PMM Service type for which check is performed and query type.
- **Query** (string, optional) contains a SQL query or MongoDB query document (as a string with proper quoting) which is executed on the PMM Client side. It may be absent if type defines the whole query by itself.
- **Script** (string, required) contains a small Python program that processes query results, and returns check results. It is executed on the PMM Server side.

Check script assumes that there is a function with a fixed name _check_ that accepts a _list_ of _dicts_ containing returned rows for SQL databases and documents for MongoDB. It returns zero, one, or several check results that are then converted to alerts.

Another function that should be implemented is **check_context**. PMM 2.12.0 and earlier use **context**, while newer versions use **check_context**. Both have the same meaning.

The single query means that currently you cannot implement some advanced checks that would require several queries (and can't be implemented using SQL UNION).

Checks format and the current STT UI use different terminology for severities. Here is how different formats show up on the UI:

| Format    | UI       |
| --------- | -------- |
| emergency |          |
| alert     |          |
| critical  |          |
| error     | Critical |
| warning   | Major    |
| notice    | Trivial  |
| info      |          |
| debug     |          |

## Backend

![!](../../../_images/BackendSTT.png)

1. pmm-managed checks that this installation is opted-in into STT.
2. pmm-managed downloads checks file from SaaS.
3. pmm-managed verifies file signatures using a list of hard-coded public keys. At least one signature should be correct.
4. pmm-managed sends queries to pmm-agent and gathers results.
5. pmm-managed executes check scripts that produce alert information.
6. pmm-managed sends alerts to Alertmanager.
   - Due to Alertmanager design, pmm-managed has to send and re-send alerts to it much more often than the frequency of check execution. That's expected, and not important for using STT, but important for understanding how it works.
   - Currently, Prometheus is not involved.

## Frontend

![!](../../../_images/FrontEndSTT.png)

Our UI in Grafana uses Alertmanager API v2 to get information about failed security checks.

## Develop custom checks

1.  Download the latest PMM Server and PMM Client builds:

    - PMM Server: [percona/pmm-server:2](https://www.percona.com/software/pmm/quickstart#)
    - PMM Client: _pmm2-client-2.9.1.tar.gz_

2.  Run PMM Server with special environment variables:

    - _PMM_DEBUG=1_ to enable debug output that would be useful later;
    - _PERCONA_TEST_CHECKS_FILE=/srv/custom-checks.yml_ to use checks from the local files instead of downloading them from the SaaS
    - _PERCONA_TEST_CHECKS_DISABLE_START_DELAY=true_ to disable the default check execution start delay, which is currently set to 1 minute, so that checks are run immediately upon system start
    - _PERCONA_TEST_CHECKS_RESEND_INTERVAL=2s_ to define the frequency for sending the SA-based alerts to Alertmanager.

    ```
    docker run -p 80:80 -p 443:443 --name pmm-server \

    -e PMM_DEBUG=1 \
    -e PERCONA_TEST_CHECKS_FILE=/srv/custom-checks.yml \
    -e PERCONA_TEST_CHECKS_DISABLE_START_DELAY=true
    -e PERCONA_TEST_CHECKS_RESEND_INTERVAL=2s \
    perconalab/pmm-server:dev-latest
    ```

3.  Log in to Grafana (admin/admin) and enable STT in the settings: http://127.0.0.1/graph/d/pmm-settings/pmm-settings

    ![!](../../../_images/Grafana.png)

4.  Create _/srv/custom-checks.yml_ inside a Docker container with the content from the Security Advisor (Security Threat Tool) section above.

5.  STT checks will run with a time interval defined via UI. You can see the result of running the advisor on the home dashboard:

    ![!](../../../_images/HomeDashboard.png)

6.  Click on the number of failed checks to open the Failed Checks dashboard:

    ![!](../../../_images/FailedChecks.png)

7.  Go into Docker container to output the logs of pmm-managed and read STT logs:

    ```
    # get inside the container
    docker exec -it pmm-server bash
    # print and watch the logs
    supervisorctl tail -f pmm-managed
    ```

## Sumbit feedback

We welcome your feedback on the current process for developing and debugging checks. Send us your comments over [Slack](percona.slack.com) or post a question on the [Percona Forums](https://forums.percona.com/).
