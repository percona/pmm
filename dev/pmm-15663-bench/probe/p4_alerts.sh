#!/bin/bash
# PMM-15663 probe P4: Grafana "prometheus" alert state history into VictoriaMetrics. Runs on the VM host. Needs ADMIN_PASSWORD.
set -u
B=https://127.0.0.1
A=(-ksS -u "admin:$ADMIN_PASSWORD" -H 'Content-Type: application/json')

echo "== direct VM datasource (vmproxy only allows reads)"
curl "${A[@]}" -X POST $B/graph/api/datasources -d '{"name":"VM-direct-probe","uid":"vmdirect","type":"prometheus","access":"proxy","url":"http://127.0.0.1:9090/prometheus","jsonData":{"prometheusType":"Prometheus"}}' -o /dev/null -w '%{http_code} datasource\n'

echo "== grafana.ini state history"
docker exec pmm-server bash -c 'cat >> /etc/grafana/grafana.ini <<EOF

[unified_alerting.state_history]
enabled = true
backend = multiple
primary = annotations
secondaries = prometheus
prometheus_target_datasource_uid = vmdirect
EOF
supervisorctl restart grafana'
until curl -ks $B/graph/api/health | grep -q '"database": *"ok"'; do sleep 3; done
echo "grafana back"

echo "== feed probe_value for svc-a (loop in container)"
docker exec -d pmm-server bash -c 'while [ ! -f /tmp/stop-feed ]; do t=$(($(date +%s)*1000)); printf "{\"metric\":{\"__name__\":\"probe_value\",\"service_name\":\"svc-a\",\"node_name\":\"n1\",\"environment\":\"env-1\",\"team\":\"a\"},\"values\":[1],\"timestamps\":[%s]}\n" $t | curl -s -o /dev/null --data-binary @- http://127.0.0.1:9090/prometheus/api/v1/import; sleep 10; done'

MDS=$(curl "${A[@]}" $B/graph/api/datasources | python3 -c 'import json,sys; print([d["uid"] for d in json.load(sys.stdin) if d["name"]=="Metrics"][0])')
FUID=$(curl "${A[@]}" -X POST $B/graph/api/folders -d '{"title":"probe-15663","uid":"probe15663"}' | python3 -c 'import json,sys; print(json.load(sys.stdin).get("uid","probe15663"))')
echo "metrics ds=$MDS folder=$FUID"
curl "${A[@]}" -X PUT $B/graph/api/v1/provisioning/folder/$FUID/rule-groups/probe -d @- -o /dev/null -w '%{http_code} rule group\n' <<EOF
{"title":"probe","folderUid":"$FUID","interval":10,"rules":[{"uid":"probe15663rule","title":"probe-alert","ruleGroup":"probe","folderUID":"$FUID","orgID":1,"condition":"C","for":"0s","noDataState":"OK","execErrState":"OK",
 "data":[{"refId":"A","relativeTimeRange":{"from":600,"to":0},"datasourceUid":"$MDS","model":{"refId":"A","expr":"probe_value","instant":true}},
         {"refId":"C","datasourceUid":"__expr__","model":{"refId":"C","type":"threshold","expression":"A","conditions":[{"evaluator":{"type":"gt","params":[0]}}]}}]}]}
EOF

VMQ() { docker exec pmm-server curl -sS http://127.0.0.1:9090/prometheus/api/v1/query --data-urlencode "query=$1"; }
VMR() { docker exec pmm-server curl -sS http://127.0.0.1:9090/prometheus/api/v1/query_range --data-urlencode "query=$1" --data-urlencode "start=$2" --data-urlencode "end=$(date +%s)" --data-urlencode "step=10"; }
T0=$(date +%s)
sleep 150
echo "== GRAFANA_ALERTS while firing (labels)"
VMQ 'last_over_time(GRAFANA_ALERTS[10m])' | python3 -c 'import json,sys; [print(x["metric"], x["value"]) for x in json.load(sys.stdin)["data"]["result"]]'
echo "== stop feed -> alert should resolve"
docker exec pmm-server touch /tmp/stop-feed
T1=$(date +%s)
sleep 240
echo "== GRAFANA_ALERTS samples over the run (step 10s): count of non-empty points per series"
VMR 'GRAFANA_ALERTS' $T0 | python3 -c '
import json,sys
for x in json.load(sys.stdin)["data"]["result"]:
    ts=[int(v[0]) for v in x["values"]]
    print(x["metric"].get("grafana_alertstate"), x["metric"].get("alertstate"), "points", len(ts), "first", ts[0] if ts else None, "last", ts[-1] if ts else None)'
echo "feed stopped at $T1"
echo "== Grafana annotations backend still has the transitions"
curl "${A[@]}" "$B/graph/api/annotations?type=alert&limit=20" | python3 -c 'import json,sys; [print(a.get("newState"), a.get("prevState"), a.get("time"), a.get("alertName","")) for a in json.load(sys.stdin)]'
echo "PROBE_P4_DONE"
