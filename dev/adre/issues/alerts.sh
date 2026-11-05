#!/usr/bin/env bash
# Creates PMM alert rules from the built-in pmm_* templates for the Assistant test environment.
# Run it on the PMM host: ./alerts.sh [create|list|delete]
# PMM_URL (default https://localhost) and the admin password in PMM_ADMIN_PASSWORD or /root/admin_password.
set -euo pipefail

PMM_URL=${PMM_URL:-https://localhost}
FOLDER_TITLE=${FOLDER_TITLE:-Assistant tests}
GROUP=${GROUP:-assistant-tests}
# Every container node reports the host CPU and memory, so node rules watch one node only.
HOST_NODE=${HOST_NODE:-pmm-server}

if [[ -z "${PMM_ADMIN_PASSWORD:-}" ]]; then
  PMM_ADMIN_PASSWORD=$(cat /root/admin_password)
fi
export PMM_URL FOLDER_TITLE GROUP HOST_NODE PMM_ADMIN_PASSWORD

python3 - "${1:-create}" <<'EOF'
import base64, json, os, ssl, sys, urllib.request, urllib.error

url, folder_title, group, host_node = (os.environ[k] for k in ("PMM_URL", "FOLDER_TITLE", "GROUP", "HOST_NODE"))
auth = "Basic " + base64.b64encode(("admin:" + os.environ["PMM_ADMIN_PASSWORD"]).encode()).decode()
ctx = ssl._create_unverified_context()

def call(method, path, body=None):
    req = urllib.request.Request(url + path, method=method, data=json.dumps(body).encode() if body is not None else None,
                                 headers={"Authorization": auth, "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, context=ctx) as r:
            raw = r.read()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raise SystemExit(f"{method} {path}: HTTP {e.code}: {e.read().decode()[:400]}")

W, C = "SEVERITY_WARNING", "SEVERITY_CRITICAL"
# (template, rule name, severity, for, params, filters)
RULES = [
    ("pmm_agent_down", "PMM agent down", C, "60s", {}, []),
    ("pmm_mysql_down", "MySQL down", C, "60s", {}, []),
    ("pmm_mysql_restarted", "MySQL restarted", W, "60s", {"threshold": 300}, []),
    ("pmm_mysql_too_many_connections", "MySQL connections near max_connections", W, "60s", {"threshold": 80}, []),
    ("pmm_mysql_replication_sql_running", "MySQL replication SQL thread stopped", C, "60s", {}, []),
    ("pmm_mysql_replication_io_running", "MySQL replication IO thread stopped", C, "60s", {}, []),
    ("pmm_postgresql_down", "PostgreSQL down", C, "60s", {}, []),
    ("pmm_postgresql_restarted", "PostgreSQL restarted", W, "60s", {"threshold": 300}, []),
    ("pmm_postgresql_too_many_connections", "PostgreSQL connections near max_connections", W, "60s", {"threshold": 80}, []),
    ("pmm_postgresql_high_number_dead_tuples", "PostgreSQL high number of dead tuples", W, "120s", {}, []),
    ("pmm_postgresql_high_transaction_rollbacks", "PostgreSQL high transaction rollbacks", W, "60s", {}, []),
    ("pmm_mongodb_down", "MongoDB down", C, "60s", {}, []),
    ("pmm_mongodb_restarted", "MongoDB restarted", W, "60s", {"threshold": 300}, []),
    ("pmm_mongodb_replset_primary_changed", "MongoDB primary changed", W, "60s", {}, []),
    ("pmm_mongodb_replication_lag", "MongoDB replication lag", W, "60s", {"threshold": 60}, []),
    # MongoDB 8.0 throughput probing keeps about 16 tickets at idle, under the template default of 50.
    ("pmm_mongodb_read_tickets", "MongoDB read tickets exhausted", W, "60s", {"threshold": 4}, []),
    ("pmm_mongodb_write_tickets", "MongoDB write tickets exhausted", W, "60s", {"threshold": 4}, []),
    ("pmm_node_high_cpu_load", "Host CPU high", W, "120s", {"threshold": 70}, [("node_name", host_node)]),
    ("pmm_node_low_free_memory", "Host memory low", W, "120s", {"threshold": 20}, [("node_name", host_node)]),
]

def folder_uid(create):
    for f in call("GET", "/graph/api/folders"):
        if f["title"] == folder_title:
            return f["uid"]
    if not create:
        return None
    return call("POST", "/graph/api/folders", {"title": folder_title})["uid"]

def existing(uid):
    return {r["title"]: r for r in call("GET", "/graph/api/v1/provisioning/alert-rules") if r.get("folderUID") == uid}

cmd = sys.argv[1]
if cmd == "create":
    uid = folder_uid(True)
    have = existing(uid)
    for tpl, name, sev, dur, params, filters in RULES:
        if name in have:
            print("exists ", name)
            continue
        call("POST", "/v1/alerting/rules", {
            "template_name": tpl, "name": name, "group": group, "folder_uid": uid,
            "params": [{"name": k, "type": "PARAM_TYPE_FLOAT", "float": v} for k, v in params.items()],
            "for": dur, "severity": sev, "interval": "60s",
            "filters": [{"type": "FILTER_TYPE_MATCH", "label": l, "regexp": r} for l, r in filters],
        })
        print("created", name)
elif cmd == "list":
    uid = folder_uid(False)
    for name in sorted(existing(uid)) if uid else []:
        print(name)
    firing = call("GET", "/graph/api/alertmanager/grafana/api/v2/alerts?active=true")
    for a in firing:
        if a["labels"].get("grafana_folder") == folder_title:
            l = a["labels"]
            print("FIRING", l.get("alertname"), l.get("service_name") or l.get("node_name", ""), a["startsAt"])
elif cmd == "delete":
    uid = folder_uid(False)
    for name, r in (existing(uid) if uid else {}).items():
        call("DELETE", "/graph/api/v1/provisioning/alert-rules/" + r["uid"])
        print("deleted", name)
else:
    raise SystemExit("usage: alerts.sh [create|list|delete]")
EOF
