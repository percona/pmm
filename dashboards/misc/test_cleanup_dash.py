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


# A systemic fault puts every dashboard in the same bucket, and listing all of
# them buries the one sentence that explains what to do about it.
MAX_REPORTED = 10


def capped(entries):
    """Trim a failure list, noting how much was left out."""
    if len(entries) <= MAX_REPORTED:
        return entries
    return entries[:MAX_REPORTED] + [f'... and {len(entries) - MAX_REPORTED} more']


def indented(path, text):
    """One report block: the dashboard, then every line of detail, indented."""
    return '\n'.join([f'Dashboard: {path}',
                       *(f'  {line}' for line in text.splitlines())])


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

    def test_only_string_titles_are_collected(self):
        # walk_titles fires the callback under isinstance(value, str), so every
        # title downstream is a str. The rest of this file relies on that.
        d = {'title': ' a ', 'panels': [{'title': ' b '}, {'title': None},
                                        {'title': 7}]}
        self.assertEqual(cd.collect_titles(d), [' a ', ' b '])


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

    def test_every_untrimmed_title_is_reported(self):
        # The reporter pairs raw and cleaned titles positionally with zip()
        # (cleanup-dash.py:121), which misaligns or truncates silently if the
        # title set ever changes size. Assert the exact pairs: a misaligned
        # report still has the right number of lines, so a count proves nothing.
        path = self.write(self.clean_dashboard(panels=[
            {'title': 'A ', 'panels': [{'title': ' B'}]},
            {'title': ' C ', 'type': 'text'},
        ]))
        r = run_cli('--check-only', path)
        self.assertEqual(r.returncode, 1)
        reported = sorted(line.strip() for line in r.stdout.splitlines()
                          if line.startswith('  title:'))
        self.assertEqual(reported, [
            'title: " B" -> "B"',
            'title: " C " -> "C"',
            'title: "A " -> "A"',
        ], r.stdout)

    def test_reporter_covers_every_normalised_field(self):
        # Every cleanuper needs a matching case in the --check-only reporter, or
        # the tool exits 1 without naming a field. One dirty field at a time, so
        # a missing case cannot hide behind another field's output.
        cases = [
            ('editable:', {'editable': True}),
            ('refresh:', {'refresh': '1m'}),
            ('timezone:', {'timezone': 'browser'}),
            ('time.from:', {'time': {'from': 'now-6h', 'to': 'now'}}),
            ('time.to:', {'time': {'from': 'now-12h', 'to': 'now-1h'}}),
            ('id:', {'id': 42}),
            ('title:', {'panels': [{'title': 'Dirty '}]}),
        ]
        for field, dirty in cases:
            with self.subTest(field=field):
                r = run_cli('--check-only', self.write(self.clean_dashboard(**dirty)))
                self.assertEqual(r.returncode, 1, r.stdout)
                self.assertIn(field, r.stdout)

    def test_other_cleanupers_still_reported(self):
        # trim_titles must not mask the pre-existing checks.
        path = self.write(self.clean_dashboard(editable=True, timezone='browser'))
        r = run_cli('--check-only', path)
        self.assertEqual(r.returncode, 1)
        self.assertIn('editable:', r.stdout)
        self.assertIn('timezone:', r.stdout)


class TestRepoDashboards(unittest.TestCase):
    """Replays the dashboards.yml `check` job over every committed dashboard.

    Wider than the workflow step, which only checks the dashboards a PR touched:
    one dirty dashboard fails every PR that runs this suite. dashboards.yml
    therefore runs it only for PRs that change a dashboard JSON file or
    cleanup-dash.py itself, so nothing else is gated on the state of the tree.
    """

    def dashboards(self):
        paths = []
        for dp, dns, fns in os.walk(DASH_DIR):
            dns.sort()  # the failure reports below are ordered; keep them stable
            for fn in sorted(fns):
                if fn.endswith('.json'):
                    paths.append(os.path.join(dp, fn))
        # A renamed tree or a partial checkout would otherwise let every
        # whole-tree test in this class pass without inspecting anything.
        self.assertTrue(paths, f'no dashboards found under {DASH_DIR}')
        return paths

    def test_all_dashboards_pass_check_only(self):
        # Keep the tool's own report: the exit code alone says nothing about
        # which field is dirty, and this gate covers the whole tree. The three
        # failure kinds need different remedies, so they are collected apart --
        # telling someone to run the cleanuper when it cannot help, or to repair
        # JSON that is not broken, is worse than saying nothing.
        paths = self.dashboards()
        dirty = []       # the reporter named the fields it would change
        unreported = []  # exited non-zero having printed only its header
        crashed = []     # exited non-zero before printing anything at all
        for path in paths:
            result = run_cli('--check-only', path)
            if result.returncode == 0:
                continue

            report = result.stdout.strip()
            if report.splitlines()[1:]:
                # First stdout line is 'Dashboard: <path>', the rest are issues.
                dirty.append(report)
            elif report:
                # A header and nothing else: some cleanuper changed the
                # dashboard and the reporter has no case for it. Write mode
                # still fixes the file, so do not send anyone JSON-hunting.
                unreported.append(f'Dashboard: {path}')
            else:
                # Nothing on stdout: the tool died before the reporter ran, so
                # stderr is the only evidence there is. It is never used in the
                # branches above, where it would only be unrelated noise.
                detail = (result.stderr.strip()
                          or f'exited {result.returncode} but printed no reason')
                crashed.append(indented(path, detail))

        messages = []
        if dirty:
            messages += [
                f'{len(dirty)} of {len(paths)} dashboard(s) need cleanup:',
                *capped(dirty),
                'Fix the reported fields with '
                '`python3 dashboards/misc/cleanup-dash.py <file>`, then review '
                'the diff: write mode re-serialises the whole file, so some '
                'dashboards pick up unrelated reformatting.',
            ]
        if unreported:
            messages += [
                f'{len(unreported)} of {len(paths)} dashboard(s) failed the '
                'check without naming a field:',
                *capped(unreported),
                'Write mode still fixes these, but --check-only cannot say what '
                'it changed: a cleanuper in cleanup-dash.py has no matching case '
                'in its --check-only reporter. Add one.',
            ]
        if crashed:
            messages += [
                f'{len(crashed)} of {len(paths)} dashboard(s) could not be '
                'checked at all:',
                *capped(crashed),
                'The cleanuper cannot fix these -- write mode hits the same '
                'error. Repair the dashboard JSON, or harden cleanup-dash.py.',
            ]
        if messages:
            self.fail('\n'.join(messages))

    def test_no_untrimmed_titles_remain(self):
        offenders = []
        for path in self.dashboards():
            # subTest so an unparseable file names itself and the remaining
            # dashboards are still scanned; json.load raises without the path.
            with self.subTest(path=path), open(path, encoding='utf-8') as fh:
                # collect_titles only ever yields strings, see
                # test_only_string_titles_are_collected.
                for title in cd.collect_titles(json.load(fh)):
                    if title != title.strip():
                        offenders.append((path, title))
        self.assertEqual(offenders, [])

    def test_all_dashboards_are_valid_json(self):
        for path in self.dashboards():
            with self.subTest(path=path), open(path, encoding='utf-8') as fh:
                json.load(fh)


if __name__ == '__main__':
    unittest.main(verbosity=2)
