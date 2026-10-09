#!/usr/bin/env python3
"""Build main/pr/E/K variants of the 3 benchmark dashboards into dash/out/<variant>/<file>."""
import copy, json, os
D = os.path.dirname(os.path.abspath(__file__)) + '/dash'
CH = {'type': 'grafana-clickhouse-datasource', 'uid': 'PDEE91DDB90597936'}
PROM = {'type': 'prometheus', 'uid': 'PA58DA793C7250F1B'}
GLOBAL = {'builtIn': 1, 'datasource': {'type': 'datasource', 'uid': 'grafana'}, 'enable': True, 'hide': False, 'iconColor': '#e0752d',
          'limit': 100, 'matchAny': True, 'name': 'PMM Annotations', 'tags': ['pmm_annotation'],
          'target': {'limit': 100, 'matchAny': True, 'tags': ['pmm_annotation'], 'type': 'tags'}, 'type': 'tags'}
# label selector (E) and SQL condition (K) per dashboard
def m(col, var):
    return f"match({col}, '^(${{{var}:regex}})$')"
def ma(col, var):
    return f"arrayExists(x -> match(x, '^(${{{var}:regex}})$'), {col})"
NODE_K = m('environment', 'environment') + ' AND ' + m('node_name', 'node_name')
# label selector (E) and SQL condition (K) per dashboard
SCOPE = {
    'Home_Dashboard.json': ('environment=~"$environment",node_name=~"$node_name"', NODE_K),
    'Nodes_Overview.json': ('environment=~"$environment",node_name=~"$node_name"', NODE_K),
    'MySQL_Instances_Overview.json': (
        'environment=~"$environment",cluster=~"$cluster",replication_set=~"$replication_set",node_name=~"$node_name",'
        'service_name=~"$service_name",service_type="mysql"',
        "has(service_types, 'mysql') AND " + ' AND '.join([m('environment', 'environment'), m('cluster', 'cluster'),
            ma('replication_sets', 'replication_set'), m('node_name', 'node_name'), ma('service_names', 'service_name')])),
}
WINDOW = os.environ.get('E_WINDOW', '$__interval] offset -$__interval')

def e_anno(sel):
    expr = f'max by (annotation_id, text, node_name) (last_over_time(pmm_annotation_event{{{sel}}}[{WINDOW}))'
    return {'datasource': PROM, 'enable': True, 'hide': False, 'iconColor': '#e0752d', 'name': 'PMM Scoped Annotations',
            'expr': expr, 'useValueForTime': True, 'textFormat': '{{text}}', 'titleFormat': '', 'tagKeys': 'node_name',
            'target': {'refId': 'Anno', 'expr': expr, 'interval': ''}}

def k_anno(cond):
    sql = ('SELECT time, text, arrayConcat(tags, [node_name], service_names) AS tags FROM pmm.annotations '
           f'WHERE $__timeFilter(time) AND {cond} ORDER BY time')
    return {'datasource': CH, 'enable': True, 'hide': False, 'iconColor': '#e0752d', 'name': 'PMM Scoped Annotations',
            'target': {'refId': 'Anno', 'datasource': CH, 'editorType': 'sql', 'format': 1, 'queryType': 'table', 'rawSql': sql}}

for f, (sel, cond) in SCOPE.items():
    for variant in ('main', 'pr', 'E', 'K'):
        d = json.load(open(f'{D}/{"main" if variant == "main" else "pr"}/{f}'))
        lst = d['annotations']['list']
        rest = [a for a in lst if a.get('name') != 'PMM Annotations']
        if variant == 'E':
            d['annotations']['list'] = [GLOBAL, e_anno(sel)] + rest
        elif variant == 'K':
            d['annotations']['list'] = [GLOBAL, k_anno(cond)] + rest
        d['uid'] = f'{variant}-{d["uid"]}'[:40]
        d['title'] = f'[{variant}] {d["title"]}'
        d.pop('id', None)
        os.makedirs(f'{D}/out/{variant}', exist_ok=True)
        json.dump(d, open(f'{D}/out/{variant}/{f}', 'w'))
        print(variant, f, d['uid'])
