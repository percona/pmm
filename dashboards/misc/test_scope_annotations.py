#!/usr/bin/env python3
"""Tests for scope-annotations.py and the scope tags in every dashboard.

Run from the repo root:  python3 -m unittest discover -s dashboards/misc -p 'test_*.py' -v
"""

import glob
import importlib.util
import json
import os
import unittest

REPO = os.environ.get('PMM_REPO', os.getcwd())
SCRIPT = os.path.join(REPO, 'dashboards', 'misc', 'scope-annotations.py')
DASH_DIR = os.path.join(REPO, 'dashboards', 'dashboards')


def load_module():
    spec = importlib.util.spec_from_file_location('scope_annotations', SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


sa = load_module()


def variable(name, query, include_all=True, all_value='.+'):
    return {'name': name, 'type': 'query', 'includeAll': include_all, 'allValue': all_value,
            'query': {'query': query, 'refId': 'A'}}


def dashboard(tags, variables, match_any=True):
    return {
        'annotations': {'list': [{
            'name': 'PMM Annotations',
            'tags': tags,
            'target': {'limit': 100, 'matchAny': match_any, 'tags': tags, 'type': 'tags'},
        }]},
        'panels': [{'legendFormat': '$node_name', 'title': '$service_name'}],
        'templating': {'list': variables},
    }


class TestScopeTag(unittest.TestCase):
    def tag(self, label, variables, rel='X.json'):
        return sa.scope_tag(rel, label, {v['name']: v for v in variables})

    def test_appends_own_matcher_and_regex_format(self):
        tag = self.tag('service_name', [
            variable('environment', 'label_values(up, environment)', all_value='.*'),
            variable('node_name', 'label_values(up{environment=~"$environment"}, node_name)'),
            variable('service_name', 'label_values(mysql_up{environment=~"${environment}",node_name=~"$node_name"}, service_name)'),
        ])
        self.assertEqual(tag, 'pmm_scope:service_name:mysql_up{environment=~"${environment:regex}",'
                              'node_name=~"${node_name:regex}",service_name=~"${service_name:regex}"}')

    def test_exact_matcher_becomes_regex(self):
        tag = self.tag('service_name', [
            variable('node_name', 'label_values(up, node_name)', include_all=False, all_value=None),
            variable('service_name', 'label_values({__name__=~"a|b", node_name="$node_name", job!~"$node_name"}, service_name)'),
        ])
        self.assertEqual(tag, 'pmm_scope:service_name:{__name__=~"a|b", node_name=~"${node_name:regex}", '
                              'job!~"${node_name:regex}",service_name=~"${service_name:regex}"}')

    def test_bare_metric_and_variable_selectors(self):
        self.assertEqual(self.tag('node_name', [variable('node_name', 'label_values(pg_up,node_name)')]),
                         'pmm_scope:node_name:pg_up{node_name=~"${node_name:regex}"}')
        self.assertEqual(self.tag('node_name', [
            variable('metric', 'metrics(.+)', include_all=False, all_value=None),
            variable('node_name', 'label_values($metric, node_name)'),
        ]), 'pmm_scope:node_name:{__name__=~"${metric:regex}",node_name=~"${node_name:regex}"}')
        self.assertEqual(self.tag('node_name', [
            variable('metric', 'metrics(.+)', include_all=False, all_value=None),
            variable('node_name', 'label_values($metric{job="x"}, node_name)'),
        ]), 'pmm_scope:node_name:{__name__=~"${metric:regex}",job="x",node_name=~"${node_name:regex}"}')

    def test_rejects(self):
        cases = {
            'query_result': [variable('service_name', 'query_result(up)')],
            'other label': [variable('service_name', 'label_values(up, node_name)')],
            'unknown variable': [variable('service_name', 'label_values(up{env=~"$env"}, service_name)')],
            'backslash': [variable('service_name', 'label_values(up{env=~"a\\\\.b"}, service_name)')],
            # A regex-escaped value compared with != would be compared literally.
            'negative exact matcher': [
                variable('node_name', 'label_values(up, node_name)', include_all=False, all_value=None),
                variable('service_name', 'label_values(up{job!="$node_name"}, service_name)'),
            ],
            # All without an allValue would put the whole list of clusters in the tag.
            'parent All without allValue': [
                variable('cluster', 'label_values(up, cluster)', all_value=None),
                variable('service_name', 'label_values(up{cluster=~"$cluster"}, service_name)'),
            ],
            'own All without allValue': [variable('service_name', 'label_values(up, service_name)', all_value=None)],
            'stray dollar': [variable('service_name', 'label_values(up{job=~"a$"}, service_name)')],
        }
        for name, variables in cases.items():
            with self.subTest(name), self.assertRaises(sa.ScopeError):
                self.tag('service_name', variables)
        with self.assertRaises(sa.ScopeError):
            self.tag('environment', [variable('environment', 'label_values(up, environment)')])


class TestRewrite(unittest.TestCase):
    VARIABLES = [
        variable('node_name', 'label_values(up, node_name)'),
        variable('service_name', 'label_values(up{node_name=~"$node_name"}, service_name)'),
        variable('picked', 'label_values(up, service_name)', all_value=None),
    ]

    def test_changes_only_annotation_tags(self):
        d = dashboard(['pmm_annotation', '$node_name', '$service_name', '$picked'], self.VARIABLES)
        text = json.dumps(d, indent=4, sort_keys=True)
        out = json.loads(sa.rewrite('X.json', text))

        tags = out['annotations']['list'][0]['target']['tags']
        self.assertEqual(tags, ['pmm_annotation', 'pmm_scope:node_name:up{node_name=~"${node_name:regex}"}',
                                'pmm_scope:service_name:up{node_name=~"${node_name:regex}",'
                                'service_name=~"${service_name:regex}"}', '$picked'])
        self.assertEqual(out['annotations']['list'][0]['tags'], tags)
        self.assertEqual(out['panels'], d['panels'])
        self.assertEqual(sa.rewrite('X.json', sa.rewrite('X.json', text)), sa.rewrite('X.json', text))

    def test_rebuilds_stale_scope_tags(self):
        d = dashboard(['pmm_annotation', 'pmm_scope:node_name:old{}'], self.VARIABLES)
        out = json.loads(sa.rewrite('X.json', json.dumps(d)))
        self.assertEqual(out['annotations']['list'][0]['target']['tags'][1],
                         'pmm_scope:node_name:up{node_name=~"${node_name:regex}"}')

    def test_requires_match_any(self):
        with self.assertRaises(sa.ScopeError):
            sa.rewrite('X.json', json.dumps(dashboard(['$node_name'], self.VARIABLES, match_any=False)))

    def test_exempt_dashboard_is_left_alone(self):
        text = json.dumps(dashboard(['$node_name'], self.VARIABLES, match_any=False))
        self.assertEqual(sa.rewrite(next(iter(sa.EXEMPT)), text), text)


class TestDashboards(unittest.TestCase):
    def dashboards(self):
        paths = sorted(glob.glob(os.path.join(DASH_DIR, '**', '*.json'), recursive=True))
        # Run from the wrong directory, every check here would pass on nothing.
        self.assertTrue(paths, f'no dashboards found under {DASH_DIR}')
        return paths

    def test_annotation_tags_are_scoped(self):
        bad = []
        for path in self.dashboards():
            rel = os.path.relpath(path, DASH_DIR)
            with open(path, encoding='utf-8') as f:
                text = f.read()
            try:
                if sa.rewrite(rel, text) != text:
                    bad.append(f'{rel}: tags need updating')
            except sa.ScopeError as e:
                bad.append(f'{rel}: {e}')
        self.assertEqual(bad, [], 'Run dashboards/misc/scope-annotations.py on:\n' + '\n'.join(bad))

    def test_overrides_and_exemptions_still_apply(self):
        stale = []
        for rel, name in sorted(sa.OVERRIDES):
            path = os.path.join(DASH_DIR, rel)
            if not os.path.exists(path):
                stale.append(f'{rel}: file is gone')
                continue
            with open(path, encoding='utf-8') as f:
                var = sa.first_variables(json.load(f)).get(name)
            query = (var or {}).get('query')
            query = query.get('query') if isinstance(query, dict) else query
            if not var or sa.LABEL_VALUES.fullmatch(query or ''):
                stale.append(f'{rel}: {name} needs no override')
        for rel in sorted(sa.EXEMPT):
            if not os.path.exists(os.path.join(DASH_DIR, rel)):
                stale.append(f'{rel}: file is gone')
        self.assertEqual(stale, [], 'Remove stale OVERRIDES or EXEMPT entries:\n' + '\n'.join(stale))


if __name__ == '__main__':
    unittest.main()
