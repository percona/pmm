# Advanced Data Exploration

The *Advanced Data Exploration* dashboard provides detailed information about
the progress of a single Prometheus metric across one or more hosts.

!!! note "NUMA-related metrics"

    This dashboard supports metrics related to NUMA. The names of all these metrics start with `node_memory_numa`.

    ![image](/_images/metrics-monitor.advanced-data-exploration.node-memory-numa.png)

## View actual metric values (Gauge)

In this section, the values of the selected metric may increase or decrease over
time (similar to temperature or memory usage).

## View actual metric values (Counters)

In this section, the values of the selected metric are accummulated over time
(useful to count the number of served requests, for example).

## Metric Data Table

This section presents the values of the selected metric in the tabular form.

!!! seealso "See also"

    [Prometheus: Metric types](https://prometheus.io/docs/concepts/metric_types/)
