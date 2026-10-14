#!/bin/bash
# PMM-15663 probe P3: LBAC with a custom label. Runs on the VM host. Needs ADMIN_PASSWORD.
set -u
B=https://127.0.0.1
A=(-ksS -u "admin:$ADMIN_PASSWORD" -H 'Content-Type: application/json')
NOW=$(($(date +%s) * 1000 - 600000))

docker exec -i pmm-server curl -sS -o /dev/null -w '%{http_code} import\n' --data-binary @- http://127.0.0.1:9090/prometheus/api/v1/import <<EOF
{"metric":{"__name__":"pmm_annotation_event","annotation_id":"a1","text":"team a","node_name":"n1","environment":"env-1","team":"a"},"values":[$NOW],"timestamps":[$NOW]}
{"metric":{"__name__":"pmm_annotation_event","annotation_id":"b1","text":"team b","node_name":"n2","environment":"env-1","team":"b"},"values":[$NOW],"timestamps":[$NOW]}
{"metric":{"__name__":"pmm_annotation_event","annotation_id":"x1","text":"no team label","node_name":"n1","environment":"env-1"},"values":[$NOW],"timestamps":[$NOW]}
EOF

curl "${A[@]}" -X PUT $B/v1/server/settings -d '{"enable_access_control": true}' -o /dev/null -w '%{http_code} enable access control\n'
DS=$(curl "${A[@]}" $B/graph/api/datasources | python3 -c 'import json,sys; print([d["uid"] for d in json.load(sys.stdin) if d["type"]=="prometheus"][0])')
echo "prometheus datasource uid=$DS"

mkrole() { curl "${A[@]}" -X POST $B/v1/accesscontrol/roles -d "{\"title\":\"$1\",\"filter\":\"$2\"}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["role_id"])'; }
mkuser() { curl "${A[@]}" -X POST $B/graph/api/admin/users -d "{\"name\":\"$1\",\"login\":\"$1\",\"password\":\"$2\"}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])'; }

PW=$(openssl rand -hex 12)
R_TEAM=$(mkrole team-a '{team=\"a\"}')
R_ENV=$(mkrole env-1 '{environment=\"env-1\"}')
U_TEAM=$(mkuser viewer-team-a "$PW")
U_ENV=$(mkuser viewer-env-1 "$PW")
curl "${A[@]}" -X POST $B/v1/accesscontrol/roles:assign -d "{\"role_ids\":[$R_TEAM],\"user_id\":$U_TEAM}" -o /dev/null -w '%{http_code} assign team-a\n'
curl "${A[@]}" -X POST $B/v1/accesscontrol/roles:assign -d "{\"role_ids\":[$R_ENV],\"user_id\":$U_ENV}" -o /dev/null -w '%{http_code} assign env-1\n'
sleep 70

ask() {
  local user=$1 expr=$2
  curl -ksS -u "$user:$PW" -H 'Content-Type: application/json' $B/graph/api/ds/query -d @- <<EOF | python3 -c 'import json,sys; d=json.load(sys.stdin); fr=d.get("results",{}).get("A",{}).get("frames",[]); print(sorted({f["schema"]["fields"][1].get("labels",{}).get("annotation_id","?") for f in fr if len(f["schema"]["fields"])>1}) or d)'
{"queries":[{"refId":"A","datasource":{"type":"prometheus","uid":"$DS"},"expr":"$expr","instant":true,"range":false}],"from":"now-1h","to":"now"}
EOF
}
E='last_over_time(pmm_annotation_event{annotation_id=~\"a1|b1|x1\"}[1h])'
echo -n "admin:          "; curl "${A[@]}" $B/graph/api/ds/query -d "{\"queries\":[{\"refId\":\"A\",\"datasource\":{\"type\":\"prometheus\",\"uid\":\"$DS\"},\"expr\":\"$E\",\"instant\":true,\"range\":false}],\"from\":\"now-1h\",\"to\":\"now\"}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(sorted({f["schema"]["fields"][1].get("labels",{}).get("annotation_id","?") for f in d["results"]["A"]["frames"] if len(f["schema"]["fields"])>1}))'
echo -n "viewer-team-a:  "; ask viewer-team-a "$E"
echo -n "viewer-env-1:   "; ask viewer-env-1 "$E"
echo "PROBE_P3_DONE"
