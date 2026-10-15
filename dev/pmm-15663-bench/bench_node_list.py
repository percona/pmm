#!/usr/bin/env python3
"""PMM-15663 task 1: node_name=~"<1000 names joined with |>" vs node_name=~".+" on pmm_annotation_event, 30 days.
Median of 5 through Grafana /api/ds/query (Metrics datasource) plus engine time (VictoriaMetrics executionTimeMsec).
usage: bench_node_list.py RUN OUT.json   (runs on the runner host; adapted from bench.py)"""
import base64, json, re, ssl, statistics, subprocess, sys, time, urllib.error, urllib.request

RUN, OUT = sys.argv[1], sys.argv[2]
VMSH = '/opt/pmm-ai/plugins/pmm-qa/skills/chaos-docker-provisioning/scripts/chaos-vm.sh'
ctx = ssl._create_unverified_context()


def remote(cmd, inp=None, timeout=900):
    p = subprocess.run([VMSH, 'ssh', RUN, cmd], input=inp, capture_output=True, text=True, timeout=timeout)
    if p.returncode:
        raise RuntimeError(f'remote failed rc={p.returncode}: {cmd[:120]}\n{p.stderr[-500:]}')
    return p.stdout


HOST = subprocess.run([VMSH, 'info', RUN], capture_output=True, text=True).stdout
HOST = re.search(r'hostname:\s*(\S+)', HOST).group(1)
BASE = f'https://{HOST}/graph'
PW = remote('cat /root/admin_password').strip()
AUTH = 'Basic ' + base64.b64encode(f'admin:{PW}'.encode()).decode()


def http(path, body=None):
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode() if body else None, method='POST' if body else 'GET',
                                 headers={'Authorization': AUTH, 'Content-Type': 'application/json'})
    t = time.perf_counter()
    try:
        r = json.load(urllib.request.urlopen(req, context=ctx, timeout=300))
    except urllib.error.HTTPError as e:
        return {'http_error': e.code, 'body': e.read().decode()[:500]}, (time.perf_counter() - t) * 1000
    return r, (time.perf_counter() - t) * 1000


def med(xs):
    return round(statistics.median(xs), 1)


# 1. the node names, from VictoriaMetrics (synthetic fleet = node-NNNN; the PMM server's own node is not part of the fleet)
raw = remote("docker exec pmm-server curl -s -G 'http://127.0.0.1:9090/prometheus/api/v1/label/node_name/values' --data-urlencode 'match[]=up'")
all_names = json.loads(raw)['data']
NODES = sorted(n for n in all_names if re.fullmatch(r'node-\d{4}', n))
print(f'node_name label values from VictoriaMetrics: {len(all_names)}, fleet nodes used: {len(NODES)}', flush=True)
assert len(NODES) == 1000, len(NODES)
other = [n for n in all_names if n not in NODES]

# 2. datasource uid
dss, _ = http('/api/datasources')
prom = [d for d in dss if d['type'] == 'prometheus' and d['name'] == 'Metrics']
assert prom, [(d['name'], d['type']) for d in dss]
PROM = {'type': 'prometheus', 'uid': prom[0]['uid']}
print('metrics datasource', PROM, flush=True)

# 3. expected annotations (node-scoped = no service_name) in range, from the file that was loaded
anns = json.loads(remote('cat /root/ann-5000.json'))
node_ann_ids = {a['id'] for a in anns if not a['services']}

LIST = '|'.join(NODES)
FORMS = {
    'list': f'max by (annotation_id, text, tags) (last_over_time(pmm_annotation_event{{service_name="",node_name=~"{LIST}"}}[$__interval] offset -$__interval))',
    'dotplus': 'max by (annotation_id, text, tags) (last_over_time(pmm_annotation_event{service_name="",node_name=~".+"}[$__interval] offset -$__interval))',
}

now = int(time.time() * 1000)
frm = now - 30 * 86400 * 1000
in_range = {a['id'] for a in anns if not a['services'] and frm <= a['time'] <= now}
res = []
ids_by_form = {}
for form, expr in FORMS.items():
    body = {'from': str(frm), 'to': str(now), 'queries': [{'refId': 'Anno', 'datasource': PROM, 'expr': expr, 'interval': '60s', 'range': True,
                                                          'intervalMs': 60000, 'maxDataPoints': 1600}]}
    walls, step, err, ids = [], None, None, set()
    for _ in range(5):
        r, ms = http('/api/ds/query', body)
        walls.append(ms)
        if 'http_error' in r:
            err = f"HTTP {r['http_error']} {r['body']}"
            continue
        rr = r['results']['Anno']
        err = rr.get('error') or err
        frames = rr.get('frames', [])
        ids = set()
        for f in frames:
            for fld in f['schema']['fields']:
                if fld.get('labels', {}).get('annotation_id'):
                    ids.add(fld['labels']['annotation_id'])
        m = re.search(r'Step:\s*(\S+)', frames[0]['schema']['meta'].get('executedQueryString', '')) if frames else None
        step = m.group(1) if m else step
    ids_by_form[form] = ids
    # engine time: same expression straight at VictoriaMetrics with the step Grafana used
    st = step or '30m0s'
    vexpr = expr.replace('$__interval', st)
    remote('cat > /root/q.txt', inp=vexpr)
    remote('docker cp /root/q.txt pmm-server:/tmp/q.txt')
    cmd = ('for i in 1 2 3 4 5; do docker exec pmm-server curl -s http://127.0.0.1:9090/prometheus/api/v1/query_range '
           f'--data-urlencode query@/tmp/q.txt -d start={frm // 1000} -d end={now // 1000} -d step={st} '
           "| grep -o '\"executionTimeMsec\":[0-9]*\\|\"status\":\"[a-z]*\"' | tr '\\n' ' '; echo; done")
    out = remote(cmd).strip().splitlines()
    eng = [int(re.search(r'"executionTimeMsec":(\d+)', l).group(1)) for l in out if 'executionTimeMsec' in l]
    stat = sorted({re.search(r'"status":"(\w+)"', l).group(1) for l in out if '"status"' in l})
    res.append(dict(form=form, rng='30d', wall_ms=med(walls), wall_ms_all=[round(w, 1) for w in walls],
                    engine_ms=med(eng) if eng else None, engine_ms_all=eng, vm_status=stat,
                    series=len(ids), step=step, error=err, expr_bytes=len(expr), req_bytes=len(json.dumps(body)),
                    names_in_matcher=len(NODES) if form == 'list' else None))
    print(json.dumps({k: v for k, v in res[-1].items() if k != 'expr'}), flush=True)

tsdb = json.loads(remote("docker exec pmm-server curl -s 'http://127.0.0.1:9090/prometheus/api/v1/status/tsdb?topN=1'"))
total_series = tsdb['data']['totalSeries']
summary = dict(
    run=RUN, now_ms=now, from_ms=frm, total_series=total_series, fleet_nodes=len(NODES), other_node_name_values=other,
    expected_node_annotations_total=len(node_ann_ids), expected_node_annotations_in_range=len(in_range),
    list_equals_dotplus=ids_by_form['list'] == ids_by_form['dotplus'],
    list_subset_of_expected=ids_by_form['list'] <= node_ann_ids,
    list_missing_vs_in_range=len(in_range - ids_by_form['list']),
    results=res)
json.dump(summary, open(OUT, 'w'), indent=1)
print(json.dumps({k: v for k, v in summary.items() if k != 'results'}), flush=True)
