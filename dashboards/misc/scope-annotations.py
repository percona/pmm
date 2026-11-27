#!/usr/bin/env python3
"""Turn node and service annotation tags into scope tags.

A scope tag carries the series selector of the variable it replaces:

    pmm_scope:<label>:<selector>{...,<label>=~"${<label>:regex}"}

nginx routes annotation queries with such a tag to pmm-managed, which expands the selector into
names with the caller's LBAC filters and asks Grafana for those names. Existing scope tags are
rebuilt from the current variable query.

Usage: scope-annotations.py [--check-only] <dashboard.json>...
"""

import argparse
import json
import os
import re
import sys

SCOPE_PREFIX = 'pmm_scope:'
SCOPE_LABELS = ('node_name', 'service_name')
# pmm-managed accepts at most this many scope tags in one query.
MAX_SCOPE_TAGS = 4
# Templates stay short; Grafana fills in the values, and pmm-managed takes up to 64 KiB.
MAX_SELECTOR_LENGTH = 4096

# allValues that Grafana sends as a pattern, which no annotation tag equals.
PATTERN_ALL_VALUES = ('.+', '.*')

# Variables whose options don't come from label_values(), with an equivalent selector.
OVERRIDES = {
    # The options come from query_result() of a join, which label values can't evaluate;
    # every service with Galera variables also reports wsrep_auto_increment_control.
    ('MySQL/PXC_Galera_Nodes_Compare.json', 'service_name'):
        'label_values(mysql_galera_variables_info{wsrep_cluster_name=~"$cluster"}, service_name)',
}

# Definitions left alone, with the reason.
EXEMPT = {
    # QAN draws no annotation markers, and its definition requires every tag (matchAny off).
    'Query Analytics/pmm-qan.json',
}

VAR_REF = re.compile(r'\$(?:\{(\w+)(?::\w+)?\}|(\w+))')
REGEX_REF = re.compile(r'\$\{\w+:regex\}')
LABEL_VALUES = re.compile(r'\s*label_values\(\s*(.*?)\s*,\s*(\w+)\s*\)\s*', re.S)
MATCHER = re.compile(r'(\w+)\s*(=~|!~|!=|=)\s*"((?:[^"\\]|\\.)*)"')


class ScopeError(Exception):
    pass


def first_variables(dashboard):
    """Grafana resolves a duplicated variable name to its first definition."""
    seen = {}
    for var in dashboard.get('templating', {}).get('list', []):
        seen.setdefault(var.get('name'), var)
    return seen


def var_name(ref):
    m = VAR_REF.fullmatch(ref)
    return m and (m.group(1) or m.group(2))


def check_all_value(name, variables):
    """Refuse a variable whose All would put its whole option list into the tag."""
    var = variables.get(name)
    if var is None:
        raise ScopeError(f'${name} is not a dashboard variable')
    if var.get('includeAll') and not var.get('allValue'):
        raise ScopeError(f'${name} offers All without an allValue, so All would put its whole list '
                         'in the tag; give it allValue ".*" or ".+"')


def regex_refs(text, variables):
    """Rewrite every variable reference to ${name:regex}.

    The regex format keeps a multi-value selection in one tag; with any other format Grafana
    splits it into one tag per value.
    """
    def sub(m):
        name = m.group(1) or m.group(2)
        check_all_value(name, variables)
        return '${%s:regex}' % name

    return VAR_REF.sub(sub, text)


def scope_tag(rel, label, variables):
    """Build the scope tag for the variable named label from its own query."""
    var = variables.get(label)
    if label not in SCOPE_LABELS or not var:
        raise ScopeError(f'no {label} variable to scope')
    check_all_value(label, variables)

    query = OVERRIDES.get((rel, label)) or var.get('query')
    if isinstance(query, dict):
        query = query.get('query')
    m = LABEL_VALUES.fullmatch(query or '')
    if not m or m.group(2) != label:
        raise ScopeError(f'{label}: options don\'t come from label_values(..., {label}), add an override')

    selector = m.group(1).strip()
    if '{' not in selector:
        selector += '{}'

    def matcher(mm):
        name, op, value = mm.groups()
        if not VAR_REF.search(value):
            return mm.group(0)
        if op == '!=':
            # The regex form of the value would be compared literally.
            raise ScopeError(f'{label}: {name}!= on a variable has no regex equivalent; use !~')
        op = {'=': '=~'}.get(op, op)
        return f'{name}{op}"{regex_refs(value, variables)}"'

    head, inner = selector.split('{', 1)
    head = head.strip()
    inner = MATCHER.sub(matcher, inner.rstrip().removesuffix('}').strip())
    matchers = [inner] if inner else []
    if var_name(head):
        # A variable holding the metric name becomes a __name__ matcher.
        matchers.insert(0, '__name__=~"%s"' % regex_refs(head, variables))
        head = ''
    matchers.append('%s=~"${%s:regex}"' % (label, label))
    selector = '%s{%s}' % (head, ','.join(matchers))

    if selector.count('$') != len(REGEX_REF.findall(selector)) or '\\' in selector:
        raise ScopeError(f'{label}: selector must use only ${{var:regex}} references and no backslashes')
    if len(selector) > MAX_SELECTOR_LENGTH:
        raise ScopeError(f'{label}: selector longer than {MAX_SELECTOR_LENGTH}')

    return f'{SCOPE_PREFIX}{label}:{selector}'


