#!/usr/bin/env python3
"""Checks for Valkey/Redis dashboard queries (see issue #5497).

Run from the repo root:  python3 -m unittest discover -s dashboards/misc -p 'test_*.py' -v
"""

import glob
import json
import os
import re
import unittest

REPO = os.environ.get('PMM_REPO', os.getcwd())
VALKEY_DIR = os.path.join(REPO, 'dashboards', 'dashboards', 'Valkey')

# `role` is an optional custom label, so a positive match drops unlabeled services.
POSITIVE_ROLE = re.compile(r'\brole\s*=~?\s*"primary"')


def exprs(node):
    if isinstance(node, dict):
        for key, value in node.items():
            if key == 'expr' and isinstance(value, str):
                yield value
            else:
                yield from exprs(value)
    elif isinstance(node, list):
        for item in node:
            yield from exprs(item)


class TestValkeyDashboards(unittest.TestCase):
    def test_queries_do_not_require_role_label(self):
        paths = sorted(glob.glob(os.path.join(VALKEY_DIR, '*.json')))
        self.assertTrue(paths, f'no dashboards found in {VALKEY_DIR}')
        offending = []
        for path in paths:
            with open(path, encoding='utf-8') as f:
                dash = json.load(f)
            offending += [f'{os.path.basename(path)}: {e}' for e in exprs(dash) if POSITIVE_ROLE.search(e)]
        self.assertEqual(offending, [])


if __name__ == '__main__':
    unittest.main()
