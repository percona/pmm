#!/usr/bin/env python3
"""Node- and service-scoped annotations come from VictoriaMetrics, not Grafana tags (PMM-15663).

A Grafana tag must equal the variable value, so on All (allValue ".+") a $node_name or
$service_name tag matches nothing. Dashboards read the pmm_annotation_event series instead,
with the same label filters as their panels.

Run from the repo root:  python3 -m unittest discover -s dashboards/misc -p 'test_*.py' -v
"""

import glob
import json
import os
import re
import unittest

REPO = os.environ.get('PMM_REPO', os.getcwd())
DASH_DIR = os.path.join(REPO, 'dashboards', 'dashboards')

GLOBAL_ENTRY = 'PMM Annotations'
SCOPED_ENTRY = 'PMM Annotations: nodes and services'

# Node_Temperature_Details names its hidden service variable "service".
FLEET_VARIABLES = ('node_name', 'service_name', 'service')

# Labels a fleet variable filters on, in matcher order. Node series carry only node labels.
SERVICE_LABELS = ('environment', 'cluster', 'replication_set', 'region', 'az', 'node_type', 'node_name',
                  'service_name')
NODE_LABELS = ('region', 'az', 'node_type', 'node_name')

SERVICE_TYPES = {'mysql', 'postgresql', 'mongodb', 'valkey', 'proxysql', 'haproxy'}

# Their "PMM Annotations" never had a node or service tag, so they show global annotations only.
GLOBAL_ONLY = {
    # Tagged $namespace,$pod: Kubernetes objects, not PMM nodes or services.
    'Experimental/DB_Cluster_Summary.json',
    'MongoDB/MongoDB_Cluster_Summary.json',
    'MongoDB/MongoDB_Collections_Overview.json',
    'MongoDB/MongoDB_Unused_Indexes.json',
    'MySQL/HAProxy_Instance_Summary.json',
    'MySQL/PXC_Galera_Cluster_Summary.json',
    'MySQL/ProxySQL_Instance_Summary.json',
}

# A $var reference, plain or ${var:format}.
VAR_REF = r'\$(?:\{{{name}(?::\w+)?\}}|{name}(?![A-Za-z0-9_]))'
SCOPED_TAG = re.compile('|'.join(VAR_REF.format(name=n) for n in FLEET_VARIABLES))

TERM = (r'max by \(pmm_annotation_id, pmm_annotation_text, pmm_annotation_tags\) '
        r'\(last_over_time\(pmm_annotation_event\{(.*)\}'
        r'\[\$__interval\] offset -\$__interval\)\)')
MATCHER = re.compile(r'(\w+)(=~|!=|=)"([^"]*)"')


def first_variables(dashboard):
    """Grafana resolves a duplicated variable name to its first definition."""
    seen = {}
    for var in dashboard.get('templating', {}).get('list', []):
        seen.setdefault(var.get('name'), var)
    return seen


def base(name):
    """The fleet variable a hidden <var>_list narrows."""
    return name.removesuffix('_list') if name in ('node_name_list', 'service_name_list') else name


def label_of(name):
    return 'service_name' if base(name) == 'service' else base(name)


def var_query(var):
    query = var.get('query')
    return query.get('query', '') if isinstance(query, dict) else query or ''


def descendants(variables, root):
    """Variables whose query depends on $root, directly or through another variable."""
    found, todo = set(), [root]
    while todo:
        parent = todo.pop()
        for name, var in variables.items():
            if name != root and name not in found and re.search(VAR_REF.format(name=parent), var_query(var)):
                found.add(name)
                todo.append(name)
    return found


def exprs(panels):
    for panel in panels:
        yield from exprs(panel.get('panels', []))
        for target in panel.get('targets', None) or []:
            if isinstance(target.get('expr'), str):
                yield target['expr']


def panel_filters(dashboard):
    """(label, variable) pairs the panels filter on, as label="$variable" or label=~"$variable"."""
    names = set(first_variables(dashboard))
    pairs = set()
    for expr in exprs(dashboard.get('panels', [])):
        for label, op, value in MATCHER.findall(expr):
            for name in names:
                if label == label_of(name) and op != '!=' and re.search(VAR_REF.format(name=re.escape(name)), value):
                    pairs.add((label, name))
    return pairs


def uses_node_list(variables):
    """On All, node_name ".+" drops the environment filter that node series can't carry."""
    node = variables.get('node_name') or {}
    return bool(node.get('includeAll') and node.get('allValue') and 'environment' in variables)


