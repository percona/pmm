#!/usr/bin/env python3
"""Synthetic PMM fleet + annotations for the PMM-15663 benchmark.

gen.py metrics [--stress] [--now MS]   -> VictoriaMetrics /api/v1/import JSON lines on stdout
gen.py annotations N [--now MS]       -> writes ann-N.json (list of dicts) in cwd
gen.py fleet                          -> prints fleet JSON (nodes, services)
"""
import json, random, sys

NODES = int(__import__("os").environ.get("NODES", 1000))
ENVS = 5
KINDS = [('mysql', 2), ('postgresql', 2), ('mongodb', 2), ('valkey', 2)]
STEP = 60_000
RECENT = 180          # 3h of 1-min samples
DAYS = 30             # plus one sample per day for 30 days (per-day index entries)

def fleet():
    nodes, services = [], []
    for i in range(NODES):
        env = f'env-{i % ENVS}'
        n = dict(node_name=f'node-{i:04d}', node_id=f'nid-{i:04d}', environment=env,
                 cluster=f'cl-{i // 10:03d}')
        nodes.append(n)
        for kind, cnt in KINDS:
            for k in range(cnt):
                sn = f'{n["node_name"]}-{kind}-{k}'
                services.append(dict(service_name=sn, service_id=f'sid-{i:04d}-{kind}-{k}', service_type=kind,
                                     node_name=n['node_name'], node_id=n['node_id'], environment=env,
                                     cluster=n['cluster'], replication_set=f'rs-{i // 3:03d}-{kind}'))
    return nodes, services

def timestamps(now):
    now -= now % STEP
    old = [now - d * 86_400_000 for d in range(DAYS, 0, -1)]
    return old + [now - (RECENT - j) * STEP for j in range(RECENT)]

NODE_METRICS = []
def nm(name, kind, **dims):
    NODE_METRICS.append((name, kind, dims))
for m in ['user', 'system', 'idle', 'iowait', 'irq', 'softirq', 'steal', 'nice']:
    pass
nm('node_cpu_seconds_total', 'c', cpu=[str(c) for c in range(4)], mode=['user', 'system', 'idle', 'iowait', 'irq', 'softirq', 'steal', 'nice'])
nm('node_cpu_average', 'g', mode=['user', 'system', 'idle', 'iowait', 'irq', 'softirq', 'steal', 'nice', 'total'])
for d in ['read_bytes', 'written_bytes', 'reads_completed', 'writes_completed', 'read_time_seconds', 'write_time_seconds', 'io_time_seconds', 'io_time_weighted_seconds']:
    nm(f'node_disk_{d}_total', 'c', device=['sda', 'sdb'])
for f in ['size', 'free', 'avail']:
    nm(f'node_filesystem_{f}_bytes', 'g', mountpoint=['/', '/data', '/boot'], fstype=['xfs'], device=['/dev/sda1'])
for d in ['receive', 'transmit']:
    for x in ['bytes', 'packets', 'errs', 'drop']:
        nm(f'node_network_{d}_{x}_total', 'c', device=['eth0', 'eth1'])
for g in ['node_load1', 'node_load5', 'node_load15', 'node_procs_running', 'node_procs_blocked', 'node_boot_time_seconds',
          'node_time_seconds', 'process_virtual_memory_max_bytes', 'node_memory_MemTotal_bytes', 'node_memory_MemAvailable_bytes',
          'node_memory_MemFree_bytes', 'node_memory_Cached_bytes', 'node_memory_Buffers_bytes', 'node_memory_SwapTotal_bytes',
          'node_memory_SwapFree_bytes']:
    nm(g, 'g')
for c in ['node_context_switches_total', 'node_intr_total', 'node_forks_total', 'node_vmstat_pgpgin', 'node_vmstat_pgpgout',
          'node_vmstat_pswpin', 'node_vmstat_pswpout']:
    nm(c, 'c')

