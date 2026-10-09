#!/usr/bin/env python3
"""Annotation query cost: A (Grafana tags API), E (VictoriaMetrics), K (ClickHouse); median of 5, caches off.
Wall time through Grafana plus engine time (VM executionTimeMsec, ClickHouse query_log). usage: bench.py label out.json"""
import base64, json, re, ssl, statistics, subprocess, sys, time, urllib.parse, urllib.request
LABEL, OUT = sys.argv[1], sys.argv[2]
BASE = 'https://pmmqa-pmm-15663-bench.tp.int.percona.com/graph'
AUTH = 'Basic ' + base64.b64encode(b'admin:Bench-15663').decode()
ctx = ssl._create_unverified_context()
VM = '/opt/pmm-ai/plugins/pmm-qa/skills/chaos-docker-provisioning/scripts/chaos-vm.sh'
PROM = {'type': 'prometheus', 'uid': 'PA58DA793C7250F1B'}
CH = {'type': 'grafana-clickhouse-datasource', 'uid': 'PDEE91DDB90597936'}
ALL_NODES = [f'node-{i:04d}' for i in range(1000)]
ALL_SVCS = [f'{n}-{k}-{j}' for n in ALL_NODES for k in ('mysql', 'postgresql', 'mongodb', 'valkey') for j in (0, 1)]
RANGES = {'6h': 6 * 3600, '7d': 7 * 86400, '30d': 30 * 86400}

def http(path, body=None):
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode() if body else None, method='POST' if body else 'GET',
                                 headers={'Authorization': AUTH, 'Content-Type': 'application/json'})
    t = time.perf_counter()
    r = json.load(urllib.request.urlopen(req, context=ctx, timeout=120))
    return r, (time.perf_counter() - t) * 1000

def remote(cmd, inp=None):
    return subprocess.run([VM, 'ssh', 'PMM-15663-bench', cmd], input=inp, capture_output=True, text=True, timeout=600).stdout

def med(xs):
    return round(statistics.median(xs), 1)

def q(s):
    return "'" + s + "'"

E_SEL = {
    'all': 'environment=~".*",node_name=~".+"',
    'env': 'environment=~"env-4",node_name=~".+"',
    'node': 'environment=~".*",node_name=~"node-0000"',
    'svc': 'environment=~".*",node_name=~"node-0856",service_name=~"node-0856-mysql-1",service_type="mysql"',
}
K_COND = {
    'all': "match(environment, '^(.*)$') AND match(node_name, '^(.+)$')",
    'env': "match(environment, '^(env-4)$') AND match(node_name, '^(.+)$')",
    'node': "match(environment, '^(.*)$') AND match(node_name, '^(node-0000)$')",
    'svc': "has(service_types, 'mysql') AND match(node_name, '^(node-0856)$') AND arrayExists(x -> match(x, '^(node-0856-mysql-1)$'), service_names)",
    # the _list form: every name inlined, as IN (${service_name_list:singlequote}) would send on All
    'list-nodes': 'node_name IN (' + ','.join(map(q, ALL_NODES)) + ')',
    'list-svcs': 'hasAny(service_names, [' + ','.join(map(q, ALL_SVCS)) + '])',
}
A_TAGS = {'all-main': ['pmm_annotation'] + ALL_NODES, 'all-pr': ['pmm_annotation', '.+'], 'node': ['pmm_annotation', 'node-0000']}

