import type { OmCluster, OmService, OmTopologyResponse } from '@pmm/plugins-om';
// The package exports this same function; its index would load the whole Operations UI here.
import { toClusterRows } from '@pmm/plugins-om/src/topology';
import { healthySnapshot } from './fixtures/healthy';
import { wrongSnapshot } from './fixtures/wrong';
import {
  hasLostQuorum,
  hasNoPrimary,
  type HealthVerdict,
  isLost,
  isStuck,
  lagLevel,
  type MemberHealthInput,
  memberHealth,
  memberVerdict,
  oplogLevel,
  routersHealth,
  setHealth,
  shardedHealth,
  standaloneHealth,
  upgradingHealth,
  worstHealth,
} from './mongodb.health';
import { buildClusterTree, groupClusters } from './mongodb.topology';
import type { MongoSnapshot } from './mongodb.types';
import { emptyMetrics, NOW, service } from './mongodb.testUtils';

let count = 0;
const named = (member: Omit<MemberHealthInput, 'name'>): MemberHealthInput => {
  count += 1;

  return { name: `m${count}`, ...member };
};
const primary = () => named({ verdict: 'up', state: 'PRIMARY' });
const secondary = () => named({ verdict: 'up', state: 'SECONDARY' });
const down = () => named({ verdict: 'down' });
const noData = () => named({ verdict: 'noData' });
const inState = (state: string, stateSeconds?: number) =>
  named({
    verdict: 'up',
    state,
    ...(stateSeconds !== undefined ? { stateSeconds } : {}),
  });
const healthOf = (verdict: HealthVerdict) => verdict.health;
const kinds = (verdict: HealthVerdict) =>
  verdict.conditions.map((condition) => condition.kind);
const parts = (...healths: HealthVerdict['health'][]): HealthVerdict[] =>
  healths.map((health) => ({ health, conditions: [] }));

describe('memberVerdict', () => {
  const svc = service('a');

  it('is up with an up series', () => {
    const metrics = emptyMetrics();

    metrics.up[svc.serviceId] = { value: 1, clusterRole: 'mongod' };
    expect(memberVerdict(svc, metrics)).toBe('up');
  });

  it('is down when the exporter reports 0, or the inventory says down', () => {
    const metrics = emptyMetrics();

    metrics.up[svc.serviceId] = { value: 0, clusterRole: 'mongod' };
    expect(memberVerdict(svc, metrics)).toBe('down');

    metrics.up[svc.serviceId] = { value: 1, clusterRole: 'mongod' };
    expect(memberVerdict({ ...svc, inventoryStatus: 'down' }, metrics)).toBe(
      'down'
    );
  });

  it('is no data without a series, never up', () => {
    expect(memberVerdict(svc, emptyMetrics())).toBe('noData');
    expect(
      memberVerdict({ ...svc, inventoryStatus: 'down' }, emptyMetrics())
    ).toBe('down');
  });

  it('falls back to the inventory when metrics are missing altogether', () => {
    expect(memberVerdict(svc, null)).toBe('up');
    expect(memberVerdict({ ...svc, inventoryStatus: 'down' }, null)).toBe(
      'down'
    );
    expect(memberVerdict({ ...svc, inventoryStatus: 'unknown' }, null)).toBe(
      'noData'
    );
  });
});

