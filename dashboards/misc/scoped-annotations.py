#!/usr/bin/env python3
"""Read node- and service-scoped annotations from VictoriaMetrics instead of Grafana tags (PMM-15663).

For each dashboard whose "PMM Annotations" entry is tagged $node_name, $service_name or $service:
- "PMM Annotations" keeps only the "pmm_annotation" tag (global annotations);
- "PMM Annotations: nodes and services" reads the pmm_annotation_event series with the label
  filters the dashboard's panels use, except variables derived from the service selection;
- a hidden node_name_list is added where node series need it to follow the environment filter.

Idempotent: a dashboard rewritten earlier keeps its scope through its existing scoped entry.

Usage, from the repo root:  python3 dashboards/misc/scoped-annotations.py [dashboard.json ...]
Without arguments it rewrites every dashboard under dashboards/dashboards/. Check the files it
reports with cleanup-dash.py --check-only.
"""

import argparse
import collections
import glob
import json
import os
import re
import sys

DASH_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'dashboards')

GLOBAL_ENTRY = 'PMM Annotations'
SCOPED_ENTRY = 'PMM Annotations: nodes and services'
GLOBAL_TAG = 'pmm_annotation'

# Labels in matcher order. Service series carry node and service labels, node series only node labels
# (node_name is matched last, on $node_name or $node_name_list).
SERVICE_LABELS = ('environment', 'cluster', 'replication_set', 'region', 'az', 'node_type', 'node_name',
                  'service_name')
NODE_LABELS = ('region', 'az', 'node_type')

# Metric name prefixes of the service variable's query, by service_type.
TYPE_PREFIXES = (
    ('mysql_', 'mysql'),
    ('pg_', 'postgresql'),
    ('postgresql_', 'postgresql'),
    ('mongodb_', 'mongodb'),
    ('redis_', 'valkey'),
    ('valkey_', 'valkey'),
    ('proxysql_', 'proxysql'),
    ('haproxy_', 'haproxy'),
)

TERM = ('max by (pmm_annotation_id, pmm_annotation_text, pmm_annotation_tags) '
        '(last_over_time(pmm_annotation_event{{{matchers}}}[$__interval] offset -$__interval))')

# A $var reference, plain or ${var:format}.
VAR_REF = r'\$(?:\{{{name}(?::\w+)?\}}|{name}(?![A-Za-z0-9_]))'
MATCHER = re.compile(r'(\w+)(=~|!=|=)"([^"]*)"')


class Unfit(Exception):
    """The dashboard does not fit a shape this tool knows."""


def first_variables(dashboard):
    """Grafana resolves a duplicated variable name to its first definition."""
    seen = {}
    for var in dashboard.get('templating', {}).get('list', []):
        seen.setdefault(var.get('name'), var)
    return seen


def var_query(var):
    query = var.get('query')
    return query.get('query', '') if isinstance(query, dict) else query or ''


def walk_panels(panels):
    for panel in panels:
        yield from walk_panels(panel.get('panels', []))
        yield panel


def prometheus_targets(panel):
    return [t for t in panel.get('targets', None) or [] if isinstance(t.get('expr'), str)]


def panel_filters(dashboard, variables):
    """Fleet labels the panels filter on: label -> variable names, from label="$var" or label=~"$var"."""
    found = collections.defaultdict(set)
    for panel in walk_panels(dashboard.get('panels', [])):
        for target in prometheus_targets(panel):
            for label, op, value in MATCHER.findall(target['expr']):
                if op == '!=' or label not in SERVICE_LABELS:
                    continue
                names = (label, 'service', 'service_name_list') if label == 'service_name' else (label,)
                for name in names:
                    if name in variables and re.search(VAR_REF.format(name=name), value):
                        found[label].add(name)
    return found


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


def datasource(dashboard):
    """The Prometheus datasource reference the panels use most, "Metrics" if they all use the default."""
    refs = collections.Counter()
    for panel in walk_panels(dashboard.get('panels', [])):
        targets = prometheus_targets(panel)
        if not targets:
            continue
        for ds in [panel.get('datasource')] + [t.get('datasource') for t in targets if 'datasource' in t]:
            if isinstance(ds, str) and ds and not ds.startswith('-- '):
                refs[json.dumps(ds)] += 1
            elif isinstance(ds, dict) and ds.get('uid') and ds.get('type', 'prometheus') == 'prometheus':
                refs[json.dumps(ds, sort_keys=True)] += 1
    return json.loads(refs.most_common(1)[0][0]) if refs else 'Metrics'


