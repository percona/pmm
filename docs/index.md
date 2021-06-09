# Welcome

**Percona Monitoring and Management** (PMM) is a free, open-source monitoring tool for MySQL, PostgreSQL, MongoDB, and ProxySQL, and the servers they run on.

- PMM **collects** from databases and their hosts thousands of out-of-the-box performance **metrics**.

- The PMM [web UI](using/interface.md) **visualizes data** in [dashboards](details/dashboards/).

- Additional features include checking databases for [security threats](using/platform/security-threat-tool.md).

!!! alert alert-success ""
    This is for the latest release, **PMM {{release}}** ([Release Notes](release-notes/{{release}}.md)).

Percona Monitoring and Management helps you improve the performance of databases, simplify their management, and strengthen their security. It is efficient, quick to [set up](setting-up/index.md) and easy to use.

A minimal PMM set-up comprises one [server](details/architecture.md#pmm-server) and a [client agent](details/architecture.md#pmm-client) on every system you want to monitor. Clients send metrics to the server which stores, collates and displays them.

Here's how the web UI home page looks on our <a href='https://pmmdemo.percona.com/' target='_blank'>live demo system</a>. (It's free to use---why not try it?)

<a href='https://pmmdemo.percona.com/' target='_blank'><img src="_images/PMM_Home_Dashboard.jpg" width=600px class="imgcenter"/></a>

PMM can run as a cloud service, on-prem, or across hybrid platforms. It's supported by our [legendary expertise][PERCONA_SERVICES] in open source databases, and by a vibrant developer and user [community].

## Next steps

The [Quickstart installation guide](https://www.percona.com/software/pmm/quickstart) shows how to run PMM Server as a Docker container, and how to install PMM Client on Ubuntu or Red Hat Linux hosts.

Full instructions for setting up are in:

- [Setting up PMM Server](setting-up/server/index.md)
- [Setting up PMM Client](setting-up/client/index.md)

## Reading guide

Links to popular sections.

```plantuml format="svg_object" width="90%" height="90%" source="_resources/diagrams/Navigation_Topics.puml"
```

??? note "Full section map (click to show/hide)"
    ```plantuml format="svg_object" width="100%" height="100%" source="_resources/diagrams/Navigation_Map.puml"
    ```

[PERCONA_SERVICES]: https://www.percona.com/services
[community]: https://www.percona.com/forums/questions-discussions/percona-monitoring-and-management