describe('setHealth', () => {
  it('is healthy with a primary and every member up', () => {
    expect(setHealth([primary(), secondary(), secondary()])).toEqual({
      health: 'healthy',
      conditions: [],
    });
  });

  it('is down when every member is down', () => {
    const verdict = setHealth([down(), down(), down()], 'billing');

    expect(verdict.health).toBe('down');
    expect(verdict.conditions).toEqual([
      { kind: 'allDown', total: 3, set: 'billing' },
    ]);
  });

  it('is down when members answer but none is primary, and counts the down ones once', () => {
    const verdict = setHealth([secondary(), down(), down()]);

    expect(verdict.health).toBe('down');
    expect(verdict.conditions).toEqual([
      { kind: 'noPrimary', down: 2, total: 3 },
    ]);
    expect(healthOf(setHealth([secondary(), secondary()]))).toBe('down');
  });

  it('is down when half the members or more are down, primary or not', () => {
    const verdict = setHealth([primary(), secondary(), down(), down()]);

    expect(verdict.health).toBe('down');
    expect(kinds(verdict)).toEqual(['quorumLost']);
  });

  it('is down while a member rolls back or is removed, even with a primary', () => {
    const verdict = setHealth([primary(), secondary(), inState('ROLLBACK')]);

    expect(verdict.health).toBe('down');
    expect(kinds(verdict)).toEqual(['memberLost']);
    expect(
      healthOf(setHealth([primary(), secondary(), inState('REMOVED')]))
    ).toBe('down');
  });

  it('is degraded when a primary was elected after one went down', () => {
    const verdict = setHealth([primary(), secondary(), down()]);

    expect(verdict.health).toBe('degraded');
    expect(kinds(verdict)).toEqual(['memberDown']);
  });

  it('is no data when no member answers and too few are down to say more', () => {
    expect(setHealth([noData(), noData()])).toEqual({
      health: 'noData',
      conditions: [{ kind: 'noData', total: 2 }],
    });
    expect(healthOf(setHealth([down(), noData(), noData()]))).toBe('noData');
  });

  it('is down with nothing up and half the members down', () => {
    expect(kinds(setHealth([down(), noData()]))).toEqual(['quorumLost']);
  });

  it('is degraded, never healthy, when some members are silent', () => {
    expect(kinds(setHealth([primary(), secondary(), noData()]))).toEqual([
      'memberNoData',
    ]);
  });

  it('is degraded when a secondary is 10s behind, sustained', () => {
    const lagging = named({
      verdict: 'up',
      state: 'SECONDARY',
      lag: { now: 12, sustained: 10 },
    });
    const verdict = setHealth([primary(), lagging]);

    expect(verdict.health).toBe('degraded');
    expect(verdict.conditions).toEqual([
      { kind: 'behind', member: lagging.name, seconds: 12 },
    ]);
    expect(
      healthOf(
        setHealth([
          primary(),
          named({
            verdict: 'up',
            state: 'SECONDARY',
            lag: { now: 30, sustained: 9 },
          }),
        ])
      )
    ).toBe('healthy');
  });

  it('is degraded when the primary oplog window runs under 24h once full', () => {
    const short = named({
      verdict: 'up',
      state: 'PRIMARY',
      oplog: { seconds: 23 * 3600, fill: 0.95 },
    });

    expect(setHealth([short, secondary()]).conditions).toEqual([
      { kind: 'oplogShort', member: short.name, seconds: 23 * 3600 },
    ]);
  });

  it('is no data without members', () => {
    expect(setHealth([])).toEqual({ health: 'noData', conditions: [] });
  });

  it.each(['STARTUP', 'STARTUP2', 'RECOVERING', 'UNKNOWN'])(
    'lets a member stay %s for the 10m grace period, then degrades',
    (state) => {
      expect(
        healthOf(setHealth([primary(), secondary(), inState(state, 599)]))
      ).toBe('healthy');

      const stuck = inState(state, 600);

      expect(setHealth([primary(), secondary(), stuck]).conditions).toEqual([
        { kind: 'memberState', member: stuck.name, state, seconds: 600 },
      ]);
    }
  );

  it('degrades a transitional state of unknown age, and any state outside the allow-list at once', () => {
    expect(
      healthOf(setHealth([primary(), secondary(), inState('RECOVERING')]))
    ).toBe('degraded');
    expect(
      healthOf(setHealth([primary(), secondary(), inState('FATAL', 5)]))
    ).toBe('degraded');
  });

  it('lists every condition, the deciding rule first', () => {
    const lagging = named({
      verdict: 'up',
      state: 'SECONDARY',
      lag: { now: 40, sustained: 38 },
    });
    const verdict = setHealth([
      primary(),
      lagging,
      inState('RECOVERING', 900),
      inState('ROLLBACK'),
    ]);

    expect(verdict.health).toBe('down');
    expect(kinds(verdict)).toEqual(['memberLost', 'memberState', 'behind']);
  });
});