def service_type(var):
    """service_type of the metrics the service variable lists, None if they mix types."""
    if var.get('type') != 'query':
        return None
    query = var_query(var)
    names = []
    for _, value in re.findall(r'__name__\s*(=~|=)\s*"([^"]*)"', query):
        names += value.split('|')
    body = re.sub(r'\{[^}]*\}', '', query)
    body = re.sub(r'\b(?:on|ignoring|by|without|group_left|group_right)\s*\([^)]*\)', '', body)
    m = re.fullmatch(r'\s*label_values\((.*),\s*\w+\s*\)\s*', body)
    if m:
        body = m.group(1)
    else:
        body = re.sub(r'^\s*query_result\(', '', body)
    names += re.findall(r'\b[a-zA-Z_:][a-zA-Z0-9_:]*\b(?!\s*\()', body)
    types = set()
    for name in names:
        types.add(next((t for prefix, t in TYPE_PREFIXES if name.startswith(prefix)), None))
    return types.pop() if len(types) == 1 else None


def uses_node_list(variables):
    """On All, node_name ".+" drops the environment filter that node series can't carry."""
    node = variables.get('node_name') or {}
    return bool(node.get('includeAll') and node.get('allValue') and 'environment' in variables)


def node_list_variable(node):
    """A hidden node_name_list like PMM-15609's: node_name's query narrowed by $node_name, without allValue."""
    m = re.fullmatch(r'label_values\((.*?)(,\s*node_name\s*\))', var_query(node), re.S)
    if not m:
        raise Unfit(f'node_name query is not label_values(..., node_name): {var_query(node)}')
    selector, tail = m.groups()
    if selector.rstrip().endswith('}'):
        head = selector.rstrip()[:-1]
        sep = '' if head.rstrip().endswith('{') else ','
        selector = f'{head}{sep}node_name=~"$node_name"}}'
    else:
        selector = f'{selector}{{node_name=~"$node_name"}}'
    query = f'label_values({selector}{tail}'
    var = {
        'current': {'selected': True, 'text': ['All'], 'value': ['$__all']},
        'definition': query,
        'description': 'Every Node Name that matches the dashboard filters, expanded. Used where a query cannot '
                       f'filter on the parent labels itself; node_name uses allValue "{node["allValue"]}". '
                       'See PMM-15609.',
        'hide': 2,
        'includeAll': True,
        'label': 'Node Name list',
        'multi': True,
        'name': 'node_name_list',
        'options': [],
        'query': {'qryType': 1, 'query': query, 'refId': 'PrometheusVariableQueryEditor-VariableQuery'},
        'refresh': 2,
        'regex': node.get('regex', ''),
        'skipUrlSync': True,
        'sort': node.get('sort', 1),
        'type': 'query',
    }
    if 'datasource' in node:
        var['datasource'] = node['datasource']
    return var


def scopes(entry, scoped):
    """Which terms the dashboard needs: from today's tags, or from an earlier rewrite."""
    tags = (entry.get('tags') or []) + ((entry.get('target') or {}).get('tags') or [])
    found = set()
    for tag in tags:
        if re.fullmatch(VAR_REF.format(name='node_name'), tag):
            found.add('node')
        if re.fullmatch(VAR_REF.format(name='service_name') + '|' + VAR_REF.format(name='service'), tag):
            found.add('service')
    if not found and scoped:
        expr = scoped.get('expr', '')
        if 'service_name=""' in expr:
            found.add('node')
        if 'service_name!=""' in expr:
            found.add('service')
    return found


def term(matchers):
    return TERM.format(matchers=','.join(matchers))


