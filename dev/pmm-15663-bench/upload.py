#!/usr/bin/env python3
import base64, glob, json, os, ssl, urllib.request
BASE = 'https://pmmqa-pmm-15663-bench.tp.int.percona.com/graph'
A = 'Basic ' + base64.b64encode(b'admin:Bench-15663').decode()
ctx = ssl._create_unverified_context()
def call(path, body=None, method='POST'):
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode() if body is not None else None, method=method,
                                 headers={'Authorization': A, 'Content-Type': 'application/json'})
    try:
        return json.load(urllib.request.urlopen(req, context=ctx))
    except urllib.error.HTTPError as e:
        return {'error': e.code, 'body': e.read().decode()[:200]}
D = os.path.dirname(os.path.abspath(__file__)) + '/dash/out'
for v in sorted(os.listdir(D)):
    call('/api/folders', {'uid': f'bench-{v}', 'title': f'bench-{v}'})
    for f in glob.glob(f'{D}/{v}/*.json'):
        r = call('/api/dashboards/db', {'dashboard': json.load(open(f)), 'folderUid': f'bench-{v}', 'overwrite': True})
        print(v, os.path.basename(f), r.get('status', r))