MYSQL = ['aborted_clients', 'aborted_connects', 'bytes_received', 'bytes_sent', 'created_tmp_disk_tables', 'created_tmp_files',
         'created_tmp_tables', 'innodb_data_fsyncs', 'innodb_data_reads', 'innodb_data_writes', 'opened_files',
         'opened_table_definitions', 'queries', 'questions', 'select_full_join', 'select_full_range_join', 'select_range',
         'select_range_check', 'select_scan', 'sort_merge_passes', 'sort_range', 'sort_rows', 'sort_scan',
         'table_locks_immediate', 'table_locks_waited', 'table_open_cache_hits', 'table_open_cache_misses']
MYSQL_G = ['max_used_connections', 'open_files', 'open_table_definitions', 'qcache_free_memory', 'threads_cached',
           'threads_connected', 'threads_running', 'uptime']
MYSQL_V = ['innodb_buffer_pool_size', 'max_connections', 'open_files_limit', 'query_cache_size', 'table_definition_cache',
           'thread_cache_size']
SVC_METRICS = {
    'mysql': [('mysql_up', 'u', {})] + [(f'mysql_global_status_{m}', 'c', {}) for m in MYSQL]
             + [(f'mysql_global_status_{m}', 'g', {}) for m in MYSQL_G] + [(f'mysql_global_variables_{m}', 'g', {}) for m in MYSQL_V]
             + [('mysql_info_schema_user_statistics_connected_time_seconds_total', 'c', {'user': ['app', 'root']})],
    'postgresql': [('pg_up', 'u', {}), ('postgresql_up', 'u', {}), ('pg_postmaster_start_time_seconds', 'g', {}),
                   ('pg_stat_database_numbackends', 'g', {'datname': ['app', 'postgres']}),
                   ('pg_stat_database_xact_commit', 'c', {'datname': ['app', 'postgres']}),
                   ('pg_stat_database_xact_rollback', 'c', {'datname': ['app', 'postgres']}),
                   ('pg_stat_database_tup_fetched', 'c', {'datname': ['app', 'postgres']})],
    'mongodb': [('mongodb_up', 'u', {}), ('mongodb_instance_uptime_seconds', 'g', {}), ('mongodb_connections', 'g', {'state': ['current', 'available']}),
                ('mongodb_op_counters_total', 'c', {'type': ['query', 'insert', 'update', 'delete', 'getmore', 'command']})],
    'valkey': [('redis_up', 'u', {}), ('redis_instance_info', 'u', {}), ('redis_connected_clients', 'g', {}),
               ('redis_commands_total', 'c', {'cmd': ['get', 'set', 'del', 'hget', 'hset']})],
}
STRESS_SVC = {
    'mysql': [('mysql_info_schema_table_rows', 'g', 350), ('mysql_info_schema_table_size', 'g', 350)],
    'postgresql': [('pg_stat_user_tables_n_live_tup', 'g', 300), ('pg_stat_user_tables_seq_scan', 'c', 300)],
}
STRESS_NODE = [('node_interrupts_total', 'c', 200)]

def expand(dims):
    combos = [{}]
    for k, vals in dims.items():
        combos = [dict(c, **{k: v}) for c in combos for v in vals]
    return combos

def values(kind, ts, rnd):
    if kind == 'u':
        return [1] * len(ts)
    if kind == 'g':
        base = rnd.uniform(1, 1000)
        return [round(base * rnd.uniform(0.8, 1.2), 3) for _ in ts]
    rate = rnd.uniform(0.1, 500)
    v, out, prev = rnd.uniform(1e3, 1e6), [], ts[0]
    for t in ts:
        v += rate * (t - prev) / 1000
        prev = t
        out.append(round(v, 1))
    return out

def emit(labels, kind, ts, rnd, w):
    w.write(json.dumps({'metric': labels, 'values': values(kind, ts, rnd), 'timestamps': ts}, separators=(',', ':')))
    w.write('\n')

