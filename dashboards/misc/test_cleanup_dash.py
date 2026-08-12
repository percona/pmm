#!/usr/bin/env python3
"""Tests for dashboards/misc/cleanup-dash.py (PMM-15308).

Run from the repo root:  python3 -m unittest discover -s <dir> -p 'test_*.py' -v
"""

import copy
import importlib.util
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

REPO = os.environ.get('PMM_REPO', os.getcwd())
SCRIPT = os.path.join(REPO, 'dashboards', 'misc', 'cleanup-dash.py')
DASH_DIR = os.path.join(REPO, 'dashboards', 'dashboards')


def load_module():
    spec = importlib.util.spec_from_file_location('cleanup_dash', SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


cd = load_module()


def run_cli(*args):
    return subprocess.run([sys.executable, SCRIPT, *args],
                          capture_output=True, text=True)


MINIMAL = {
    'editable': True,
    'refresh': '1m',
    'timezone': 'browser',
    'time': {'from': 'now-6h', 'to': 'now'},
    'id': 42,
    'title': 'Dash',
    'panels': [],
}


class TestTrimTitles(unittest.TestCase):
    def test_trims_root_nested_and_templating(self):
        d = {
            'title': '  Root  ',
            'panels': [
                {'title': 'Panel A ', 'panels': [{'title': ' Nested B '}]},
            ],
            'templating': {'list': [{'title': ' Var '}]},
        }
        out = cd.trim_titles(copy.deepcopy(d))
        self.assertEqual(out['title'], 'Root')
        self.assertEqual(out['panels'][0]['title'], 'Panel A')
        self.assertEqual(out['panels'][0]['panels'][0]['title'], 'Nested B')
        self.assertEqual(out['templating']['list'][0]['title'], 'Var')

    def test_whitespace_only_title_becomes_empty(self):
        # PMM-15308: MySQL_User_Details spacer panel, title is literally " ".
        out = cd.trim_titles({'panels': [{'title': ' ', 'type': 'text'}]})
        self.assertEqual(out['panels'][0]['title'], '')

    def test_strips_tabs_and_newlines(self):
        out = cd.trim_titles({'title': '\tSpaced\n'})
        self.assertEqual(out['title'], 'Spaced')

    def test_non_string_titles_untouched(self):
        # Must not raise: a null title is valid JSON.
        out = cd.trim_titles({'panels': [{'title': None}, {'title': 7}]})
        self.assertIsNone(out['panels'][0]['title'])
        self.assertEqual(out['panels'][1]['title'], 7)

    def test_nested_dict_under_title_key_is_traversed(self):
        out = cd.trim_titles({'title': {'title': ' inner '}})
        self.assertEqual(out['title']['title'], 'inner')

    def test_non_title_keys_untouched(self):
        out = cd.trim_titles({'legendFormat': '  keep me  ', 'expr': ' x '})
        self.assertEqual(out['legendFormat'], '  keep me  ')
        self.assertEqual(out['expr'], ' x ')

    def test_idempotent(self):
        d = {'title': ' A ', 'panels': [{'title': ' B '}]}
        once = cd.trim_titles(copy.deepcopy(d))
        twice = cd.trim_titles(copy.deepcopy(once))
        self.assertEqual(once, twice)

    def test_unicode_preserved(self):
        out = cd.trim_titles({'title': ' Répliqué ☠ '})
        self.assertEqual(out['title'], 'Répliqué ☠')

    def test_returns_dashboard_like_other_cleanupers(self):
        d = {'title': ' A '}
        self.assertIs(cd.trim_titles(d), d)


class TestCollectTitles(unittest.TestCase):
    def test_collects_in_traversal_order_without_mutating(self):
        d = {'title': 'root', 'panels': [{'title': 'a'}, {'title': 'b'}]}
        before = copy.deepcopy(d)
        self.assertEqual(cd.collect_titles(d), ['root', 'a', 'b'])
        self.assertEqual(d, before, 'collect_titles must not mutate its input')

    def test_counts_match_between_raw_and_trimmed(self):
        d = {'title': ' a ', 'panels': [{'title': ' b '}, {'title': None}]}
        trimmed = cd.trim_titles(copy.deepcopy(d))
        self.assertEqual(len(cd.collect_titles(d)), len(cd.collect_titles(trimmed)))


class TestCheckOnlyCLI(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, self.tmp)

    def write(self, dashboard):
        path = os.path.join(self.tmp, 'dash.json')
        with open(path, 'w', encoding='utf-8') as fh:
            json.dump(dashboard, fh, sort_keys=True, indent=4, ensure_ascii=False)
            fh.write('\n')
        return path

    def clean_dashboard(self, **over):
        d = copy.deepcopy(MINIMAL)
        d.update({'editable': False, 'refresh': False, 'timezone': '',
                  'time': {'from': 'now-12h', 'to': 'now'}, 'id': None})
        d.update(over)
        return d

    def test_clean_dashboard_passes(self):
        r = run_cli('--check-only', self.write(self.clean_dashboard()))
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)

    def test_untrimmed_title_fails_and_explains_why(self):
        # The regression this guards: exit 1 must PRINT the reason, not just fail.
        path = self.write(self.clean_dashboard(panels=[{'title': 'Buffer Pool Data '}]))
        r = run_cli('--check-only', path)
        self.assertEqual(r.returncode, 1)
        self.assertIn('title:', r.stdout)
        self.assertIn('"Buffer Pool Data " -> "Buffer Pool Data"', r.stdout)

    def test_write_mode_fixes_then_check_passes(self):
        path = self.write(self.clean_dashboard(panels=[{'title': 'Space Used '}]))
        self.assertEqual(run_cli(path).returncode, 0)
        with open(path, encoding='utf-8') as fh:
            self.assertEqual(json.load(fh)['panels'][0]['title'], 'Space Used')
        self.assertEqual(run_cli('--check-only', path).returncode, 0)

    def test_other_cleanupers_still_reported(self):
        # trim_titles must not mask the pre-existing checks.
        path = self.write(self.clean_dashboard(editable=True, timezone='browser'))
        r = run_cli('--check-only', path)
        self.assertEqual(r.returncode, 1)
        self.assertIn('editable:', r.stdout)
        self.assertIn('timezone:', r.stdout)


class TestRepoDashboards(unittest.TestCase):
    """Replays the dashboards.yml `check` job over every committed dashboard."""

    def dashboards(self):
        for dp, _, fns in os.walk(DASH_DIR):
            for fn in sorted(fns):
                if fn.endswith('.json'):
                    yield os.path.join(dp, fn)

    def test_all_dashboards_pass_check_only(self):
        failures = [p for p in self.dashboards()
                    if run_cli('--check-only', p).returncode != 0]
        self.assertEqual(failures, [], f'{len(failures)} dashboard(s) fail the gate')

    def test_no_untrimmed_titles_remain(self):
        offenders = []
        for path in self.dashboards():
            with open(path, encoding='utf-8') as fh:
                for title in cd.collect_titles(json.load(fh)):
                    if isinstance(title, str) and title != title.strip():
                        offenders.append((path, title))
        self.assertEqual(offenders, [])

    def test_all_dashboards_are_valid_json(self):
        for path in self.dashboards():
            with open(path, encoding='utf-8') as fh:
                json.load(fh)


if __name__ == '__main__':
    unittest.main(verbosity=2)