describe('member state rules', () => {
  it('reads rolling back and removed as lost', () => {
    expect(isLost({ verdict: 'up', state: 'ROLLBACK' })).toBe(true);
    expect(isLost({ verdict: 'up', state: 'REMOVED' })).toBe(true);
    expect(isLost({ verdict: 'down', state: 'ROLLBACK' })).toBe(false);
    expect(isLost({ verdict: 'up', state: 'RECOVERING' })).toBe(false);
  });

  it('holds primary, secondary and arbiter, and a member without a reported state, as serving', () => {
    ['PRIMARY', 'SECONDARY', 'ARBITER'].forEach((state) =>
      expect(isStuck({ verdict: 'up', state })).toBe(false)
    );
    expect(isStuck({ verdict: 'up' })).toBe(false);
    expect(isStuck({ verdict: 'noData', state: 'RECOVERING' })).toBe(false);
  });
});

describe('hasNoPrimary', () => {
  it('needs an answering member and no primary among them', () => {
    expect(hasNoPrimary([secondary(), down()])).toBe(true);
    expect(hasNoPrimary([primary(), secondary()])).toBe(false);
    expect(hasNoPrimary([down(), down()])).toBe(false);
    expect(hasNoPrimary([noData()])).toBe(false);
  });
});

describe('hasLostQuorum', () => {
  it('needs half the members or more down; silent ones may still vote', () => {
    expect(hasLostQuorum([primary(), down(), down()])).toBe(true);
    expect(hasLostQuorum([primary(), secondary(), down()])).toBe(false);
    expect(hasLostQuorum([primary(), down()])).toBe(true);
    expect(hasLostQuorum([primary(), noData(), noData()])).toBe(false);
  });
});

describe('memberHealth', () => {
  it('is Down or No data by its verdict, else what the member breaks', () => {
    expect(healthOf(memberHealth(down()))).toBe('down');
    expect(healthOf(memberHealth(noData()))).toBe('noData');
    expect(healthOf(memberHealth(inState('ROLLBACK')))).toBe('down');
    expect(healthOf(memberHealth(inState('RECOVERING', 700)))).toBe('degraded');
    expect(healthOf(memberHealth(inState('RECOVERING', 60)))).toBe('healthy');
    expect(healthOf(memberHealth(secondary()))).toBe('healthy');
  });
});

describe('routersHealth', () => {
  it('covers every cell', () => {
    expect(healthOf(routersHealth([secondary(), secondary()]))).toBe('healthy');
    expect(routersHealth([down(), down()], 'Routers')).toEqual({
      health: 'down',
      conditions: [{ kind: 'allDown', total: 2, set: 'Routers' }],
    });
    expect(kinds(routersHealth([secondary(), down()]))).toEqual(['memberDown']);
    expect(kinds(routersHealth([secondary(), noData()]))).toEqual([
      'memberNoData',
    ]);
    expect(healthOf(routersHealth([noData(), noData()]))).toBe('noData');
    expect(healthOf(routersHealth([down(), noData()]))).toBe('noData');
  });
});

describe('shardedHealth', () => {
  it('is down when a shard, the config servers or the routers are down', () => {
    expect(healthOf(shardedHealth({ shards: parts('healthy', 'down') }))).toBe(
      'down'
    );
    expect(
      healthOf(
        shardedHealth({ shards: parts('healthy'), config: parts('down')[0] })
      )
    ).toBe('down');
    expect(
      healthOf(
        shardedHealth({
          shards: parts('healthy'),
          config: parts('healthy')[0],
          routers: parts('down')[0],
        })
      )
    ).toBe('down');
  });

  it('is degraded when a part is degraded or some are silent', () => {
    expect(
      healthOf(shardedHealth({ shards: parts('healthy', 'degraded') }))
    ).toBe('degraded');
    expect(
      healthOf(shardedHealth({ shards: parts('healthy', 'noData') }))
    ).toBe('degraded');
  });

  it('is no data when every part is silent', () => {
    expect(
      healthOf(
        shardedHealth({
          shards: parts('noData'),
          config: parts('noData')[0],
          routers: parts('noData')[0],
        })
      )
    ).toBe('noData');
  });

  it('is healthy when every part is, and gathers their conditions worst first', () => {
    expect(
      healthOf(
        shardedHealth({
          shards: parts('healthy'),
          config: parts('healthy')[0],
          routers: parts('healthy')[0],
        })
      )
    ).toBe('healthy');

    const verdict = shardedHealth({
      shards: [
        setHealth([primary(), secondary(), inState('RECOVERING', 900)]),
        setHealth([secondary(), down()], 'shard2'),
      ],
    });

    expect(verdict.health).toBe('down');
    expect(kinds(verdict)).toEqual(['noPrimary', 'memberState']);
  });
});