def rewrite(dashboard):
    """Rewrite the dashboard in place; return a short description, or None if it has no scoped annotations."""
    annotations = dashboard.get('annotations', {}).get('list', [])
    current = [a for a in annotations if a.get('name') == GLOBAL_ENTRY]
    scoped = [a for a in annotations if a.get('name') == SCOPED_ENTRY]
    if len(current) != 1 or len(scoped) > 1:
        if scoped or any('$' in json.dumps([a.get('tags'), (a.get('target') or {}).get('tags')]) for a in current):
            raise Unfit(f'{len(current)} "{GLOBAL_ENTRY}" and {len(scoped)} "{SCOPED_ENTRY}" entries')
        return None
    entry = current[0]
    found = scopes(entry, scoped[0] if scoped else None)
    if not found:
        return None

    variables = first_variables(dashboard)
    filters = panel_filters(dashboard, variables)
    parts = []

    if 'node' in found:
        node = variables.get('node_name')
        if not node:
            raise Unfit('node annotations without a node_name variable')
        matchers = ['service_name=""']
        matchers += [f'{label}=~"${label}"' for label in NODE_LABELS if label in filters]
        if uses_node_list(variables):
            if 'node_name_list' not in variables:
                lst = dashboard['templating']['list']
                lst.insert(lst.index(node) + 1, node_list_variable(node))
            matchers.append('node_name=~"$node_name_list"')
        else:
            matchers.append('node_name=~"$node_name"')
        parts.append(term(matchers))

    if 'service' in found:
        service = 'service_name' if 'service_name' in variables else 'service' if 'service' in variables else None
        if not service:
            raise Unfit('service annotations without a service_name or service variable')
        # A variable derived from the service selection (node_name of the picked service) would hide
        # the other services' annotations; $service_name_list carries filters the series lack (set).
        derived = descendants(variables, service)
        matchers = ['service_name!=""']
        for label in SERVICE_LABELS:
            if label == 'service_name':
                name = 'service_name_list' if 'service_name_list' in filters[label] else service
            elif label in filters and label not in derived:
                name = label
            else:
                continue
            matchers.append(f'{label}=~"${name}"')
        stype = service_type(variables[service])
        if stype:
            matchers.append(f'service_type="{stype}"')
        parts.append(term(matchers))

    expr = ' or '.join(parts)
    if 'tags' in entry:
        entry['tags'] = [GLOBAL_TAG]
    if isinstance(entry.get('target'), dict) and 'tags' in entry['target']:
        entry['target']['tags'] = [GLOBAL_TAG]
    new = {
        'datasource': datasource(dashboard),
        'enable': entry.get('enable', True),
        'expr': expr,
        'hide': entry.get('hide', False),
        'iconColor': entry.get('iconColor', '#e0752d'),
        'name': SCOPED_ENTRY,
        'tagKeys': 'pmm_annotation_tags',
        'target': {'expr': expr, 'interval': '', 'refId': 'Anno'},
        'textFormat': '{{pmm_annotation_text}}',
        'titleFormat': 'Annotation {{pmm_annotation_id}}',
        'useValueForTime': True,
    }
    if scoped:
        annotations.remove(scoped[0])
    annotations.insert(annotations.index(entry) + 1, new)
    return ' + '.join(sorted(found))


def dump(dashboard, raw):
    """cleanup-dash.py's format, keeping the file's indentation so a 2-space file isn't reindented whole."""
    m = re.match(r'\{\n( +)"', raw)
    indent = len(m.group(1)) if m else 4
    return json.dumps(dashboard, sort_keys=True, indent=indent, separators=(',', ': '), ensure_ascii=False) + '\n'


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('files', nargs='*', help='dashboards to rewrite (default: all)')
    args = parser.parse_args()
    files = args.files or sorted(glob.glob(os.path.join(DASH_DIR, '**', '*.json'), recursive=True))

    failed = False
    for path in files:
        with open(path, encoding='utf-8') as f:
            raw = f.read()
        dashboard = json.loads(raw)
        try:
            scope = rewrite(dashboard)
        except Unfit as e:
            print(f'SKIP {path}: {e}', file=sys.stderr)
            failed = True
            continue
        if scope is None:
            continue
        out = dump(dashboard, raw)
        if json.loads(out) == json.loads(raw):
            continue
        with open(path, 'w', encoding='utf-8') as f:
            f.write(out)
        print(f'{scope}: {path}')
    return 1 if failed else 0


if __name__ == '__main__':
    sys.exit(main())
