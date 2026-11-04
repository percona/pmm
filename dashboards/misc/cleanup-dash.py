#!/usr/bin/env python3

import sys
import json
import copy
import datetime
import argparse
import re

def set_dashboard_id_to_null(dashboard):
    """To remove any dashboard id. New one is set by grafana."""
    for element in enumerate(dashboard.copy()):
        if 'id' in element:
            dashboard['id'] = None

    return dashboard

def set_editable(dashboard):
    """Set Editable Dashboard."""
    if 'editable' not in dashboard.keys():
        return dashboard

    dashboard['editable'] = False
    return dashboard

def set_refresh(dashboard):
    """Set Dashboard refresh."""
    if 'refresh' not in dashboard.keys():
        return dashboard

    dashboard['refresh'] = False
    return dashboard

def set_timezone(dashboard):
    """Set Dashboard Time zone."""
    
    dashboard['timezone'] = ""
    return dashboard

def set_time(dashboard):
    """Set Dashboard Time Range."""

    dashboard['time']['from'] = "now-12h"
    dashboard['time']['to'] = "now"
    return dashboard

def walk_titles(node, callback):
    """Apply callback to every 'title' value in the dashboard tree."""
    if isinstance(node, dict):
        for key, value in node.items():
            if key == 'title' and isinstance(value, str):
                node[key] = callback(value)
            else:
                walk_titles(value, callback)
    elif isinstance(node, list):
        for item in node:
            walk_titles(item, callback)

    return node

def trim_titles(dashboard):
    """Strip leading/trailing whitespace from every title."""

    return walk_titles(dashboard, str.strip)

def collect_titles(dashboard):
    """Return every title in the dashboard, in traversal order."""
    titles = []

    def record(title):
        titles.append(title)
        return title

    walk_titles(dashboard, record)
    return titles

def main():
    parser = argparse.ArgumentParser(description='Dashboard cleaner')
    parser.add_argument('dashboard_file', type=str, help='dashboard file to cleanup')
    parser.add_argument('--check-only', action='store_true', help='check only mode')
    args = parser.parse_args()

    with open(args.dashboard_file, 'r') as dashboard_file:
        dashboard = json.loads(dashboard_file.read())
        raw_dashboard = copy.deepcopy(dashboard)

    CLEANUPERS = [set_editable, set_time, set_timezone, set_refresh, set_dashboard_id_to_null, trim_titles]

    for func in CLEANUPERS:
        dashboard = func(dashboard)

    dashboard_json = json.dumps(
        dashboard,
        sort_keys=True,
        indent=4,
        separators=(',', ': '),
        ensure_ascii=False,
    )

    if args.check_only:
        if raw_dashboard == dashboard:
            print('Dashboard is already cleaned up.')
            exit(0)
        else:
            def jv(v):
                return json.dumps(v)

            issues = []
            if raw_dashboard.get('editable') != dashboard.get('editable'):
                issues.append(f"  editable: {jv(raw_dashboard.get('editable'))} -> {jv(dashboard.get('editable'))}")
            if raw_dashboard.get('refresh') != dashboard.get('refresh'):
                issues.append(f"  refresh: {jv(raw_dashboard.get('refresh'))} -> {jv(dashboard.get('refresh'))}")
            if raw_dashboard.get('timezone') != dashboard.get('timezone'):
                issues.append(f"  timezone: {jv(raw_dashboard.get('timezone'))} -> {jv(dashboard.get('timezone'))}")
            if raw_dashboard.get('time', {}).get('from') != dashboard.get('time', {}).get('from'):
                issues.append(f"  time.from: {jv(raw_dashboard.get('time', {}).get('from'))} -> {jv(dashboard.get('time', {}).get('from'))}")
            if raw_dashboard.get('time', {}).get('to') != dashboard.get('time', {}).get('to'):
                issues.append(f"  time.to: {jv(raw_dashboard.get('time', {}).get('to'))} -> {jv(dashboard.get('time', {}).get('to'))}")
            if raw_dashboard.get('id') != dashboard.get('id'):
                issues.append(f"  id: {jv(raw_dashboard.get('id'))} -> {jv(dashboard.get('id'))}")
            for raw_title, title in zip(collect_titles(raw_dashboard), collect_titles(dashboard)):
                if raw_title != title:
                    issues.append(f"  title: {jv(raw_title)} -> {jv(title)}")
            print(f'Dashboard: {args.dashboard_file}')
            for issue in issues:
                print(issue)
            exit(1)

    with open(args.dashboard_file, 'w') as dashboard_file:
        dashboard_file.write(dashboard_json)
        dashboard_file.write('\n')
        print('Dashboard is cleaned up successfully.')


if __name__ == '__main__':
    main()