res = []
now = int(time.time() * 1000)
for rng, secs in RANGES.items():
    frm = now - secs * 1000
    for scope, sel in E_SEL.items():
        expr = f'max by (annotation_id, text, node_name) (last_over_time(pmm_annotation_event{{{sel}}}[$__interval] offset -$__interval))'
        body = {'from': str(frm), 'to': str(now), 'queries': [{'refId': 'Anno', 'datasource': PROM, 'expr': expr, 'interval': '60s', 'range': True,
                                                              'intervalMs': 60000, 'maxDataPoints': 1600}]}
        walls, n, step = [], 0, None
        for _ in range(5):
            r, ms = http('/api/ds/query', body)
            walls.append(ms)
            frames = r['results']['Anno'].get('frames', [])
            n = len(frames)
            m = re.search(r'Step:\s*(\S+)', frames[0]['schema']['meta'].get('executedQueryString', '')) if frames else None
            step = m.group(1) if m else step
        # engine time: same query straight at VictoriaMetrics with the step Grafana used
        st = step or '60s'
        vexpr = expr.replace('$__interval', st)
        cmd = ('for i in 1 2 3 4 5; do docker exec pmm-server curl -s http://127.0.0.1:9090/prometheus/api/v1/query_range '
               f'--data-urlencode {json.dumps("query=" + vexpr)} -d start={frm // 1000} -d end={now // 1000} -d step={st} '
               "| grep -o '\"executionTimeMsec\":[0-9]*'; done")
        eng = [int(x.split(':')[1]) for x in remote(cmd).split()]
        res.append(dict(store='E', rng=rng, scope=scope, wall_ms=med(walls), engine_ms=med(eng) if eng else None, series=n, step=step,
                        req_bytes=len(json.dumps(body))))
        print(json.dumps(res[-1]), flush=True)
    for scope, cond in K_COND.items():
        sql = ('SELECT time, text, arrayConcat(tags, [node_name], service_names) AS tags FROM pmm.annotations '
               f'WHERE $__timeFilter(time) AND {cond} ORDER BY time')
        body = {'from': str(frm), 'to': str(now), 'queries': [{'refId': 'Anno', 'datasource': CH, 'editorType': 'sql', 'format': 1,
                                                              'queryType': 'table', 'rawSql': sql}]}
        walls, rows, err = [], None, None
        for _ in range(5):
            try:
                r, ms = http('/api/ds/query', body)
            except urllib.error.HTTPError as e:
                err = f'{e.code} {e.read().decode()[:200]}'
                break
            walls.append(ms)
            rr = r['results']['Anno']
            err = rr.get('error')
            rows = sum(len(f['data']['values'][0]) for f in rr.get('frames', []) if f['data']['values'])
        tag = f'bench-{LABEL}-{rng}-{scope}-{now}'
        csql = sql.replace('$__timeFilter(time)', f'time >= fromUnixTimestamp64Milli({frm}) AND time <= fromUnixTimestamp64Milli({now})')
        remote('cat > /root/q.sql', inp=csql)
        remote(f"for i in 1 2 3 4 5; do docker exec -i pmm-server clickhouse-client --password clickhouse --log_comment {tag} --use_query_cache 0 --format Null < /root/q.sql; done")
        eng = remote(f"docker exec pmm-server clickhouse-client --password clickhouse -q \"SYSTEM FLUSH LOGS; SELECT query_duration_ms FROM system.query_log WHERE log_comment='{tag}' AND type='QueryFinish'\"").split()
        eng = [int(x) for x in eng if x.isdigit()]
        res.append(dict(store='K', rng=rng, scope=scope, wall_ms=med(walls) if walls else None, engine_ms=med(eng) if eng else None,
                        rows=rows, error=err, sql_bytes=len(sql), req_bytes=len(json.dumps(body))))
        print(json.dumps(res[-1]), flush=True)
    for scope, tags in A_TAGS.items():
        qs = urllib.parse.urlencode([('from', frm), ('to', now), ('limit', 100), ('matchAny', 'true'), ('type', 'tags')] + [('tags', t) for t in tags])
        walls, n = [], None
        for _ in range(5):
            r, ms = http('/api/annotations?' + qs)
            walls.append(ms)
            n = len(r)
        res.append(dict(store='A', rng=rng, scope=scope, wall_ms=med(walls), rows=n, url_bytes=len(qs)))
        print(json.dumps(res[-1]), flush=True)
json.dump(res, open(OUT, 'w'), indent=1)
