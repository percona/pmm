#!/usr/bin/env python3
"""Correctness matrix: every variant x dashboard x case, checked against the expected set. usage: matrix.py N_ANN OUTDIR"""
import json, os, subprocess, sys, time, urllib.parse
N, OUT = sys.argv[1], sys.argv[2]
os.makedirs(OUT, exist_ok=True)
UID = {'home': 'pmm-home', 'nodes': 'node-instance-overview', 'mysql': 'mysql-instance-overview'}
ALL_NODES = [f'node-{i:04d}' for i in range(1000)]
def env_nodes(env):
    return [n for i, n in enumerate(ALL_NODES) if env is None or f'env-{i % 5}' == env]
CASES = [  # name, env, nodes, services, ranges
    ('all', None, None, None, ['6h', '7d', '30d']),
    ('env', 'env-4', None, None, ['7d']),
    ('node', None, ['node-0000'], None, ['6h', '7d', '30d']),
    ('multi', None, ['node-0000', 'node-0459', 'node-0856'], None, ['7d']),
    ('svc', None, ['node-0856'], ['node-0856-mysql-1'], ['6h', '30d']),
]
H = {'6h': 6 * 3600_000, '7d': 7 * 86400_000, '30d': 30 * 86400_000}
rows = []
to = int(time.time() * 1000)
for dash in ('home', 'nodes', 'mysql'):
    for variant in ('main', 'pr', 'E', 'K'):
        for case, env, nodes, svcs, ranges in CASES:
            if case == 'svc' and dash != 'mysql':
                continue
            for rng in ranges:
                frm = to - H[rng]
                q = [('orgId', '1'), ('from', str(frm)), ('to', str(to))]
                if env:
                    q.append(('var-environment', env))
                for n in nodes or []:
                    q.append(('var-node_name', n))
                for s in svcs or []:
                    q.append(('var-service_name', s))
                path = f'/graph/d/{variant}-{UID[dash]}?' + urllib.parse.urlencode(q)
                cap = f'{OUT}/{dash}-{variant}-{case}-{rng}.json'
                r = subprocess.run(['node', 'capture.js', path, cap], capture_output=True, text=True)
                if r.returncode:
                    rows.append(dict(dash=dash, variant=variant, case=case, rng=rng, error=r.stderr[-300:]))
                    print(rows[-1], flush=True)
                    continue
                summ = json.loads(r.stdout)
                if variant in ('main', 'pr'):
                    rule = 'A-svc' if dash == 'mysql' else 'A-node'
                    # main expands All into every value as a tag; the PR sends '.+' which matches no tag
                    n_arg = nodes if nodes else (env_nodes(env) if variant == 'main' else None)
                    s_arg = svcs if svcs else None
                    if dash == 'mysql' and variant == 'main' and not svcs:
                        s_arg = [f'{n}-mysql-{k}' for n in (nodes or env_nodes(env)) for k in (0, 1)]
                else:
                    rule = 'mysql' if dash == 'mysql' else 'node'
                    n_arg, s_arg = nodes, svcs
                args = ['python3', 'check.py', cap, rule, str(frm), str(to), f'n={N}']
                if env and variant in ('E', 'K'):
                    args.append(f'env={env}')
                if n_arg:
                    args.append('nodes=' + ','.join(n_arg))
                if s_arg:
                    args.append('services=' + ','.join(s_arg))
                chk = json.loads(subprocess.run(args, capture_output=True, text=True).stdout)
                rows.append(dict(dash=dash, variant=variant, case=case, rng=rng, loadMs=summ['loadMs'], failed=summ['failed'],
                                 maxBody=summ['maxBody'], maxUrl=summ['maxUrl'], panelErrors=summ['panelErrors'],
                                 anno=summ['anno'], **chk))
                print(json.dumps(rows[-1])[:400], flush=True)
json.dump(rows, open(f'{OUT}/matrix.json', 'w'), indent=1)