def is_prometheus(ds):
    """A datasource name such as "Metrics", or a reference with a uid, as the panels use."""
    if isinstance(ds, str):
        return not ds.startswith('-- ')
    return isinstance(ds, dict) and bool(ds.get('uid')) and ds.get('type', 'prometheus') == 'prometheus'


def entries(dashboard, name):
    return [a for a in dashboard.get('annotations', {}).get('list', []) if a.get('name') == name]


def terms(expr):
    """{'node' | 'service': [(label, op, value), ...]} for a scoped entry's expression."""
    out = {}
    for part in expr.split(' or '):
        m = re.fullmatch(TERM, part)
        if not m:
            raise ValueError(f'term does not match the expected shape: {part}')
        matchers = MATCHER.findall(m.group(1))
        if ','.join(f'{l}{o}"{v}"' for l, o, v in matchers) != m.group(1):
            raise ValueError(f'unparsable matchers: {m.group(1)}')
        kind = {('service_name', '!=', ''): 'service', ('service_name', '=', ''): 'node'}.get(matchers[0])
        if not kind or kind in out:
            raise ValueError(f'each term starts with service_name!="" or service_name="", once: {part}')
        out[kind] = matchers[1:]
    return out


class TestScopedAnnotations(unittest.TestCase):
    def dashboards(self):
        paths = sorted(glob.glob(os.path.join(DASH_DIR, '**', '*.json'), recursive=True))
        # Run from the wrong directory, every check here would pass on nothing.
        self.assertTrue(paths, f'no dashboards found under {DASH_DIR}')
        for path in paths:
            with open(path, encoding='utf-8') as f:
                yield os.path.relpath(path, DASH_DIR), json.load(f)

    def scoped(self):
        """(rel, dashboard, terms) for every dashboard with the scoped entry."""
        for rel, dashboard in self.dashboards():
            for entry in entries(dashboard, SCOPED_ENTRY):
                yield rel, dashboard, terms(entry.get('expr', ''))

    def test_no_scoped_tags(self):
        bad = []
        for rel, dashboard in self.dashboards():
            for entry in dashboard.get('annotations', {}).get('list', []):
                tags = (entry.get('tags') or []) + ((entry.get('target') or {}).get('tags') or [])
                if any(SCOPED_TAG.search(tag) for tag in tags):
                    bad.append(f'{rel}: "{entry.get("name")}" tags {tags}')
        self.assertEqual(bad, [], 'A Grafana tag matches only one exact value, so on All it hides scoped '
                         f'annotations. Tag "pmm_annotation" only and add "{SCOPED_ENTRY}":\n' + '\n'.join(bad))

    def test_scoped_entry_present(self):
        bad = []
        for rel, dashboard in self.dashboards():
            has_fleet = any(n in first_variables(dashboard) for n in FLEET_VARIABLES)
            scoped = entries(dashboard, SCOPED_ENTRY)
            if rel in GLOBAL_ONLY:
                if scoped:
                    bad.append(f'{rel}: has "{SCOPED_ENTRY}", remove it from GLOBAL_ONLY')
                continue
            if not entries(dashboard, GLOBAL_ENTRY) or not has_fleet:
                if scoped:
                    bad.append(f'{rel}: "{SCOPED_ENTRY}" without "{GLOBAL_ENTRY}" or a fleet variable')
                continue
            if len(scoped) != 1:
                bad.append(f'{rel}: {len(scoped)} "{SCOPED_ENTRY}" entries, want 1')
                continue
            entry = scoped[0]
            if not is_prometheus(entry.get('datasource')):
                bad.append(f'{rel}: datasource {entry.get("datasource")!r} is not a Prometheus datasource')
            want = {'useValueForTime': True, 'titleFormat': 'Annotation {{pmm_annotation_id}}',
                    'textFormat': '{{pmm_annotation_text}}', 'tagKeys': 'pmm_annotation_tags'}
            for key, value in want.items():
                if entry.get(key) != value:
                    bad.append(f'{rel}: {key}={entry.get(key)!r}, want {value!r}')
            if entry.get('expr') != (entry.get('target') or {}).get('expr'):
                bad.append(f'{rel}: expr and target.expr differ')
            try:
                terms(entry.get('expr', ''))
            except ValueError as e:
                bad.append(f'{rel}: {e}')
        for rel in sorted(GLOBAL_ONLY):
            if not os.path.exists(os.path.join(DASH_DIR, rel)):
                bad.append(f'{rel}: file is gone, remove it from GLOBAL_ONLY')
        self.assertEqual(bad, [], '\n'.join(bad))

    def test_matchers_cover_variables(self):
        """Each term filters on exactly the fleet labels the panels filter on, plus its own object.

        A hidden variable no panel uses would only hide annotations: on All without allValue
        it expands to the listed values, which never match a series without that label. The
        service term also leaves out variables derived from the service selection (node_name of
        the picked service would hide the other services), and reads $service_name_list where
        the panels do, since that list carries filters the series lack (MongoDB "set").
        """
        bad = []
        for rel, dashboard, found in self.scoped():
            variables = first_variables(dashboard)
            filters = panel_filters(dashboard)
            for kind, matchers in found.items():
                allowed = SERVICE_LABELS if kind == 'service' else NODE_LABELS
                got = set()
                for label, op, value in matchers:
                    if label == 'service_type' and kind == 'service' and op == '=':
                        continue
                    m = re.fullmatch(r'\$(\w+)', value)
                    if op != '=~' or not m or label not in allowed:
                        bad.append(f'{rel}: {kind} term: unexpected matcher {label}{op}"{value}"')
                        continue
                    name = m.group(1)
                    if name not in variables:
                        bad.append(f'{rel}: {kind} term: ${name} is not a variable')
                    if label_of(name) != label:
                        bad.append(f'{rel}: {kind} term: {label} filtered by ${name}')
                    got.add((label, base(name)))
                if kind == 'service':
                    service = 'service_name' if 'service_name' in variables else 'service'
                    derived = descendants(variables, service)
                    want = {(l, base(n)) for l, n in filters
                            if l in allowed and l != 'service_name' and base(n) not in derived}
                    want.add(('service_name', service))
                    scope = 'service_name_list' if ('service_name', 'service_name_list') in filters else service
                    got_scope = [v for l, _, v in matchers if l == 'service_name']
                    if got_scope != [f'${scope}']:
                        bad.append(f'{rel}: service term filters service_name on {got_scope}, want ${scope}')
                else:
                    want = {(l, base(n)) for l, n in filters if l in allowed}
                    want.add(('node_name', 'node_name'))
                if got != want:
                    bad.append(f'{rel}: {kind} term filters on {sorted(got)}, want {sorted(want)}')
        self.assertEqual(bad, [], '\n'.join(bad))

    def test_node_list_used(self):
        bad = []
        for rel, dashboard, found in self.scoped():
            if 'node' not in found:
                continue
            want = '$node_name_list' if uses_node_list(first_variables(dashboard)) else '$node_name'
            got = [v for l, _, v in found['node'] if l == 'node_name']
            if got != [want]:
                bad.append(f'{rel}: node term filters node_name on {got}, want {want}')
        self.assertEqual(bad, [], 'Node series have no environment label; with node_name on All, '
                         'filter on the hidden node_name_list instead:\n' + '\n'.join(bad))

    def test_type_specific(self):
        typed = {
            'MySQL/MySQL_Instance_Summary.json': 'mysql',
            'MySQL/MySQL_Instances_Overview.json': 'mysql',
            'MySQL/PXC_Galera_Node_Summary.json': 'mysql',
            'PostgreSQL/PostgreSQL_Instance_Summary.json': 'postgresql',
            'PostgreSQL/PostgreSQL_Instances_Compare.json': 'postgresql',
            'MongoDB/MongoDB_Instance_Summary.json': 'mongodb',
            'MongoDB/MongoDB_Instances_Overview.json': 'mongodb',
            'Valkey/Valkey_Overview.json': 'valkey',
            # The service variable mixes metrics of several types, or of an external exporter.
            'Insight/Prometheus_Exporter_Status.json': None,
            'PostgreSQL/PostgreSQL_Patroni_Details.json': None,
            'Query Analytics/pmm-qan.json': None,
        }
        seen, bad = set(), []
        for rel, dashboard, found in self.scoped():
            types = [v for l, _, v in found.get('service', []) if l == 'service_type']
            if len(types) > 1 or not set(types) <= SERVICE_TYPES:
                bad.append(f'{rel}: service_type {types}')
            if rel in typed:
                seen.add(rel)
                want = [typed[rel]] if typed[rel] else []
                if types != want:
                    bad.append(f'{rel}: service_type {types}, want {want}')
        for rel in sorted(set(typed) - seen):
            bad.append(f'{rel}: no "{SCOPED_ENTRY}" entry')
        self.assertEqual(bad, [], '\n'.join(bad))


if __name__ == '__main__':
    unittest.main()
