# Alertmanager integration

Alertmanager manages alerts, de-duplicating, grouping, and routing them to the appropriate receiver or display component.

This section lets you configure how VictoriaMetrics integrates with an external Alertmanager.

!!! hint alert alert-success "Tip"
    If possible, use [Percona Alerting](../alert/index.md) instead of Alertmanager.

- The **Alertmanager URL** field should contain the URL of the Alertmanager which would serve your PMM alerts.
- The **Prometheus Alerting rules** field is used to specify alerting rules in the YAML configuration format.

Fill in both fields and click the **Apply Alertmanager settings** button to proceed.

