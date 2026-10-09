#!/usr/bin/env python3
"""Load annotations [a:b) of ann-5000.json into Grafana (A), VictoriaMetrics (E) and ClickHouse (K). Runs on the VM host."""
import json, ssl, subprocess, sys, urllib.request, base64
a, b = int(sys.argv[1]), int(sys.argv[2])
parts = sys.argv[3] if len(sys.argv) > 3 else 'AEK'
import datetime
anns = json.load(open('/root/ann-5000.json'))[a:b]
ctx = ssl._create_unverified_context()
AUTH = 'Basic ' + base64.b64encode(b'admin:Bench-15663').decode()

def grafana(path, body):
    req = urllib.request.Request('https://127.0.0.1/graph' + path, data=json.dumps(body).encode(), method='POST',
                                 headers={'Authorization': AUTH, 'Content-Type': 'application/json'})
    return json.load(urllib.request.urlopen(req, context=ctx))

# A: same tags pmm-managed AddAnnotation produces (scoped ones carry no pmm_annotation tag)
for x in (anns if 'A' in parts else []):
    names = [s['service_name'] for s in x['services']]
    tags = x['user_tags'] + (names if names else [x['node_name']])
    grafana('/api/annotations', {'time': x['time'], 'text': x['text'], 'tags': tags})

# E: one series per (annotation, service), value = annotation time in ms
lines = []
for x in (anns if 'E' in parts else []):
    base = {'__name__': 'pmm_annotation_event', 'annotation_id': x['id'], 'text': x['text'], 'tags': ','.join(x['user_tags']),
            'node_name': x['node_name'], 'environment': x['environment'], 'cluster': x['cluster']}
    for s in x['services'] or [{}]:
        lines.append(json.dumps({'metric': dict(base, **s), 'values': [x['time']], 'timestamps': [x['time']]}))
if lines:
  subprocess.run(['docker', 'exec', '-i', 'pmm-server', 'curl', '-sf', '--data-binary', '@-',
                'http://127.0.0.1:9090/prometheus/api/v1/import'], input='\n'.join(lines).encode(), check=True)

# K: one row per annotation
if 'K' not in parts:
    sys.exit(0)
ch = ['docker', 'exec', '-i', 'pmm-server', 'clickhouse-client', '--password', 'clickhouse']
subprocess.run(ch + ['-q', """CREATE TABLE IF NOT EXISTS pmm.annotations (
  time DateTime64(3), annotation_id String, text String, tags Array(String), node_name String, environment String,
  cluster String, service_names Array(String), service_types Array(String), replication_sets Array(String)
) ENGINE = MergeTree ORDER BY time"""], check=True)
rows = '\n'.join(json.dumps({'time': datetime.datetime.fromtimestamp(x['time'] / 1000, datetime.UTC).strftime('%Y-%m-%d %H:%M:%S.%f')[:-3], 'annotation_id': x['id'], 'text': x['text'], 'tags': x['user_tags'],
                             'node_name': x['node_name'], 'environment': x['environment'], 'cluster': x['cluster'],
                             'service_names': [s['service_name'] for s in x['services']],
                             'service_types': [s['service_type'] for s in x['services']],
                             'replication_sets': [s['replication_set'] for s in x['services']]}) for x in anns)
subprocess.run(ch + ['-q', 'INSERT INTO pmm.annotations FORMAT JSONEachRow'], input=rows.encode(), check=True)
print(f'loaded {len(anns)} annotations, {len(lines)} E series')
