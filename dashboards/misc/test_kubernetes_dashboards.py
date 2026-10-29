"""Contract for the provisioned Kubernetes dashboards (PMM-13674).

These checks pin what an edit must keep: uids and links, the cluster label contract,
and the OpenShift/PS query fixes.
"""
import glob
import json
import os
import re
import subprocess
import unittest

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
DASH_DIR = os.path.join(REPO, 'dashboards', 'dashboards')
K8S_DIR = os.path.join(DASH_DIR, 'Kubernetes')
CLEANUP = os.path.join(REPO, 'dashboards', 'misc', 'cleanup-dash.py')
DS = {'type': 'prometheus', 'uid': 'PA58DA793C7250F1B'}
EXPECTED = {
    'Kubernetes_Overview.json': ('k8s-overview', 'Kubernetes Overview'),
    'Kubernetes_Clusters.json': ('k8s-clusters', 'Kubernetes Clusters'),
    'Database_Clusters_on_Kubernetes.json': ('k8s-db-clusters', 'Database Clusters on Kubernetes'),
    'Kubernetes_Cluster_Summary.json': ('k8s-cluster-summary', 'Kubernetes Cluster Summary'),
    'Database_Cluster_on_Kubernetes.json': ('k8s-db-cluster', 'Database Cluster on Kubernetes'),
    'Kubernetes_Pod.json': ('k8s-pod', 'Kubernetes Pod'),
    'Kubernetes_Node.json': ('k8s-node', 'Kubernetes Node'),
}


def load(name):
    with open(os.path.join(K8S_DIR, name)) as f:
        return json.load(f)


def walk(node):
    """Every dict in a dashboard, depth first."""
    if isinstance(node, dict):
        yield node
        for v in node.values():
            yield from walk(v)
    elif isinstance(node, list):
        for v in node:
            yield from walk(v)


def exprs(dash):
    return [d['expr'] for d in walk(dash) if isinstance(d.get('expr'), str)]


def all_uids():
    uids = set()
    for path in glob.glob(os.path.join(DASH_DIR, '**', '*.json'), recursive=True):
        with open(path) as f:
            uid = json.load(f).get('uid')
        if uid:
            uids.add(uid)
    return uids


