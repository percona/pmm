#!/bin/bash
# PMM-15663 probe P1 (label limits) and P2 (delete + re-import). Runs inside the pmm-server container.
set -u
VM=http://127.0.0.1:9090/prometheus
NOW=$(($(date +%s) * 1000 - 600000))

imp() { curl -sS -o /dev/null -w '%{http_code}' --data-binary @- "$VM/api/v1/import"; }
q() { curl -sS "$VM/api/v1/query" --data-urlencode "query=$1" --data-urlencode "time=$(date +%s)"; }
qlen() { q "$1" | python3 -c 'import json,sys; r=json.load(sys.stdin)["data"]["result"]; print(len(r), [ (k, len(v)) for x in r for k,v in x["metric"].items() if k=="text"])'; }

echo "== P1 label value length"
for n in 1000 4000 4096 4097 5000 16384 20000; do
  text=$(python3 -c "print('x'*$n)")
  printf '{"metric":{"__name__":"pmm_annotation_event","annotation_id":"len-%s","text":"%s","node_name":"n1","environment":"env-1"},"values":[%s],"timestamps":[%s]}\n' "$n" "$text" "$NOW" "$NOW" | imp; echo " import len=$n"
done

echo "== P1 special characters"
python3 - "$NOW" <<'PY' | curl -sS -o /dev/null -w '%{http_code} special\n' --data-binary @- http://127.0.0.1:9090/prometheus/api/v1/import
import json, sys
now = int(sys.argv[1])
text = 'line1\nline2 "quoted" \'single\' <b>html</b> {{.x}} ünïcødé 🚀 back\\slash'
print(json.dumps({"metric": {"__name__": "pmm_annotation_event", "annotation_id": "special", "text": text, "node_name": "n1", "environment": "env-1"}, "values": [now], "timestamps": [now]}))
PY

echo "== P1 label count"
for n in 30 40 41 60; do
  python3 - "$NOW" "$n" <<'PY' | curl -sS -o /dev/null -w "%{http_code} labels\n" --data-binary @- http://127.0.0.1:9090/prometheus/api/v1/import
import json, sys
now, n = int(sys.argv[1]), int(sys.argv[2])
m = {"__name__": "pmm_annotation_event", "annotation_id": f"labels-{n}"}
for i in range(n - 2):
    m[f"l{i:02d}"] = "v"
print(json.dumps({"metric": m, "values": [now], "timestamps": [now]}))
PY
done

sleep 70
echo "== P1 results (len): count, text lengths"
for n in 1000 4000 4096 4097 5000 16384 20000; do echo -n "len-$n: "; qlen "last_over_time(pmm_annotation_event{annotation_id=\"len-$n\"}[1h])"; done
echo -n "special: "; q 'last_over_time(pmm_annotation_event{annotation_id="special"}[1h])' | python3 -c 'import json,sys; r=json.load(sys.stdin)["data"]["result"]; print([x["metric"].get("text") for x in r])'
for n in 30 40 41 60; do echo -n "labels-$n: "; q "last_over_time(pmm_annotation_event{annotation_id=\"labels-$n\"}[1h])" | python3 -c 'import json,sys; r=json.load(sys.stdin)["data"]["result"]; print(len(r), [len(x["metric"]) for x in r])'; done
echo "== VM log lines about labels"
grep -iE "label|truncat|too long|exceed" /srv/logs/victoriametrics.log | tail -8
echo "== VM counters"
curl -sS http://127.0.0.1:9090/prometheus/metrics | grep -iE "too_long|dropped|labels_truncated|max_labels|label_value" | grep -v '^#' | head -12

echo "== P2 delete + re-import"
T=$NOW
line() { printf '{"metric":{"__name__":"pmm_annotation_event","annotation_id":"edit-1","text":"%s","node_name":"n1","environment":"env-1"},"values":[%s],"timestamps":[%s]}\n' "$1" "$T" "$T"; }
line v1 | imp; echo " import v1"
sleep 70
echo -n "before delete: "; qlen 'last_over_time(pmm_annotation_event{annotation_id="edit-1"}[1h])'
curl -sS -w ' delete=%{http_code}\n' "$VM/api/v1/admin/tsdb/delete_series" --data-urlencode 'match[]=pmm_annotation_event{annotation_id="edit-1"}'
echo -n "right after delete: "; qlen 'last_over_time(pmm_annotation_event{annotation_id="edit-1"}[1h])'
line v2 | imp; echo " import v2 (new text)"
line v1 | imp; echo " import v1 again (same labels as deleted)"
START=$(date +%s)
for s in 0 30 60 120 300; do
  sleep $s
  echo -n "t+$(($(date +%s) - START))s after reimport: v1="; qlen 'last_over_time(pmm_annotation_event{annotation_id="edit-1",text="v1"}[1h])'
  echo -n "                       v2="; qlen 'last_over_time(pmm_annotation_event{annotation_id="edit-1",text="v2"}[1h])'
done
echo "PROBE_P12_DONE"