describe('standaloneHealth', () => {
  it('maps each verdict', () => {
    expect(standaloneHealth(secondary())).toEqual({
      health: 'healthy',
      conditions: [],
    });
    expect(standaloneHealth(down())).toEqual({
      health: 'down',
      conditions: [{ kind: 'allDown', total: 1 }],
    });
    expect(standaloneHealth(noData())).toEqual({
      health: 'noData',
      conditions: [{ kind: 'noData', total: 1 }],
    });
  });
});

describe('upgradingHealth', () => {
  it('decides before anything else and keeps what else holds', () => {
    const verdict = upgradingHealth(setHealth([primary(), down()]), {
      done: 1,
      total: 3,
      to: '7.0.30-16',
    });

    expect(verdict.health).toBe('upgrading');
    expect(verdict.conditions[0]).toEqual({
      kind: 'upgrading',
      done: 1,
      total: 3,
      to: '7.0.30-16',
    });
    expect(kinds(verdict)).toEqual(['upgrading', 'quorumLost']);
  });
});

describe('lagLevel', () => {
  it('warns from 10s sustained, and has no error level', () => {
    expect(lagLevel(9)).toBe('ok');
    expect(lagLevel(10)).toBe('warning');
    expect(lagLevel(3600)).toBe('warning');
  });
});

describe('oplogLevel', () => {
  it('counts a window under 24h only once the oplog is 90% full', () => {
    expect(oplogLevel(23 * 3600, 0.89)).toBe('ok');
    expect(oplogLevel(23 * 3600, 0.9)).toBe('warning');
    expect(oplogLevel(59 * 60, 0.95)).toBe('warning');
    expect(oplogLevel(59 * 60, undefined)).toBe('ok');
  });

  it('is fine from 24h', () => {
    expect(oplogLevel(24 * 3600, 1)).toBe('ok');
  });
});

describe('worstHealth', () => {
  it('orders Down, No data, Degraded, Upgrading, Healthy', () => {
    expect(worstHealth(['healthy', 'upgrading'])).toBe('upgrading');
    expect(worstHealth(['upgrading', 'degraded'])).toBe('degraded');
    expect(worstHealth(['degraded', 'noData'])).toBe('noData');
    expect(worstHealth(['noData', 'down', 'healthy'])).toBe('down');
    expect(worstHealth([])).toBe('healthy');
  });
});

/** The snapshot as Operations would see it: no series reads as down, roles from metrics. */
const toOmTopology = (snapshot: MongoSnapshot): OmTopologyResponse => {
  const metrics = snapshot.metrics ?? emptyMetrics();
  const groups = new Map<
    string,
    { env: string; name: string; services: OmService[] }
  >();

  (snapshot.services ?? []).forEach((svc) => {
    const name = svc.cluster ?? svc.replicationSet ?? svc.serviceName;
    const key = `${svc.environment}/${name}`;
    const group = groups.get(key) ?? {
      env: svc.environment ?? '',
      name,
      services: [],
    };
    const self = metrics.self[svc.serviceId];
    const router = metrics.up[svc.serviceId]?.clusterRole === 'mongos';

    group.services.push({
      service_name: svc.serviceName,
      service_id: svc.serviceId,
      replication_set: svc.replicationSet ?? null,
      state: self?.state ?? null,
      status:
        metrics.up[svc.serviceId]?.value === 1
          ? 'SERVICE_STATUS_UP'
          : 'SERVICE_STATUS_DOWN',
      process_role: router
        ? 'PROCESS_ROLE_MONGOS'
        : self?.clRole === 'configsvr'
          ? 'PROCESS_ROLE_CONFIGSVR'
          : self?.clRole === 'shardsvr'
            ? 'PROCESS_ROLE_SHARDSVR'
            : 'PROCESS_ROLE_MONGOD',
      cpu_usage_percent: -1,
      connections_free_percent: -1,
    });
    groups.set(key, group);
  });

  const envs = new Map<string, OmCluster[]>();

  groups.forEach((group) => {
    const sets = new Set(
      group.services.map((svc) => svc.replication_set).filter(Boolean)
    );
    const sharded =
      group.services.some(
        (svc) => svc.process_role === 'PROCESS_ROLE_MONGOS'
      ) || sets.size > 1;

    envs.set(group.env, [
      ...(envs.get(group.env) ?? []),
      {
        id: `om-${group.name}`,
        name: group.name,
        type: sharded
          ? 'CLUSTER_TYPE_SHARDED'
          : sets.size === 1
            ? 'CLUSTER_TYPE_REPLICA_SET'
            : 'CLUSTER_TYPE_STANDALONE',
        services: group.services,
      },
    ]);
  });

  return {
    snapshot: { stale: false, schema_version: 1, run_id: 'test' },
    source_queries: [],
    summary: {
      environments: envs.size,
      clusters: groups.size,
      total_services: 0,
      up_services: 0,
      down_services: 0,
      process_role_counts: {},
    },
    environments: [...envs.entries()].map(([env_name, clusters]) => ({
      env_name,
      clusters,
    })),
  };
};