def scoped_tags(rel, tags, variables):
    """Return the tags with every pattern-on-All variable replaced by a scope tag."""
    out = []
    for tag in tags:
        if tag.startswith(SCOPE_PREFIX):
            out.append(scope_tag(rel, tag[len(SCOPE_PREFIX):].split(':', 1)[0], variables))
            continue
        name = var_name(tag)
        var = variables.get(name) if name else None
        if var and var.get('includeAll') and var.get('allValue') in PATTERN_ALL_VALUES:
            out.append(scope_tag(rel, name, variables))
            continue
        out.append(tag)

    if sum(t.startswith(SCOPE_PREFIX) for t in out) > MAX_SCOPE_TAGS:
        raise ScopeError(f'more than {MAX_SCOPE_TAGS} scope tags')
    return out


def tag_changes(rel, dashboard):
    """Return {old tag: new tag} for the dashboard's PMM Annotations definitions."""
    if rel in EXEMPT:
        return {}

    variables = first_variables(dashboard)
    changes = {}
    for a in dashboard.get('annotations', {}).get('list', []):
        if a.get('name') != 'PMM Annotations':
            continue
        for holder in (a, a.get('target') or {}):
            tags = holder.get('tags') or []
            new = scoped_tags(rel, tags, variables)
            if any(t.startswith(SCOPE_PREFIX) for t in new) and not (a.get('target') or {}).get('matchAny'):
                raise ScopeError('scope tags need matchAny')
            for old_tag, new_tag in zip(tags, new):
                if old_tag != new_tag:
                    changes[old_tag] = new_tag
    return changes


def annotations_span(text):
    """Return the start and end offsets of the top-level "annotations" value."""
    decoder = json.JSONDecoder()
    depth, i = 0, 0
    while i < len(text):
        c = text[i]
        if c == '"':
            _, end = decoder.raw_decode(text, i)
            rest = text[end:].lstrip()
            if depth == 1 and text[i:end] == '"annotations"' and rest.startswith(':'):
                start = len(text) - len(rest[1:].lstrip())
                _, stop = decoder.raw_decode(text, start)
                return start, stop
            i = end
            continue
        if c in '{[':
            depth += 1
        elif c in '}]':
            depth -= 1
        i += 1
    raise ScopeError('no top-level annotations object')


def rewrite(rel, text):
    """Return the dashboard text with scope tags, changing only the tag strings."""
    dashboard = json.loads(text)
    changes = tag_changes(rel, dashboard)
    if not changes:
        return text

    start, end = annotations_span(text)
    block = text[start:end]
    for old_tag, new_tag in changes.items():
        block = block.replace(json.dumps(old_tag, ensure_ascii=False), json.dumps(new_tag, ensure_ascii=False))
    new_text = text[:start] + block + text[end:]

    # The textual edit must equal the structural one.
    expected = json.loads(text)
    for a in expected.get('annotations', {}).get('list', []):
        if a.get('name') == 'PMM Annotations':
            for holder in (a, a.get('target') or {}):
                if 'tags' in holder:
                    holder['tags'] = [changes.get(t, t) for t in holder['tags']]
    if json.loads(new_text) != expected:
        raise ScopeError('replacing the tags changed something else in the annotations block')

    return new_text


def main():
    parser = argparse.ArgumentParser(description=__doc__.split('\n', 1)[0])
    parser.add_argument('dashboards', nargs='+', help='dashboard JSON files')
    parser.add_argument('--check-only', action='store_true', help='report dashboards that need changes')
    args = parser.parse_args()

    dash_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'dashboards')
    failed = False
    for path in args.dashboards:
        rel = os.path.relpath(os.path.abspath(path), os.path.abspath(dash_dir))
        with open(path, encoding='utf-8') as f:
            text = f.read()
        try:
            new_text = rewrite(rel, text)
        except ScopeError as e:
            print(f'{rel}: {e}', file=sys.stderr)
            failed = True
            continue
        if new_text == text:
            continue
        if args.check_only:
            print(f'{rel}: annotation tags need scope tags')
            failed = True
            continue
        with open(path, 'w', encoding='utf-8') as f:
            f.write(new_text)
        print(f'{rel}: scope tags written')

    sys.exit(1 if failed else 0)


if __name__ == '__main__':
    main()
