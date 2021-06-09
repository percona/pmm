# Architecture

PMM is a client/server application built by us with our own and third-party open-source tools.

```plantuml
@startuml "1 - PMM Context"
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Context.puml
!include docs/_images/plantuml_styles.puml
HIDE_STEREOTYPE()
'title PMM Context
caption PMM's client/server architecture
Person_Ext(user, "User")
System_Ext(monitored, "Monitored systems", "Servers, databases, services or applications")
System(pmm_client, "PMM Client", "Runs on every monitored host to extract metrics data from databases and services and forwards it to PMM Server")
System(pmm_server, "PMM Server", "Receives, stores and organizes metrics data from PMM Clients, presents it in web UI as graphs, charts, and tables")
System_Ext(platform, "Percona Platform", "Value-added services:\n- Security Threat Tool\n- DBaaS (Coming soon)")
Lay_D(user, pmm_client)
Lay_D(user, pmm_server)
Rel_R(monitored, pmm_client, "Metrics")
BiRel_R(pmm_client, pmm_server, " ")
BiRel_R(pmm_server, platform, " ")
Rel(user, pmm_server, " ")
Rel(user, pmm_client, " ")
@enduml
```

**PMM Server**

PMM Server is the heart of PMM. It receives data from clients, collates it and stores it. Metrics are drawn as tables, charts and graphs within [*dashboards*](dashboards/), each a part of the web-based [user interface](../using/interface.md).

**PMM Client**

PMM Client runs on every database host or node you want to monitor. The client collects server metrics, general system metrics, and query analytics data, and sends it to the server. Except when monitoring AWS RDS instances, a PMM Client must be running on the host to be monitored.

**Percona Platform**

[Percona Platform](../using/platform/) (in development) provides value-added services for PMM.

## PMM context

The PMM Client package provides:

- Exporters for each database and service type. When an exporter runs, it connects to the database or service instance, runs the metrics collection routines, and sends the results to PMM Server.
- `pmm-agent`: Run as a daemon process, it starts and stops exporters when instructed.
- `vmagent`: A VictoriaMetrics daemon process that sends metrics data (*pushes*) to PMM Server.

The PMM Server package provides:

- pmm-managed
- Query Analytics
- Grafana
- VictoriaMetrics

## PMM Server

![!image](../_images/PMM_Architecture_Client_Server.jpg)

<!-- incomplete replacement for above
```plantuml
@startuml "3 - PMM components - server"
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Component.puml
!include docs/_images/plantuml_styles.puml
HIDE_STEREOTYPE()
'title
caption PMM Server components
System_Ext(pmm_client, "PMM Client")
Boundary(pmm_server, "PMM Server") {
    Component(pmm_managed, "pmm-managed", "golang")
    Boundary(query_analytics, "Query Analytics") {

'        Boundary(clickhouse_server, "clickhouse-server"){
            ComponentDb(clickhouse, "pmm", "clickhouse")
 '       }

        Component(qan_api, "qan-api2", "golang")
        Component(qan_app, "qan-app", "typescript")
    }
   ComponentDb(victoriametrics, "VictoriaMetrics", "")
'    {        ComponentDb(vmdb, "VictoriaMetrics", "")    }
    Component(grafana, "Grafana", " ")
    Component(web_server, "Web server", "Nginx")
    ComponentDb(postgres, "Persistence", "PostgreSQL")
    Boundary(alerting, "Alerting") {
        Component(vmalert, "Integrated Alerting", " ")
        Component(alert_manager1, "Alertmanager 1", "Bundled")
    }
        Component(alert_manager2, "Alertmanager 2", " ")
'    Component(pmm_update, "??> PMM Update", "golang")
}
BiRel_R(pmm_client, pmm_managed, "Query Analytics metrics")
Rel(pmm_client, victoriametrics, "Metrics")
Rel_R(pmm_client, web_server, "HTTP")
Rel_R(pmm_managed, qan_api, " ")
Rel(pmm_managed, victoriametrics, " ")
Rel(qan_api, clickhouse, "Analytics")
Rel(grafana, qan_app, " ")
Rel_L(qan_app, qan_api, " ")
Rel_L(grafana, web_server, " ")
Rel(web_server, pmm_managed, " ")
Rel(pmm_managed, postgres, " ")
@enduml
```
-->

PMM Server includes the following tools:

- Query Analytics (QAN) enables you to analyze MySQL query performance over periods of time. In addition to the client-side QAN agent, it includes the following:

    - QAN API is the back-end for storing and accessing query data collected by the QAN agent running on a PMM Client.
    - QAN Web App is a web application for visualizing collected Query Analytics data.

