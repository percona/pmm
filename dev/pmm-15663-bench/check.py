#!/usr/bin/env python3
"""Compare a capture's annotation texts with the expected set. usage: check.py cap.json rule from_ms to_ms [env=..] [nodes=a,b] [services=a,b]"""
import json, sys
cap, rule, frm, to = sys.argv[1], sys.argv[2], int(sys.argv[3]), int(sys.argv[4])
kv = dict(a.split('=', 1) for a in sys.argv[5:])
envs = set(kv['env'].split(',')) if kv.get('env') else None
nodes = set(kv['nodes'].split(',')) if kv.get('nodes') else None
svcs = set(kv['services'].split(',')) if kv.get('services') else None
anns = json.load(open('ann-5000.json'))[:int(kv.get('n', 500))]
got = set(json.load(open(cap))['annoTexts'])
glob = {t for t in got if t.startswith('global-')}
exp = set()
for x in anns:
    if not frm <= x['time'] <= to:
        continue
    names = [s['service_name'] for s in x['services']]
    if rule == 'A-node':  # tags [$node_name]: node-scoped only, never on All
        ok = nodes is not None and not names and x['node_name'] in nodes
    elif rule == 'A-svc':  # tags [$node_name, $service_name]
        ok = (nodes is not None and not names and x['node_name'] in nodes) or (svcs is not None and bool(set(names) & svcs))
    elif rule == 'node':
        ok = (envs is None or x['environment'] in envs) and (nodes is None or x['node_name'] in nodes)
    elif rule == 'mysql':
        my = [s['service_name'] for s in x['services'] if s['service_type'] == 'mysql']
        ok = bool(my) and (envs is None or x['environment'] in envs) and (nodes is None or x['node_name'] in nodes) \
             and (svcs is None or bool(set(my) & svcs))
    if ok:
        exp.add(x['text'])
scoped = got - glob
print(json.dumps({'expected': len(exp), 'got': len(scoped), 'missing': len(exp - scoped), 'extra': len(scoped - exp),
                  'globals': len(glob), 'ok': exp == scoped,
                  'sample_missing': sorted(exp - scoped)[:2], 'sample_extra': sorted(scoped - exp)[:2]}))
