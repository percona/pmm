# PMM-15663 probe results

Probe run on 14 Oct 2026 on the throwaway CHAOS VM `PMM-15663-probe` (8 vCPU / 16 GB, torn down after the run):
`perconalab/pmm-server:3-dev-latest`, single node, `percona-grafana-12.4.5-117.2610140456.3650a2f`,
`victoria-metrics-20260929-074548-pmm-6401-v1.152.0`.

The raw logs were lost when the session moved to another runner. The output below is transcribed from what the
session captured while the probes ran. Lines it did not capture are marked as such, not reconstructed.

## P1: label limits (`p12_labels_delete.sh`, inside the container)

Import responses: only `204 import len=20000` was captured for the length series; `special` and all four label-count
imports returned `204`.

Query 70 s later (series found, text length):

```
len-1000: 1 [('text', 1000)]
len-4000: 1 [('text', 4000)]
len-4096: 1 [('text', 4096)]
len-4097: 0 []
len-5000: 0 []
len-16384: 0 []
len-20000: 0 []
special: ['line1\nline2 "quoted" \'single\' <b>html</b> {{.x}} ünïcødé 🚀 back\\slash']
labels-30: 1 [30]
labels-40: 1 [40]
labels-41: 0 []
labels-60: 0 []
```

VictoriaMetrics log (lines cut in the capture):

```
warn lib/timeserieslimits/timeseries_limits.go:104 ignoring series with text="xxx...
warn lib/timeserieslimits/timeseries_limits.go:91  ignoring series with 41 labels for {__name__="pmm_annotation_event",annotation_id="labels-41",...
```

Counters: `vm_rows_ignored_total{reason="too_long_label_value"} 4`, `too_long_label_name 0`. The grep pattern did not
match the too-many-labels counter, so its value was not captured.

Result: a label value over 4,096 bytes or a series with more than 40 labels (`__name__` included) drops the whole
series, and the import still returns 204. Newlines, quotes, HTML, template braces and non-ASCII text are stored as is.

## P2: delete and re-import (`p12_labels_delete.sh`)

```
204 import v1
before delete: 1 [('text', 2)]
 delete=204
right after delete: 0 []
204 import v2 (new text)
204 import v1 again (same labels as deleted)
t+0s after reimport: v1=0 []   v2=0 []
t+30s after reimport: v1=1     v2=1
t+90s after reimport: v1=1     v2=1
t+210s after reimport: v1=1    v2=1
t+511s after reimport: v1=1    v2=1
```

Result: `delete_series` hides the series at once, and both a changed and an identical label set can be written again
and show within 30 s. Single node only; HA (v1.128 cluster) was not tested.

## P3: LBAC on a custom label (`p3_lbac.sh`, on the VM host)

Series: `a1` with `team="a"`, `b1` with `team="b"`, `x1` without `team`; all `environment="env-1"`.

```
204 import
200 enable access control
prometheus datasource uid=PA58DA793C7250F1B
200 assign team-a
200 assign env-1
admin:          ['a1', 'b1', 'x1']
viewer-team-a:  ['a1']
viewer-env-1:   ['a1', 'b1', 'x1']
```

Result: a role filtered on a custom label sees only series carrying that label. An annotation series must carry every
label a role may filter on, custom labels included, or restricted users won't see it.

## P4: Grafana alert history as metrics (`p4_alerts.sh`, on the VM host)

```
200 datasource
grafana: stopped
grafana: started
grafana back
metrics ds=PA58DA793C7250F1B folder=probe15663
200 rule group
== GRAFANA_ALERTS while firing (labels)
{'__name__': 'GRAFANA_ALERTS', 'alertname': 'probe-alert', 'alertstate': 'firing', 'environment': 'env-1', 'grafana_alertstate': 'alerting', 'grafana_folder': 'probe-15663', 'grafana_rule_uid': 'probe15663rule', 'node_name': 'n1', 'service_name': 'svc-a', 'team': 'a'} [1792009967, '1']
alerting firing points 38 first 1792009837 last 1792010207
feed stopped at 1792009967
Alerting Normal 1792009830000 probe-alert
```

The alert was still firing when the script ended, because the test metric stays visible for about 5 minutes after the
last sample. Follow-up queries, step 10 s:

```
at 1792010226: alerting points 40 first 1792009830 last 1792010220 gaps []
at 1792010416: alerting points 45 first 1792009830 last 1792010270   (probe_value gone)
```

Result: with `backend = multiple`, `primary = annotations`, `secondaries = prometheus`, Grafana writes `GRAFANA_ALERTS`
with every label of the alert instance at each evaluation, as one continuous series from firing to resolve. Per
Grafana's code (`packages/grafana-prometheus/src/annotations.ts:98-127`), Prometheus annotations draw such a series as
a region; the probe did not render it on a dashboard. It needs a Prometheus datasource that points straight at
VictoriaMetrics: the provisioned "Metrics" datasource goes through vmproxy, which only allows reads.

## P5: node_name_list matcher (`results/bench-node-list.json`)

Run on 15 Oct 2026 on the throwaway CHAOS VM `PMM-15663-bench2` (16 vCPU / 32 GB, torn down after the run):
`perconalab/pmm-server:3-dev-latest` (started without `PMM_DEBUG`), `percona-grafana-12.4.5-117.2610140456.3650a2f`,
`percona-victoriametrics-1.152.0` with `-search.disableCache=true`, `-retentionPeriod=30d`, `-search.maxQueryLen=1MB`.

Setup: stress fleet (`bash load.sh`, then `FILLER_ONLY=1 bash load.sh --stress`; all 16 shard imports returned 204) and
5,000 annotations loaded as `pmm_annotation_event` series only (`python3 ann_load.py 0 5000 E`). `/api/v1/status/tsdb`
reported `totalSeries` 3,070,727 when the benchmark ran; 5,981 of them are annotation series, 1,504 of them with no
`service_name` (node annotations). The 1,000 node names are the `node-NNNN` values of `label/node_name/values` for `up`
(that call also returns `pmm-server`, which is not part of the synthetic fleet and was left out).

Query: 30 days, Metrics datasource through Grafana `/api/ds/query`, median of 5 per form, no warm-up. Engine time is the
same expression sent straight to VictoriaMetrics `query_range` with the 30m0s step Grafana used (`executionTimeMsec`, median of 5).

```
max by (annotation_id, text, tags) (last_over_time(pmm_annotation_event{service_name="",node_name=~"<matcher>"}[$__interval] offset -$__interval))
```

| `node_name` matcher | Grafana median | VictoriaMetrics median | series returned | query size |
|---|---|---|---|---|
| `=~"node-0000\|node-0001\|...\|node-0999"` (1,000 names) | 168.6 ms (253.8, 171.5, 168.6, 156.3, 164.8) | 30 ms (31, 30, 30, 30, 30) | 1,504 | 10,136 bytes |
| `=~".+"` | 158.5 ms (202.3, 145.1, 187.0, 158.5, 153.3) | 29 ms (61, 39, 28, 29, 27) | 1,504 | 139 bytes |

Both forms returned the same 1,504 annotation ids, which are all the node annotations in the loaded set (none missing,
none extra), with no Grafana or VictoriaMetrics error.

Result: GO. The list form takes 168.6 ms through Grafana against the 1 s threshold, and costs about 10 ms more through
Grafana than `.+`. Not measured: other ranges, concurrent dashboard load, a browser load of a real dashboard, HA.
