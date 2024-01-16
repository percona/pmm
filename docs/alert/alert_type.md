# Alert types

Percona Alerting is powered by Grafana infrastructure. It leverages Grafana's advanced alerting capabilities and provides pre-configured Alert Rule Templates that simplify creating powerful alerting rules.

Depending on the datasources that you want to query, and the complexity of your required evaluation criteria, Percona Alerting enables you to create the following types of alerts:

- **Percona templated alerts**: alerts based on a set of Percona-supplied templates with common events and expressions for alerting.
If you need custom expressions on which to base your alert rules, you can also create your own templates.
- **Grafana managed alerts**: alerts that handle complex conditions and can span multiple different data sources like SQL, Prometheus, InfluxDB, etc. These alerts are stored and executed by Grafana.

The Alerting page contains are split into eight tabs: Fired Alerts, Alert Rules, Alert Rule Templates, Contact Points, Notification Policies, Silences, Alert Groups and Admin.

![!](../_images/alerting-page-tabs.png)