const compare = (snapshot: MongoSnapshot) => {
  const om = toClusterRows(toOmTopology(snapshot));
  const hub = buildClusterTree(
    groupClusters(snapshot.services ?? [], snapshot.metrics, NOW),
    {
      capabilities: { operations: false },
    }
  );

  return om.map((row) => {
    const node = hub.find((item) => item.name === row.cluster_name);
    const members = node?.members ?? { up: 0, total: 0, noData: 0 };

    return {
      cluster: row.cluster_name,
      om: {
        up: row.up_services,
        down: row.down_services,
        total: row.total_services,
        health: row.health,
      },
      hub: {
        up: members.up,
        notUp: members.total - members.up,
        total: members.total,
        health: node?.health,
      },
    };
  });
};

describe('parity with Operations', () => {
  it('agrees on counts and health for the healthy estate', () => {
    compare(healthySnapshot(NOW, false)).forEach(({ om, hub }) => {
      expect(hub.up).toBe(om.up);
      expect(hub.notUp).toBe(om.down);
      expect(hub.total).toBe(om.total);
      expect(om.health).toBe('healthy');
      expect(hub.health).toBe('healthy');
    });
  });

  it('agrees on counts for things wrong, and differs only where the hub has its own rule', () => {
    const rows = compare(wrongSnapshot(NOW, false));
    const byCluster = Object.fromEntries(rows.map((row) => [row.cluster, row]));

    rows.forEach(({ om, hub }) => {
      expect(hub.up).toBe(om.up);
      // Operations counts a member without a series as down; the hub counts it as no data.
      expect(hub.notUp).toBe(om.down);
      expect(hub.total).toBe(om.total);
    });

    // Same verdict where no hub-only rule applies.
    expect(byCluster.orders.om.health).toBe('degraded');
    expect(byCluster.orders.hub.health).toBe('degraded');

    // Absent series: Operations says Down, the hub says No data.
    expect(byCluster.reports.om.health).toBe('down');
    expect(byCluster.reports.hub.health).toBe('noData');

    // No primary: Operations says Degraded, the hub says Down.
    expect(byCluster.billing.om.health).toBe('degraded');
    expect(byCluster.billing.hub.health).toBe('down');
  });

  it('differs on an unreported member: Operations says unknown, the hub says Degraded', () => {
    const snapshot = healthySnapshot(NOW, false);
    const topology = toOmTopology(snapshot);
    const billing = topology.environments
      .flatMap((env) => env.clusters)
      .find((cluster) => cluster.name === 'billing');

    if (!billing) {
      throw new Error('billing missing');
    }
    billing.services[2] = {
      ...billing.services[2],
      status: 'SERVICE_STATUS_UNSPECIFIED',
    };
    const om = toClusterRows(topology).find(
      (row) => row.cluster_name === 'billing'
    );

    const metrics = snapshot.metrics ?? emptyMetrics();

    delete metrics.up[billing.services[2].service_id ?? ''];
    const hub = buildClusterTree(
      groupClusters(snapshot.services ?? [], metrics, NOW),
      {
        capabilities: { operations: false },
      }
    ).find((node) => node.name === 'billing');

    expect(om?.health).toBe('unknown');
    expect(hub?.health).toBe('degraded');
    expect(hub?.members).toEqual({ up: 2, total: 3, noData: 1 });
  });
});
