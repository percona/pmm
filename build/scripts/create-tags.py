#!/usr/bin/python2

from __future__ import print_function, unicode_literals
import os, subprocess, time

REPOS = [
    "sources/pmm/src/github.com/percona/pmm",
    "sources/dbaas-controller/src/github.com/percona-platform/dbaas-controller",
    "sources/pmm-dump",
    "sources/pmm-qa/src/github.com/percona/pmm-qa",
    "sources/pmm-ui-tests/src/github.com/percona/pmm-ui-tests",
    "sources/grafana/src/github.com/grafana/grafana",
    "sources/grafana-dashboards",
    "sources/node_exporter/src/github.com/prometheus/node_exporter",
    "sources/mysqld_exporter/src/github.com/percona/mysqld_exporter",
    "sources/mongodb_exporter/src/github.com/percona/mongodb_exporter",
    "sources/postgres_exporter/src/github.com/percona/postgres_exporter",
    "sources/clickhouse_exporter/src/github.com/Percona-Lab/clickhouse_exporter",
    "sources/proxysql_exporter/src/github.com/percona/proxysql_exporter",
    "sources/rds_exporter/src/github.com/percona/rds_exporter",
    "sources/azure_metrics_exporter/src/github.com/percona/azure_metrics_exporter",
    ".",
]

tty = subprocess.check_output("tty", shell=True).strip()
env = os.environ.copy()
env["GPG_TTY"] = tty

with open("./VERSION", "r") as f:
    version = f.read().strip()

print(tty, version)

subprocess.check_call("git submodule update", shell=True)

for repo in REPOS:
    print("==>", repo)

    tag = "v" + version
    cmd = "git tag --message='Version {version}.' --sign {tag}".format(version=version, tag=tag)
    print(">", cmd)
    subprocess.check_call(cmd, shell=True, cwd=repo, env=env)

    cmd = "git push origin {tag}".format(tag=tag)
    print(">", cmd)
    subprocess.check_call(cmd, shell=True, cwd=repo)

subprocess.check_call("git submodule status", shell=True)
