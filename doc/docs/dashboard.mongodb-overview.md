# MongoDB Overview

This dashboard provides basic information about MongoDB instances.

## Command Operations

Shows how many times a command is executed per second on average during the
selected interval.

Look for peaks and drops and correlate them with other graphs.

**View all metrics of** MongoDB Overview

## Connections

Keep in mind the hard limit on the maximum number of connections set by your
distribution.

Anything over 5,000 should be a concern, because the application may not close
connections correctly.

**View all metrics of** MongoDB Overview

## Cursors

Helps identify why connections are increasing.  Shows active cursors compared to
cursors being automatically killed after 10 minutes due to an application not
closing the connection.

**View all metrics of** MongoDB Overview

## Document Operations

When used in combination with **Command Operations**, this graph can help
identify *write aplification*.  For example, when one `insert` or `update`
command actually inserts or updates hundreds, thousands, or even millions of
documents.

**View all metrics of** MongoDB Overview

## Queued Operations

Any number of queued operations for long periods of time is an indication of
possible issues.  Find the cause and fix it before requests get stuck in the
queue.

**View all metrics of** MongoDB Overview

## getLastError Write Time, getLastError Write Operations

This is useful for write-heavy workloads to understand how long it takes to
verify writes and how many concurrent writes are occurring.

**View all metrics of** MongoDB Overview

## Asserts

Asserts are not important by themselves, but you can correlate spikes with other
graphs.

**View all metrics of** MongoDB Overview

## Memory Faults

Memory faults indicate that requests are processed from disk either because an
index is missing or there is not enough memory for the data set.  Consider
increasing memory or sharding out.

**View all metrics of** MongoDB Overview

<!-- -*- mode: rst -*- -->
<!-- Tips (tip) -->
<!-- Abbreviations (abbr) -->
<!-- Docker commands (docker) -->
<!-- Graphical interface elements (gui) -->
<!-- Options and parameters (opt) -->
<!-- pmm-admin commands (pmm-admin) -->
<!-- SQL commands (sql) -->
<!-- PMM Dashboards (dbd) -->
<!-- * Text labels -->
<!-- Special headings (h) -->
<!-- Status labels (status) -->
