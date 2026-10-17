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

# A $var reference in a query, plain or ${var:format}.
VAR_REF = r'\$(?:\{{{name}(?::\w+)?\}}|{name}(?![A-Za-z0-9_]))'

# ClickHouse SQL must filter on the LBAC-filtered list of these, not drop the filter on All.
SERVICE_VARIABLES = {'service_name', 'service'}

# Lists already narrowed by a single-select parent, so they stay small. A node_name parent doesn't
# count: every remote service sits on the PMM Server node.
BOUNDED = {
    ('MongoDB/MongoDB_Cluster_Summary.json', 'node_name'),
    ('MongoDB/MongoDB_Cluster_Summary.json', 'service_name'),
    ('MongoDB/MongoDB_InMemory_Details.json', 'node_name'),
    ('MongoDB/MongoDB_MMAPv1_Details.json', 'node_name'),
    ('MongoDB/MongoDB_Router_Summary.json', 'node_name'),
    ('MongoDB/MongoDB_Router_Summary.json', 'service_name'),
    # Group Replication members only; this hidden variable is always All and feeds the annotation tag.
    ('MySQL/MySQL_Group_Replication_Summary.json', 'service_name'),
    ('MySQL/MySQL_Instances_Compare.json', 'node_name'),
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
                if var.get('allValue') not in ('.+', '.*'):
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

    def test_sql_uses_conditional_all(self):
        """Grafana substitutes a custom allValue raw, so "IN (.+)" is a SQL error.

        Wrap the filter as $__conditionalAll(col IN (${var:singlequote}), $var):
        the ClickHouse plugin drops it on All, and singlequote escapes quotes.
        Service variables are the exception, see test_list_variables.
        """
        bad = []
        for path in self.dashboards():
            rel = os.path.relpath(path, DASH_DIR)
            with open(path, encoding='utf-8') as f:
                dashboard = json.load(f)
            custom = {n for n, v in first_variables(dashboard).items() if v.get('allValue')}
            for sql in raw_sql(dashboard.get('panels', [])):
                for name in sorted(custom - SERVICE_VARIABLES):
                    if re.search(VAR_REF.format(name=re.escape(name)), strip_conditional_all(sql, name)):
                        bad.append(f'{rel}: ${name}')
        self.assertEqual(sorted(set(bad)), [], 'Wrap these in $__conditionalAll(col IN '
                         '(${var:singlequote}), $var):\n' + '\n'.join(sorted(set(bad))))

    def test_list_variables(self):
        """Where a query can't filter on the parent labels, it reads the hidden <var>_list.

        Node metrics have no environment label, and ClickHouse has no labels at all. There the
        ".+" of a fleet variable would ignore the parent filters, and a join on the variable's
        selector rescans it per query. <var>_list is that selector narrowed by $<var>, with no
        allValue, so All expands to the scoped list in the POST body. For ClickHouse it also
        keeps the service list LBAC-filtered, which $__conditionalAll would drop on All.
        """
        bad = []
        for path in self.dashboards():
            rel = os.path.relpath(path, DASH_DIR)
            with open(path, encoding='utf-8') as f:
                dashboard = json.load(f)
            variables = first_variables(dashboard)
            fleet = [n for n in FLEET_VARIABLES if (variables.get(n) or {}).get('allValue')]
            for expr in exprs(dashboard.get('panels', [])):
                for name in fleet:
                    if re.search(rf'and\s+on\s*\(\s*{name}\s*\)\s*group\s+by\s*\(\s*{name}\s*\)', expr):
                        bad.append(f'{rel}: join on {name}, use ${name}_list')
            for sql in raw_sql(dashboard.get('panels', [])):
                for name in sorted(set(fleet) & SERVICE_VARIABLES):
                    if re.search(VAR_REF.format(name=re.escape(name)), sql):
                        bad.append(f'{rel}: SQL uses ${name}, use ${name}_list')
            text = json.dumps(dashboard.get('panels', []))
            for name in FLEET_VARIABLES:
                lst = variables.get(f'{name}_list')
                if not lst:
                    continue
                query = lst.get('query') if isinstance(lst.get('query'), str) else (lst.get('query') or {}).get('query', '')
                if (lst.get('allValue') or not lst.get('includeAll') or lst.get('hide') != 2
                        or not re.search(VAR_REF.format(name=re.escape(name)), query)):
                    bad.append(f'{rel}: {name}_list must be hidden, include All, have no allValue '
                               f'and filter on ${name}')
                if not re.search(VAR_REF.format(name=re.escape(f'{name}_list')), text):
                    bad.append(f'{rel}: {name}_list is unused')
        self.assertEqual(sorted(set(bad)), [], '\n'.join(sorted(set(bad))))


def exprs(panels):
    for panel in panels:
        yield from exprs(panel.get('panels', []))
        for target in panel.get('targets', []):
            if isinstance(target.get('expr'), str):
                yield target['expr']


def raw_sql(panels):
    for panel in panels:
        yield from raw_sql(panel.get('panels', []))
        for target in panel.get('targets', []):
            if target.get('rawSql'):
                yield target['rawSql']


def strip_conditional_all(sql, name):
    """Remove each $__conditionalAll(...) whose last argument is $name."""
    macro = '$__conditionalAll('
    out, i = [], 0
    while (start := sql.find(macro, i)) != -1:
        depth, end = 0, start + len(macro) - 1
        for end in range(end, len(sql)):
            depth += {'(': 1, ')': -1}.get(sql[end], 0)
            if depth == 0:
                break
        call = sql[start:end + 1]
        last_arg = call[len(macro):-1].rsplit(',', 1)[-1].strip()
        out.append(sql[i:start])
        if last_arg not in (f'${name}', f'${{{name}}}'):
            out.append(call)
        i = end + 1
    out.append(sql[i:])
    return ''.join(out)


if __name__ == '__main__':
    unittest.main()