def metrics(stress, now):
    rnd = random.Random(15663)
    ts = timestamps(now)
    # filler series only need per-day index entries and a recent sample, not dense data
    ts_fill = ts[:DAYS] + ts[-10:]
    nodes, services = fleet()
    w = sys.stdout
    n_series = 0
    shard, nshards = map(int, __import__('os').environ.get('SHARD', '0/1').split('/'))
    nodes = [n for i, n in enumerate(nodes) if i % nshards == shard]
    keep = {n['node_name'] for n in nodes}
    services = [s for s in services if s['node_name'] in keep]
    for n in nodes:
        base = dict(node_name=n['node_name'], node_id=n['node_id'], node_type='generic', agent_type='node_exporter',
                    job='node_exporter_agent_hr-5s', instance=f'agent-node-{n["node_id"]}')
        emit(dict(base, __name__='up'), 'u', ts, rnd, w); n_series += 1
        for name, kind, dims in NODE_METRICS:
            for c in expand(dims):
                emit(dict(base, __name__=name, **c), kind, ts, rnd, w); n_series += 1
        if stress:
            for name, kind, cnt in STRESS_NODE:
                for j in range(cnt):
                    emit(dict(base, __name__=name, cpu=str(j % 4), devices=f'irq-{j}'), kind, ts_fill, rnd, w); n_series += 1
    for s in services:
        exp = {'mysql': 'mysqld_exporter', 'postgresql': 'postgres_exporter', 'mongodb': 'mongodb_exporter', 'valkey': 'valkey_exporter'}[s['service_type']]
        base = dict(s, node_type='generic', agent_type=exp, job=f'{exp}_agent_hr-5s', instance=f'agent-{s["service_id"]}')
        emit(dict(base, __name__='up'), 'u', ts, rnd, w); n_series += 1
        for name, kind, dims in SVC_METRICS[s['service_type']]:
            for c in expand(dims):
                emit(dict(base, __name__=name, **c), kind, ts, rnd, w); n_series += 1
        if stress:
            for name, kind, cnt in STRESS_SVC.get(s['service_type'], []):
                for j in range(cnt):
                    emit(dict(base, __name__=name, schema=f'db{j % 7}', table=f't{j}'), kind, ts_fill, rnd, w); n_series += 1
    print(f'series={n_series}', file=sys.stderr)

def annotations(count, now):
    """Scoped annotations over 30 days: 60% one service, 30% node, 10% 2-4 services on one node."""
    rnd = random.Random(15663)
    nodes, services = fleet()
    by_node = {}
    for s in services:
        by_node.setdefault(s['node_name'], []).append(s)
    out = []
    for i in range(count):
        t = now - rnd.randint(0, DAYS * 86_400_000 - 1)
        n = rnd.choice(nodes)
        r = rnd.random()
        if r < 0.6:
            svcs = [rnd.choice(by_node[n['node_name']])]
        elif r < 0.9:
            svcs = []
        else:
            svcs = rnd.sample(by_node[n['node_name']], rnd.randint(2, 4))
        names = [s['service_name'] for s in svcs]
        text = f'ann-{i:05d} deploy v{rnd.randint(1, 99)}.{rnd.randint(0, 9)}'
        post = []
        if names:
            post.append('Service Name: ' + ', '.join(names))
        if not names:
            post.append('Node Name: ' + n['node_name'])
        out.append(dict(id=f'ann-{i:05d}', time=t, text=text + ' (' + '. '.join(post) + ')', user_tags=['deploy'],
                        node_name=n['node_name'], services=[{k: s[k] for k in ('service_name', 'service_type', 'replication_set')} for s in svcs],
                        environment=n['environment'], cluster=n['cluster']))
    json.dump(out, open(f'ann-{count}.json', 'w'))
    print(f'wrote ann-{count}.json', file=sys.stderr)

if __name__ == '__main__':
    import time
    args = sys.argv[1:]
    now = int(time.time() * 1000)
    if '--now' in args:
        now = int(args[args.index('--now') + 1])
    if args[0] == 'metrics':
        metrics('--stress' in args, now)
    elif args[0] == 'annotations':
        annotations(int(args[1]), now)
    elif args[0] == 'fleet':
        print(json.dumps(fleet()))
