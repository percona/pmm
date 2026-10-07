#!/usr/bin/env python3
"""Fleet-sized "All" variables must not expand to every value (PMM-15609).

Without allValue, Grafana sends "All" as a regex of every node or service name,
and at a few thousand services the request is too large for nginx.

Run from the repo root:  python3 -m unittest discover -s dashboards/misc -p 'test_*.py' -v
"""

import glob
import json
import os
import re
import unittest

REPO = os.environ.get('PMM_REPO', os.getcwd())
DASH_DIR = os.path.join(REPO, 'dashboards', 'dashboards')

# Node_Temperature_Details names its hidden service variable "service".
FLEET_VARIABLES = ('node_name', 'service_name', 'service')

# Lists already narrowed by a single-select parent, so they stay small.
BOUNDED = {
    ('Insight/Prometheus_Exporter_Status.json', 'service_name'),
    ('Insight/Prometheus_Exporters_Overview.json', 'service_name'),
    ('MongoDB/MongoDB_Cluster_Summary.json', 'node_name'),
    ('MongoDB/MongoDB_Cluster_Summary.json', 'service_name'),
    ('MongoDB/MongoDB_InMemory_Details.json', 'node_name'),
    ('MongoDB/MongoDB_MMAPv1_Details.json', 'node_name'),
    ('MongoDB/MongoDB_Router_Summary.json', 'node_name'),
    ('MongoDB/MongoDB_Router_Summary.json', 'service_name'),
    ('MySQL/MySQL_Group_Replication_Summary.json', 'service_name'),
    ('MySQL/MySQL_Instances_Compare.json', 'node_name'),
    ('OS/CPU_Utilization_Details.json', 'service_name'),
    ('OS/Memory_Details.json', 'service_name'),
}


# allValue ".*" predates PMM-15609; these panels read the variable directly, as they did before.
PREDATING_ALL_VALUE = {
    ('MongoDB/MongoDB_Unused_Indexes.json', 'node_name'),
    ('MySQL/MySQL_Replication_Summary.json', 'service_name'),
    ('MySQL/PXC_Galera_Nodes_Compare.json', 'service_name'),
    ('PostgreSQL/PostgreSQL_Instances_Overview.json', 'node_name'),
    ('PostgreSQL/PostgreSQL_Replication_Overview.json', 'node_name'),
    ('PostgreSQL/PostgreSQL_Top_Queries.json', 'node_name'),
}

def first_variables(dashboard):
    """Grafana resolves a duplicated variable name to its first definition."""
    seen = {}
    for var in dashboard.get('templating', {}).get('list', []):
        seen.setdefault(var.get('name'), var)
    return seen


class TestFleetVariablesHaveAllValue(unittest.TestCase):
    def dashboards(self):
        paths = sorted(glob.glob(os.path.join(DASH_DIR, '**', '*.json'), recursive=True))
        # Run from the wrong directory, every check here would pass on nothing.
        self.assertTrue(paths, f'no dashboards found under {DASH_DIR}')
        return paths

    def test_fleet_variables_have_all_value(self):
        missing = []
        for path in self.dashboards():
            rel = os.path.relpath(path, DASH_DIR)
            with open(path, encoding='utf-8') as f:
                variables = first_variables(json.load(f))
            for name in FLEET_VARIABLES:
                var = variables.get(name)
                if not var or not var.get('includeAll') or (rel, name) in BOUNDED:
                    continue
                if not var.get('allValue'):
                    missing.append(f'{rel}: {name} allValue={var.get("allValue")!r}')
        self.assertEqual(missing, [], 'Set allValue ".+" and point panel queries at a hidden '
                         '<var>_list (see test_panels_use_list_variable):\n' + '\n'.join(missing))

    def test_bounded_entries_still_exist(self):
        stale = []
        for rel, name in sorted(BOUNDED):
            path = os.path.join(DASH_DIR, rel)
            if not os.path.exists(path):
                stale.append(f'{rel}: file is gone')
                continue
            with open(path, encoding='utf-8') as f:
                var = first_variables(json.load(f)).get(name)
            if not var or not var.get('includeAll'):
                stale.append(f'{rel}: {name} no longer offers All')
        self.assertEqual(stale, [], 'Remove stale BOUNDED entries:\n' + '\n'.join(stale))

    def test_panels_use_list_variable(self):
        """Panels read <var>_list, not a fleet variable whose All is ".+".

        ".+" keeps variable lookups and annotation URLs short, but it drops the parent filters
        (environment, cluster, ...) that the expanded list carried. The hidden <var>_list is the
        variable's own query narrowed by $<var>, so its All expands to exactly that list inside
        the panel's POST body. Panels that repeat on <var> keep $<var>: it holds one value there.
        """
        bad = []
        for path in self.dashboards():
            rel = os.path.relpath(path, DASH_DIR)
            with open(path, encoding='utf-8') as f:
                dashboard = json.load(f)
            variables = first_variables(dashboard)
            custom = [n for n in FLEET_VARIABLES if (variables.get(n) or {}).get('allValue')
                      and (rel, n) not in PREDATING_ALL_VALUE]
            for repeat, text in target_texts(dashboard.get('panels', [])):
                for name in custom:
                    if repeat != name and re.search(VAR_REF.format(name=re.escape(name)), text):
                        bad.append(f'{rel}: panel query uses ${name}, use ${name}_list')
            for name in FLEET_VARIABLES:
                lst = variables.get(f'{name}_list')
                if not lst:
                    continue
                query = lst.get('query') if isinstance(lst.get('query'), str) else (lst.get('query') or {}).get('query', '')
                if (lst.get('allValue') or not lst.get('includeAll') or lst.get('hide') != 2
                        or not re.search(VAR_REF.format(name=re.escape(name)), query)):
                    bad.append(f'{rel}: {name}_list must be hidden, include All, have no allValue '
                               f'and filter on ${name}')
        self.assertEqual(sorted(set(bad)), [], '\n'.join(sorted(set(bad))))


VAR_REF = r'\$(?:\{{{name}(?::\w+)?\}}|{name}(?![A-Za-z0-9_]))'


def target_texts(panels, repeat=None):
    """Yield (repeat variable, query text) for every PromQL or SQL panel target."""
    for panel in panels:
        rep = panel.get('repeat') or repeat
        yield from target_texts(panel.get('panels', []), rep)
        for target in panel.get('targets', []):
            for key in ('expr', 'rawSql'):
                if isinstance(target.get(key), str):
                    yield rep, target[key]


if __name__ == '__main__':
    unittest.main()