- Metrics Monitor provides a historical view of metrics that are critical to a MySQL or MongoDB server instance. It includes the following:

    - [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics), a scalable time-series database. (Replaced [Prometheus](https://prometheus.io) in [PMM 2.12.0](../release-notes/2.12.0.md).)
    - [ClickHouse](https://clickhouse.tech/) is a third-party column-oriented database that facilitates the Query Analytics functionality.
    - [Grafana](http://docs.grafana.org/) is a third-party dashboard and graph builder for visualizing data aggregated (by VictoriaMetrics or Prometheus) in an intuitive web interface.
    - Percona Dashboards is a set of dashboards for Grafana developed by us.

### PMM Client

![!image](../_images/diagram.pmm.client-architecture.png)

<!--- incomplete C4 replacement for above
```plantuml
@startuml "3 - PMM components - client"
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Component.puml
!include docs/_images/plantuml_styles.puml
HIDE_STEREOTYPE()
'title PMM Client
caption PMM Client components
Person_Ext(admin, "Administrator")
Boundary(pmm_client, "PMM Client") {
'Component(pxb, "Percona XtraBackup", "")
'Component(pgb, "pgBackupRest", "")
'Component(pbm, "Percona Backup for MongoDB", "")
    Boundary(exporters, "exporters") {
    Boundary(ptools, "PT") {
        Component(pt_summary, "pt_summary", "")
    }
        Component(rds_exporter, "rds_exporter", "")
        Component(node_exporter, "node_exporter", "")
        Component(mysqld_exporter, "mysqld_exporter", "")
        Component(mongodb_exporter, "mongodb_exporter", "")
        Component(postgres_exporter, "postgres_exporter", "")
        Component(proxysql_exporter, "proxysql_exporter", "")
    }
    Component(pmm_admin, "pmm-admin", "golang")
    Component(pmm_agent, "pmm-agent", "golang")
    Component(vmagent, "vmagent", " ")
}
System_Ext(pmm_server, "PMM Server")
Rel(admin, pmm_admin, "Commands")
Rel_R(pmm_admin, pmm_agent, " ")
Rel(pmm_agent, exporters, "Runs")
Rel(exporters, vmagent, " ")
Rel(pmm_agent, pmm_server, " ")
Rel(vmagent, pmm_server, "Pushes to")
Rel(pmm_server, exporters, "Pulls from")
@enduml
```
-->

The PMM Client package consist of the following:

- `pmm-admin` is a command-line tool for managing PMM Client, for example, adding and removing database instances that you want to monitor. ([Read more.](../details/commands/pmm-admin.md)).
- `pmm-agent` is a client-side component a minimal command-line interface, which is a central entry point in charge for bringing the client functionality: it carries on client’s authentication, gets the client configuration stored on the PMM Server, manages exporters and other agents.
- `node_exporter` is an exporter that collects general system metrics.
- `mysqld_exporter` is an exporter that collects MySQL server metrics.
- `mongodb_exporter` is an exporter that collects MongoDB server metrics.
- `postgres_exporter` is an exporter that collects PostgreSQL performance metrics.
- `proxysql_exporter` is an exporter that collects ProxySQL performance metrics.
- `rds_exporter` is an exporter that collects Amazon RDS performance metrics.

- `azure_database_exporter` is an exporter that collects Azure database performance metrics.

To make data transfer from PMM Client to PMM Server secure, all exporters are able to use SSL/TLS encrypted connections, and their communication with the PMM server is protected by the HTTP basic authentication.


```plantuml
@startuml "2 - PMM Containers"
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
!include docs/_images/plantuml_styles.puml
HIDE_STEREOTYPE()
title PMM Client-Server interactions
caption PMM Client/PMM Server connections

System_Ext(monitored, "Monitored system", "Database, node or service")

Boundary(pmm_client, "PMM Client") {
    System(exporters, "exporters", "Collection of programs, one for each monitored system type")
    System(pmm_agent, "pmm-agent", "Invokes appropriate exporter on command")
    System(vmagent, "vmagent", "VictoriaMetrics agent")
}

Person_Ext(user, "User")

Boundary(pmm_server, "PMM Server") {
    System(pmm_managed, "pmm-managed", "Manages configuration, exposes API for other components")
    System(query_analytics, "Query Analytics", "Detailed database query data application")
    System(victoriametrics, "VictoriaMetrics", "Receives and stores metrics data (Prometheus-compatible)")
    System(grafana, "Grafana", "Data presentation")
}

Rel(user, grafana, "Uses")
Rel_R(monitored, pmm_agent, "Query Analytics metrics")
Rel(monitored, exporters, "Exports Metrics from")
BiRel_R(pmm_agent, pmm_managed, "Control API")
Rel(exporters, vmagent, " ")
Rel(pmm_agent, exporters, "Controls")
Rel(pmm_managed, query_analytics, " ")
Rel(pmm_managed, victoriametrics, " ")
Rel(query_analytics, grafana, " ")
Rel(victoriametrics, exporters, "Pulls from")
Rel(vmagent, victoriametrics, "Pushes to")

Lay_D(grafana, query_analytics)
Lay_R(pmm_managed, grafana)

@enduml
```

<!-- incomplete 'example deployment' diagram
```plantuml
@startuml PMM_context2
!includeurl https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Context.puml
LAYOUT_WITH_LEGEND()
' LAYOUT_TOP_DOWN()
LAYOUT_LEFT_RIGHT()
Enterprise_Boundary(platform, "Percona Platform") {
    System(stt, "Security Threat Tool")
    System(dbaas, "DB as a Service")
}
Enterprise_Boundary(enterprise, "Customer Systems") {
    Person_Ext(user, "User")

    System_Boundary(server_host, "Server host") {
        System(pmm_server, "PMM Server")
    }
    System_Boundary(enterprise1, "Customer system 1") {
        System(client1, "PMM Client (A)")
        System_Ext(monitored1, "Monitored database", "(MySQL)")
        Rel(monitored1, client1, "Metrics")
    }
    System_Boundary(enterprise2, "Customer system 2") {
        System(client2, "PMM Client (B)")
        System_Ext(monitored2, "Monitored database", "(PostgreSQL)")
        System_Ext(monitored3, "Monitored database", "(MongoDB)")
        Rel(monitored2, client2, "Metrics")
        Rel(monitored3, client2, "Metrics")
    }
    System_Boundary(enterprise3, "Customer system N") {
        System(client3, "PMM Client (X)")
        System_Ext(monitored4, "Monitored service")
        Rel(monitored4, client3, "Metrics")
    }
}
Rel(user, pmm_server, " ")
BiRel(client1, pmm_server, " ")
BiRel(client2, pmm_server, " ")
BiRel(client3, pmm_server, " ")
BiRel(pmm_server, stt, " ")
BiRel(pmm_server, dbaas, " ")
@enduml
```
-->