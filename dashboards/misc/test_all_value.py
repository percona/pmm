#!/usr/bin/env python3
"""Fleet-sized "All" variables must not expand to every value (PMM-15609).

Without allValue, Grafana sends "All" as a regex of every node or service name,
and at a few thousand services the request is too large for nginx.

Run from the repo root:  python3 -m unittest discover -s dashboards/misc -p 'test_*.py' -v
"""

import glob
import json
import os
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


def first_variables(dashboard):
    """Grafana resolves a duplicated variable name to its first definition."""
    seen = {}
    for var in dashboard.get('templating', {}).get('list', []):
        seen.setdefault(var.get('name'), var)
    return seen


class TestFleetVariablesHaveAllValue(unittest.TestCase):
    def test_fleet_variables_have_all_value(self):
        missing = []
        for path in sorted(glob.glob(os.path.join(DASH_DIR, '**', '*.json'), recursive=True)):
            rel = os.path.relpath(path, DASH_DIR)
            with open(path, encoding='utf-8') as f:
                variables = first_variables(json.load(f))
            for name in FLEET_VARIABLES:
                var = variables.get(name)
                if not var or not var.get('includeAll') or (rel, name) in BOUNDED:
                    continue
                if not var.get('allValue'):
                    missing.append(f'{rel}: {name} allValue={var.get("allValue")!r}')
        self.assertEqual(missing, [], 'Set allValue ".+" and add the variable\'s parent '
                         'filters to every query that uses it:\n' + '\n'.join(missing))

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


if __name__ == '__main__':
    unittest.main()
