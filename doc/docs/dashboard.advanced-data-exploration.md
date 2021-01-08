# *Advanced Data Exploration* Dashboard

The *Advanced Data Exploration* dashboard provides detailed information about the progress of a single Prometheus metric across one or more hosts.

!!! note "Added NUMA related metrics"
    Version added: 1.13.0

    This dashboard supports metrics related to NUMA. The names of all these metrics start with *node_memory_numa*.

    ![](_images/metrics-monitor.advanced-data-exploration.node-memory-numa.png)

## View actual metric values (Gauge)

In this section, the values of the selected metric may increase or decrease over time (similar to temperature or memory usage).

## View actual metric values (Counters)

In this section, the values of the selected metric are accumulated over time (useful to count the number of served requests, for example).

## View actual metric values (Counters)

This section presents the values of the selected metric in the tabular form.