class KubernetesDashboardsTest(unittest.TestCase):
    def test_files_uids_and_titles(self):
        self.assertEqual(sorted(os.listdir(K8S_DIR)), sorted(EXPECTED))
        for name, (uid, title) in EXPECTED.items():
            d = load(name)
            self.assertEqual((d['uid'], d['title']), (uid, title), name)
            self.assertEqual(d['tags'], ['Kubernetes', 'Percona'], name)

    def test_cleanup_check_passes(self):
        for name in EXPECTED:
            r = subprocess.run(['python3', CLEANUP, '--check-only', os.path.join(K8S_DIR, name)],
                               capture_output=True, text=True)
            self.assertEqual(r.returncode, 0, name + '\n' + r.stdout)

    def test_no_dev_uids_left(self):
        for name in EXPECTED:
            with open(os.path.join(K8S_DIR, name)) as f:
                self.assertNotIn('k8s-v2', f.read(), name)

    def test_datasource_is_the_object_form(self):
        for name in EXPECTED:
            for d in walk(load(name)):
                if 'datasource' in d:
                    self.assertEqual(d['datasource'], DS, name)

    def test_cluster_variables_match_only_labelled_clusters(self):
        # ".*" also matches series with no k8s_cluster_id (a phantom cluster). Other "All"
        # variables (the Pod page's repeated container) keep Grafana's value list.
        for name in EXPECTED:
            for v in load(name)['templating']['list']:
                if v['name'] == 'k8s_cluster' and v.get('includeAll'):
                    self.assertEqual(v.get('allValue'), '.+', name)

    def test_root_filesystem_accepts_rhcos(self):
        for name in EXPECTED:
            for e in exprs(load(name)):
                self.assertNotIn('mountpoint="/"', e, name)

    def test_percona_server_metric_name(self):
        for name in EXPECTED:
            for e in exprs(load(name)):
                self.assertNotIn('kube_ps_status_state', e, name)
                self.assertNotIn('|ps)_status_state', e, name)

    def test_overview_joins_keep_the_cluster(self):
        # The Overview spans several clusters; node and namespace names repeat across them.
        for e in exprs(load('Kubernetes_Overview.json')):
            for keys in re.findall(r'\bon \(([^)]*)\)', e):
                keys = [k.strip() for k in keys.split(',') if k.strip()]
                if keys and keys != ['service_name']:
                    self.assertIn('k8s_cluster_id', keys, e)

    def test_link_targets_exist(self):
        uids = all_uids()
        for name in EXPECTED:
            text = json.dumps(load(name))
            static = set(re.findall(r'/graph/d/([A-Za-z0-9_-]+)', text))
            generated = set(re.findall(r'\\"(?:goto|pmm_dash|pmm_cdash)\\", \\"([A-Za-z0-9_-]+)\\"', text))
            for uid in static | generated:
                self.assertIn(uid, uids, f'{name} links to {uid}')

    def test_overview_explains_an_empty_pmm(self):
        panels = {p['title']: p for p in load('Kubernetes_Clusters.json')['panels']}
        msg = panels['Kubernetes clusters']['fieldConfig']['defaults']['noValue']
        self.assertIn('PMM documentation', msg)

    def test_no_problem_state_is_not_red(self):
        # "✓ None" is the tile's no-value text; it takes the base threshold colour, so the base must not be the alarm colour.
        for name in EXPECTED:
            for p in walk(load(name)):
                if p.get('type') == 'stat' and p.get('fieldConfig', {}).get('defaults', {}).get('noValue') == '✓ None':
                    steps = p['fieldConfig']['defaults']['thresholds']['steps']
                    self.assertNotEqual(steps[0]['color'], '#d03b3b', f"{name}: {p['title']}")
                    self.assertIn({'color': '#d03b3b', 'value': 1}, steps, f"{name}: {p['title']}")

    def _summary_rows(self):
        panels = sorted(load('Kubernetes_Overview.json')['panels'], key=lambda p: (p['gridPos']['y'], p['gridPos']['x']))
        return panels[0], panels[1]

    def test_overview_summary_rows(self):
        # Inventory is one Canvas (one-line "label value" tiles); problems are one stat panel. Each row's hover menu
        # sits over its last tile, so that tile's value is moved clear of it.
        inventory, problems = self._summary_rows()
        self.assertEqual((inventory['type'], problems['type']), ('canvas', 'stat'))
        self.assertEqual((len(inventory['targets']), len(problems['targets'])), (7, 5))
        self.assertTrue(inventory['transparent'])
        self.assertEqual(inventory['options']['tooltip']['mode'], 'none', 'no data pop-up over the numbers')
        els = inventory['options']['root']['elements']
        tiles = [e for e in els if e['type'] == 'rectangle' and not e.get('links')]
        self.assertEqual(min(e['placement']['right'] for e in tiles), 0)
        last = [e for e in els if e['type'] == 'text' and e['name'].endswith(str(len(tiles) - 1))]
        self.assertGreaterEqual(min(e['placement']['right'] for e in last), 3, 'last value kept clear of the panel menu')
        # Canvas clicks only on an element's own oneClick link (field links are ignored): one layer on top of each tile.
        clicks = [e for e in els if e.get('links')]
        self.assertEqual(els[-len(tiles):], clicks, 'every tile has a click layer, drawn last')
        # Each tile opens the list its number counts.
        want = [('k8s-clusters', 'Kubernetes clusters'), ('k8s-clusters', 'Kubernetes clusters'), ('k8s-db-clusters', 'Database clusters'),
                ('k8s-db-clusters', 'Database pods')]
        dashes = {load(n)['uid']: load(n) for n in EXPECTED}
        for e, (uid, title) in zip(clicks, want):
            (l,) = e['links']
            self.assertTrue(l['oneClick'], e['name'])
            pid = next(p['id'] for p in walk(dashes[uid]) if p.get('type') == 'table' and p.get('title') == title)
            self.assertTrue(l['url'].startswith(f'/graph/d/{uid}?viewPanel=panel-{pid}&'), f"{e['name']}: {l['url']}")
        self.assertEqual((problems['title'], problems['description'], problems['options']['orientation']), ('', '', 'vertical'))
        ovs = problems['fieldConfig']['overrides']
        self.assertEqual([o['matcher']['options'] for o in ovs], [t['legendFormat'] for t in problems['targets']])
        for o in ovs:
            self.assertTrue(o['properties'][0]['value'][0]['title'].endswith('Click to see which.'), o['matcher']['options'])
        self.assertTrue(dict((q['id'], q['value']) for q in ovs[-1]['properties'])['unit'].startswith('suffix:\u00a0'))

    def test_overview_totals_render_as_slash(self):
        inventory, _ = self._summary_rows()
        units = {o['matcher']['options']: o['properties'][0]['value'] for o in inventory['fieldConfig']['overrides'] if o['properties'][0]['id'] == 'unit'}
        self.assertEqual(sorted(units.values()), ['prefix:/'] * 3)

    def test_overview_problem_tiles_quiet_when_zero(self):
        for problems in (p for p in load('Kubernetes_Overview.json')['panels'] if p['type'] == 'stat'):
            d = problems['fieldConfig']['defaults']
            self.assertNotEqual(d['thresholds']['steps'][0]['color'], '#7a2230')
            self.assertIn({'color': '#7a2230', 'value': 1}, d['thresholds']['steps'])
            self.assertEqual(d['mappings'][0]['options']['0']['text'], '✓')
            for t in problems['targets']:
                self.assertTrue(t['expr'].endswith('or on () vector(0)'), t['legendFormat'])

    def test_overview_is_tiles_and_one_attention_list(self):
        # The Overview lists only what needs attention, so it stays short at scale; full lists are separate dashboards.
        panels = sorted(load('Kubernetes_Overview.json')['panels'], key=lambda p: (p['gridPos']['y'], p['gridPos']['x']))
        self.assertEqual([p['type'] for p in panels], ['canvas', 'stat', 'stat', 'table'])
        tiles = [t['legendFormat'] for p in panels[1:3] for t in p['targets']]
        self.assertEqual(len(tiles), 10)
        for new in ('Backups failed', 'Operators down', 'Pods pending'):
            self.assertIn(new, tiles)
        attn = panels[3]
        self.assertEqual(attn['title'], 'Needs attention')
        # one query, so an empty result is still one row ("✓ Nothing needs attention"), never a bare grid
        self.assertEqual(len(attn['targets']), 1)
        q = attn['targets'][0]['expr']
        self.assertIn('✓ Nothing needs attention', q)
        # "restarts > 0 * on(...)" binds as "restarts > (0 * ...)", so the comparison is parenthesized.
        self.assertRegex(q, r'\(\(\(sum by \(k8s_cluster_id, namespace, pod\) \(round\(increase_prometheus\(kube_pod_container_status_restarts_total[^)]*\)\)\) > 0\) and on [^*]*\)\) \* on \(k8s_cluster_id, namespace, pod\) group_left \(terminated\)')
        # an operator that restarts for a few seconds is not "down"
        self.assertIn('max_over_time(kube_deployment_status_replicas_available', q)
        # restarts look back over the range: a pod deleted since then does not need attention
        self.assertIn('> 0) and on (k8s_cluster_id, namespace, pod) max by (k8s_cluster_id, namespace, pod) (kube_pod_info', q)

    def test_database_clusters_list_filters(self):
        d = load('Database_Clusters_on_Kubernetes.json')
        vars_ = {v['name']: v for v in d['templating']['list']}
        for name in ('k8s_cluster', 'namespace'):
            self.assertTrue(vars_[name]['multi'], name)
        self.assertEqual([p['title'] for p in d['panels']], ['Database clusters', 'Database pods'])

    def test_database_pods_leave_out_job_pods(self):
        # Backup Job pods are not database pods: a finished one is never Ready.
        for name in ('Kubernetes_Overview.json', 'Kubernetes_Cluster_Summary.json', 'Database_Clusters_on_Kubernetes.json'):
            for e in exprs(load(name)):
                if 'kube_pod_container_info' in e:
                    self.assertIn('owner_kind="Job"', e, name)

    def test_pmm_status_covers_haproxy(self):
        # PMM monitors operator HAProxy pods through an external exporter; they have no <engine>_up metric.
        for name in EXPECTED:
            for e in exprs(load(name)):
                if 'mysql_up' in e:
                    self.assertIn('external-exporter', e, name)

    def test_tile_labels_have_room(self):
        # Canvas text does not shrink, so a label must get enough of the tile width to stay on one line.
        for name in EXPECTED:
            for p in walk(load(name)):
                if p.get('type') != 'canvas':
                    continue
                els = {e['name']: e['placement'] for e in p['options']['root']['elements']}
                for n, tile in els.items():
                    if n.startswith('tile'):
                        lab = els['label' + n[4:]]
                        w = 100 - tile['left'] - tile['right']
                        self.assertGreaterEqual(100 - lab['left'] - lab['right'], 0.8 * w - 0.6 - 0.01, f'{name} {n}')

    def test_db_cluster_header_is_one_tile_row(self):
        d = load('Database_Cluster_on_Kubernetes.json')
        row = min(d['panels'], key=lambda p: (p['gridPos']['y'], p['gridPos']['x']))
        self.assertEqual(row['type'], 'canvas')
        self.assertEqual([t['legendFormat'] for t in row['targets']], ['Engine', 'State', 'Pods ready', 'Pods ready · total', 'Restarts'])
        els = {e['name']: e for e in row['options']['root']['elements']}
        # Engine shows its name (a label turned into a column) and opens that engine's PMM dashboards.
        self.assertEqual(els['value0']['config']['text']['field'], 'engine')
        self.assertIn({'id': 'labelsToFields', 'options': {'mode': 'columns', 'keepLabels': ['engine', 'pmm_cdash']},
                       'filter': {'id': 'byRefId', 'options': 'A'}}, row['transformations'])
        self.assertTrue(els['click0']['links'][0]['url'].startswith('/graph/d/${__data.fields.pmm_cdash}?var-cluster=$db_cluster'))
        pods = next(p['id'] for p in walk(d) if p.get('type') == 'table' and p.get('title') == 'Pods')
        for i in (2, 3):
            self.assertIn(f'viewPanel=panel-{pods}&', els[f'click{i}']['links'][0]['url'], i)
        # State and Restarts are status tiles: the tile colour follows the value.
        for i, label in ((1, 'State'), (3, 'Restarts')):
            self.assertEqual(els[f'tile{i}']['background']['color']['field'], label)

    def test_headers_are_one_tile_row(self):
        # Each dashboard opens with one Canvas row of one-line tiles; a linked tile opens the list behind its number.
        here = lambda title: ('', title)
        want = {
            'Kubernetes_Cluster_Summary.json': [('Nodes ready', here('Nodes')), ('Pods running', here('Pods')),
                                                ('DB clusters ready', here('Database clusters')), ('CPU limits', here('Namespaces')),
                                                ('Memory limits', here('Namespaces')), ('Namespaces', here('Namespaces'))],
            'Kubernetes_Pod.json': [('Phase', here('Containers')), ('Ready', here('Containers')), ('Restarts', here('Containers')), ('Age', None),
                                    ('Node', ('k8s-node', None)), ('DB cluster', ('k8s-db-cluster', None)), ('In PMM', ('${__data.fields.pmm_dash}', None))],
            'Kubernetes_Node.json': [('Status', None), ('Kubelet', None), ('OS', None), ('CPU requested', here('Pods on this node')),
                                     ('Memory requested', here('Pods on this node')), ('Pods / max', here('Pods on this node'))],
        }
        for name, tiles in want.items():
            d = load(name)
            row = min(d['panels'], key=lambda p: (p['gridPos']['y'], p['gridPos']['x']))
            self.assertEqual((row['type'], row['gridPos']['h']), ('canvas', 2), name)
            els = {e['name']: e for e in row['options']['root']['elements']}
            self.assertEqual([els[f'label{i}']['config']['text']['fixed'] for i in range(len(tiles))], [t[0] for t in tiles], name)
            for i, (label, target) in enumerate(tiles):
                click = els.get(f'click{i}')
                if target is None:
                    self.assertIsNone(click, f'{name} {label}')
                    continue
                uid, table = target
                url = click['links'][0]['url']
                self.assertTrue(url.startswith(f"/graph/d/{uid or d['uid']}?"), f'{name} {label}: {url}')
                if table:
                    pid = next(p['id'] for p in walk(d) if p.get('type') == 'table' and p.get('title') == table)
                    self.assertIn(f'viewPanel=panel-{pid}&', url, f'{name} {label}')
                    for v in [t['name'] for t in d['templating']['list']]:
                        self.assertIn(f'${{{v}:queryparam}}', url, f'{name} {label} keeps ${v}')

    def test_data_links_see_every_query(self):
        # A Canvas link reads ${__data.fields.x} from the first frame only: rows whose links use data concatenate all frames.
        for name in EXPECTED:
            for p in walk(load(name)):
                if p.get('type') == 'canvas' and '${__data.fields.' in json.dumps(p['options']):
                    self.assertEqual(p['transformations'][-1], {'id': 'concatenate', 'options': {'frameNameMode': 'drop'}}, name)

    def test_percent_of_limit_is_per_container(self):
        # Each container is divided by its own limit, then the busiest is taken: summing usage over only the
        # limited containers' limits overstates the pod.
        lim = [e for e in exprs(load('Database_Cluster_on_Kubernetes.json'))
               if 'kube_pod_container_resource_limits' in e and ('container_cpu_usage_seconds_total' in e or 'container_memory_working_set_bytes' in e)]
        self.assertGreaterEqual(len(lim), 4)
        for e in lim:
            self.assertIn('sum by (k8s_cluster_id, namespace, pod, container) (kube_pod_container_resource_limits', e)

    def test_limits_look_like_limits(self):
        # Limit lines are red; % of limit warns at 80% and alarms at 100%.
        for p in load('Kubernetes_Pod.json')['panels']:
            for o in p.get('fieldConfig', {}).get('overrides', []):
                if o['matcher']['options'] == 'Limit':
                    self.assertEqual(next(x['value'] for x in o['properties'] if x['id'] == 'color')['fixedColor'], '#d03b3b', p['title'])
        d = load('Database_Cluster_on_Kubernetes.json')
        pct = [p for p in d['panels'] if '% of limit' in p['title']]
        self.assertEqual(len(pct), 2)
        for p in pct:
            self.assertEqual([s['value'] for s in p['fieldConfig']['defaults']['thresholds']['steps']], [None, 0.8, 1], p['title'])

    def test_db_cluster_shows_components_without_limits(self):
        # Proxies, config servers and orchestrators often have no limit, so they never appear in "% of limit" charts.
        titles = [p['title'] for p in load('Database_Cluster_on_Kubernetes.json')['panels']]
        self.assertIn('CPU used, per pod', titles)
        self.assertIn('Memory used, per pod', titles)

    def test_storage_shows_free_bytes(self):
        # A percentage hides the size: every PVC table and fullest-disk tile also shows free bytes.
        for name in EXPECTED:
            for p in walk(load(name)):
                if p.get('type') == 'table' and 'PVC' in json.dumps(p.get('transformations', [])):
                    cols = p['transformations'][1]['options']['renameByName'].values()
                    self.assertIn('Free', cols, f"{name}: {p['title']}")
        row = self._storage_row()
        legends = [t['legendFormat'] for t in row['targets']]
        for tile in ('Fullest volume', 'Fullest node disk'):
            self.assertIn(f'{tile} · extra', legends)
            unit = next(o for o in row['fieldConfig']['overrides'] if o['matcher']['options'] == f'{tile} · extra')['properties']
            self.assertIn({'id': 'unit', 'value': 'suffix: GiB free'}, unit, tile)

    def _storage_row(self):
        cs = load('Kubernetes_Cluster_Summary.json')
        storage = next(p for p in cs['panels'] if p['type'] == 'row' and p['title'] == 'Storage')['gridPos']['y']
        return next(p for p in cs['panels'] if p['type'] == 'canvas' and p['gridPos']['y'] == storage + 1)

    def test_storage_row_is_one_tile_row(self):
        # Same one-line tiles as the headers, each opening the list behind it.
        row = self._storage_row()
        els = {e['name']: e for e in row['options']['root']['elements']}
        want = [('PVCs bound', 'Persistent volume claims'), ('Requested', 'Persistent volume claims'),
                ('Fullest volume', 'Persistent volume claims'), ('Fullest node disk', 'Nodes'), ('PVCs not bound', 'Persistent volume claims')]
        self.assertEqual([els[f'label{i}']['config']['text']['fixed'] for i in range(len(want))], [w[0] for w in want])
        self.assertIn('PVCs bound · total', [t['legendFormat'] for t in row['targets']])
        ids = {p['title']: p['id'] for p in load('Kubernetes_Cluster_Summary.json')['panels'] if p['type'] == 'table'}
        for i, (_, table) in enumerate(want):
            self.assertIn(f'viewPanel=panel-{ids[table]}&', els[f'click{i}']['links'][0]['url'], table)
        # no data must not leave Canvas's "Double click to set text" placeholder
        for t in row['targets']:
            if t['legendFormat'].startswith('Fullest'):
                self.assertTrue(t['expr'].endswith('or on () vector(-1)'), t['legendFormat'])

    def test_pod_tables_say_why_containers_restarted(self):
        # Every pod list that counts restarts also shows the last termination reason, one row per pod.
        for name, title in (('Database_Cluster_on_Kubernetes.json', 'Pods'), ('Database_Clusters_on_Kubernetes.json', 'Database pods'),
                            ('Kubernetes_Cluster_Summary.json', 'Pods that need attention'), ('Kubernetes_Pod.json', 'Containers')):
            p = next(p for p in walk(load(name)) if p.get('type') == 'table' and p.get('title') == title)
            self.assertIn('Last termination', p['transformations'][1]['options']['renameByName'].values(), f'{name}: {title}')
            if title != 'Containers':
                self.assertIn('topk by (k8s_cluster_id, namespace, pod) (1,', json.dumps(p['targets']), f'{name}: {title} one row per pod')

    def test_state_bars_bridge_short_gaps(self):
        # A kube-state-metrics restart leaves ~1 min without samples: bridge gaps up to 10 min, show longer ones.
        for name in EXPECTED:
            for p in walk(load(name)):
                if p.get('type') == 'state-timeline':
                    self.assertEqual(p['fieldConfig']['defaults']['custom'].get('spanNulls'), 600000, f"{name}: {p['title']}")

    def test_every_panel_says_why_it_is_empty(self):
        # Every data panel says why it is empty instead of "No data".
        for name in EXPECTED:
            for p in load(name)['panels']:
                if p['type'] in ('row', 'canvas'):
                    continue
                self.assertTrue(p['fieldConfig']['defaults'].get('noValue'), f"{name}: {p.get('title') or p['type']}")

    def test_lists_filter_and_link(self):
        # Lists filter from their headers, and every number opens its list.
        for name in EXPECTED:
            for p in walk(load(name)):
                if p.get('type') == 'table':
                    self.assertTrue(p['fieldConfig']['defaults']['custom']['filterable'], f"{name}: {p['title']}")
        cs = load('Kubernetes_Cluster_Summary.json')
        ns = next(v for v in cs['templating']['list'] if v['name'] == 'namespace')
        self.assertEqual((ns['multi'], ns['includeAll'], ns['allValue']), (True, True, '.+'))
        pods = next(p for p in walk(cs) if p.get('type') == 'table' and p.get('title') == 'Pods')
        self.assertTrue(pods['options']['footer']['enablePagination'])
        self.assertTrue(all('namespace=~"$namespace"' in t['expr'] for t in pods['targets']))
        ids = {p['title']: p['id'] for p in walk(cs) if p.get('type') == 'table'}
        cl = next(p for p in load('Kubernetes_Clusters.json')['panels'] if p.get('title') == 'Kubernetes clusters')
        links = {o['matcher']['options']: o['properties'] for o in cl['fieldConfig']['overrides']}
        for col, table in (('Nodes', 'Nodes'), ('Ready', 'Nodes'), ('DB clusters', 'Database clusters'),
                           ('Pods running', 'Pods'), ('Restarts', 'Pods that need attention')):
            url = next(p['value'][0]['url'] for p in links[col] if p['id'] == 'links')
            self.assertTrue(url.startswith(f"/graph/d/k8s-cluster-summary?viewPanel=panel-{ids[table]}&var-k8s_cluster=${{__data.fields.k8s_cluster_id}}"), f'{col}: {url}')

    def test_node_numbers_explain_themselves(self):
        # Panels whose numbers need a definition ("allocatable", "pods / max") explain them.
        for name in ('Kubernetes_Cluster_Summary.json', 'Kubernetes_Node.json'):
            for p in load(name)['panels']:
                if 'allocatable' in p.get('title', ''):
                    self.assertIn('kubelet', p['description'], f"{name}: {p['title']}")
        for p in load('Kubernetes_Node.json')['panels']:
            if p.get('title', '').startswith('Disk '):
                self.assertIn('one line per device', p['description'].lower(), p['title'])

    def test_pods_list_shows_only_existing_pods(self):
        # Restarts look back over the whole time range: only pods that still exist are listed.
        pods = next(p for p in walk(load('Kubernetes_Cluster_Summary.json')) if p.get('type') == 'table' and p.get('title') == 'Pods')
        for t in pods['targets']:
            if 'restarts_total' in t['expr'] or 'last_terminated_reason' in t['expr']:
                self.assertIn('and on (k8s_cluster_id, namespace, pod) max by (k8s_cluster_id, namespace, pod) (kube_pod_info', t['expr'], t['refId'])

    def test_detects_databases_from_any_operator(self):
        # A database pod runs a database image (any operator, Helm chart or StatefulSet); its cluster is the
        # app.kubernetes.io/instance label, else the owning controller.
        for name in EXPECTED:
            for e in exprs(load(name)):
                self.assertNotIn('percona-.+-operator', e, name)
        inv = self._summary_rows()[0]
        pods = next(t['expr'] for t in inv['targets'] if t['legendFormat'] == 'Database pods ready · total')
        for marker in ('kube_pod_container_info', 'mongo', 'postgres', 'mysql', 'valkey', 'operator',
                       'label_app_kubernetes_io_instance', 'owner_name'):
            self.assertIn(marker, pods)

    def test_cluster_state_without_an_operator_state(self):
        # Clusters without a Percona custom resource get their state from pod readiness: Ready, Degraded, Down.
        d = load('Database_Clusters_on_Kubernetes.json')
        t = next(p for p in d['panels'] if p['title'] == 'Database clusters')
        state = next(x['expr'] for x in t['targets'] if x['refId'] == 'B')
        self.assertIn('kube_(psmdb|pxc|pg|ps_mysql)_status_state', state)
        self.assertIn('kube_pod_status_ready', state)
        maps = next(o for o in t['fieldConfig']['overrides'] if o['matcher']['options'] == 'State')['properties'][0]['value'][0]['options']
        self.assertEqual((maps['5']['text'], maps['6']['text']), ('◐ Degraded', '✕ Down'))
        dbc = load('Database_Cluster_on_Kubernetes.json')
        stt = next(p for p in dbc['panels'] if p['type'] == 'state-timeline')
        self.assertIn('6', stt['fieldConfig']['defaults']['mappings'][0]['options'])
        vars_ = {v['name']: v for v in dbc['templating']['list']}
        for v in ('namespace', 'db_cluster'):
            self.assertIn('query_result(', vars_[v]['definition'], v)

    def test_tiles_never_show_the_canvas_placeholder(self):
        # A missing field makes Canvas print "Double click to set text": every tile query falls back to -1 (or "—"),
        # and every shown field maps -1 to "—" (blank for totals and extras).
        for name in EXPECTED:
            for p in walk(load(name)):
                if p.get('type') != 'canvas':
                    continue
                maps = {o['matcher']['options']: json.dumps(o['properties']) for o in p['fieldConfig']['overrides']}
                for t in p['targets']:
                    where = f"{name}: {t['legendFormat']}"
                    self.assertRegex(t['expr'], r'or on \(\) (vector\(-1\)|label_replace\(vector\(-?\d\), "[a-z_]+", "[^"]*", "", ""\))$', where)
                    if not t['legendFormat'].endswith(' · link'):
                        self.assertIn('"-1"', maps.get(t['legendFormat'], ''), where)

    def test_one_line_per_object_gets_its_own_colour(self):
        # A chart with one series per pod, container, PVC or disk uses the palette, so each series has its own colour.
        for name in EXPECTED:
            for p in load(name)['panels']:
                if p['type'] != 'timeseries' or not re.search(r'\{\{(pod|container|persistentvolumeclaim|device)\}\}', json.dumps(p['targets'])):
                    continue
                self.assertEqual(p['fieldConfig']['defaults']['color']['mode'], 'palette-classic', f"{name}: {p['title']}")
                self.assertFalse([o for o in p['fieldConfig']['overrides'] if o['matcher']['id'] == 'byFrameRefID'], f"{name}: {p['title']}")
        legends = {p['title']: [t['legendFormat'] for t in p['targets']] for p in load('Database_Cluster_on_Kubernetes.json')['panels']}
        for title in ('Network — received (+) / sent (−), per pod', 'Disk I/O, per pod'):
            self.assertTrue(all('{{component}}' in l for l in legends[title]), title)

    def test_queries_have_no_stray_escapes(self):
        # A backslash before a quote is a generator escaping bug, never valid PromQL here.
        for name in EXPECTED:
            for e in exprs(load(name)):
                self.assertNotIn('\\"', e, name)

    def test_experimental_folder_is_gone(self):
        self.assertFalse(os.path.exists(os.path.join(DASH_DIR, 'Kubernetes (experimental)')))


if __name__ == '__main__':
    unittest.main()
