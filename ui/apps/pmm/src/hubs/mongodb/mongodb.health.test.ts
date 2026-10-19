import type { OmCluster, OmService, OmTopologyResponse } from '@pmm/plugins-om';
// The package exports this same function; its index would load the whole Operations UI here.
import { toClusterRows } from '@pmm/plugins-om/src/topology';
import { healthySnapshot } from './fixtures/healthy';
import { wrongSnapshot } from './fixtures/wrong';
import {
  hasNoPrimary,
  lagLevel,
  memberVerdict,
  oplogLevel,
  routersHealth,
  setHealth,
  shardedHealth,
  standaloneHealth,
  worstHealth,
} from './mongodb.health';
import { buildClusterTree, groupClusters } from './mongodb.topology';
import type { MongoSnapshot } from './mongodb.types';
import { emptyMetrics, NOW, service } from './mongodb.testUtils';

const up = { verdict: 'up' as const, state: 'SECONDARY' };
const primary = { verdict: 'up' as const, state: 'PRIMARY' };
const down = { verdict: 'down' as const };
const noData = { verdict: 'noData' as const };

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
  const healthyLevels = { lagLevels: [], primaryOplogLevel: 'ok' as const };

  it('is healthy with a primary and every member up', () => {
    expect(setHealth({ members: [primary, up, up], ...healthyLevels })).toBe(
      'healthy'
    );
  });

  it('is down when every member is down', () => {
    expect(setHealth({ members: [down, down, down], ...healthyLevels })).toBe(
      'down'
    );
  });

  it('is down when members answer but none is primary', () => {
    expect(setHealth({ members: [up, down, down], ...healthyLevels })).toBe(
      'down'
    );
    expect(setHealth({ members: [up, up, up], ...healthyLevels })).toBe('down');
  });

  it('is degraded when a primary was elected after one went down', () => {
    expect(setHealth({ members: [primary, up, down], ...healthyLevels })).toBe(
      'degraded'
    );
  });

  it('is no data when every member is silent', () => {
    expect(setHealth({ members: [noData, noData], ...healthyLevels })).toBe(
      'noData'
    );
  });

  it('is degraded, never healthy, when some members are silent', () => {
    expect(
      setHealth({ members: [primary, up, noData], ...healthyLevels })
    ).toBe('degraded');
  });

  it('is degraded with nothing up and a mix of down and silent members', () => {
    expect(setHealth({ members: [down, noData], ...healthyLevels })).toBe(
      'degraded'
    );
  });

  it('is degraded when a secondary is behind or the oplog runs short', () => {
    expect(
      setHealth({ members: [primary, up], lagLevels: ['ok', 'warning'] })
    ).toBe('degraded');
    expect(
      setHealth({
        members: [primary, up],
        lagLevels: [],
        primaryOplogLevel: 'error',
      })
    ).toBe('degraded');
  });

  it('is no data without members', () => {
    expect(setHealth({ members: [], lagLevels: [] })).toBe('noData');
  });
});

describe('hasNoPrimary', () => {
  it('needs an answering member and no primary among them', () => {
    expect(hasNoPrimary([up, down])).toBe(true);
    expect(hasNoPrimary([primary, up])).toBe(false);
    expect(hasNoPrimary([down, down])).toBe(false);
    expect(hasNoPrimary([noData])).toBe(false);
  });
});

describe('routersHealth', () => {
  it('covers every cell', () => {
    expect(routersHealth(['up', 'up'])).toBe('healthy');
    expect(routersHealth(['down', 'down'])).toBe('down');
    expect(routersHealth(['up', 'down'])).toBe('degraded');
    expect(routersHealth(['up', 'noData'])).toBe('degraded');
    expect(routersHealth(['noData', 'noData'])).toBe('noData');
  });
});

describe('shardedHealth', () => {
  it('is down when a shard, the config servers or the routers are down', () => {
    expect(shardedHealth({ shards: ['healthy', 'down'] })).toBe('down');
    expect(shardedHealth({ shards: ['healthy'], config: 'down' })).toBe('down');
    expect(
      shardedHealth({ shards: ['healthy'], config: 'healthy', routers: 'down' })
    ).toBe('down');
  });

  it('is degraded when a child is degraded or some are silent', () => {
    expect(shardedHealth({ shards: ['healthy', 'degraded'] })).toBe('degraded');
    expect(shardedHealth({ shards: ['healthy', 'noData'] })).toBe('degraded');
  });

  it('is no data when every child is silent', () => {
    expect(
      shardedHealth({ shards: ['noData'], config: 'noData', routers: 'noData' })
    ).toBe('noData');
  });

  it('is healthy when every child is', () => {
    expect(
      shardedHealth({
        shards: ['healthy'],
        config: 'healthy',
        routers: 'healthy',
      })
    ).toBe('healthy');
  });
});

describe('standaloneHealth', () => {
  it('maps each verdict', () => {
    expect(standaloneHealth('up')).toBe('healthy');
    expect(standaloneHealth('down')).toBe('down');
    expect(standaloneHealth('noData')).toBe('noData');
  });
});

describe('lagLevel', () => {
  it('warns from 30 s sustained and errors from 600 s', () => {
    expect(lagLevel(29)).toBe('ok');
    expect(lagLevel(30)).toBe('warning');
    expect(lagLevel(31)).toBe('warning');
    expect(lagLevel(599)).toBe('warning');
    expect(lagLevel(600)).toBe('error');
    expect(lagLevel(601)).toBe('error');
  });

  it('errors at half the primary’s oplog window', () => {
    expect(lagLevel(100, 300)).toBe('warning');
    expect(lagLevel(150, 300)).toBe('error');
  });
});

describe('oplogLevel', () => {
  it('counts a short window only once the oplog is 90 % full', () => {
    expect(oplogLevel(23 * 3600, 0.89)).toBe('ok');
    expect(oplogLevel(23 * 3600, 0.91)).toBe('warning');
    expect(oplogLevel(59 * 60, 0.95)).toBe('error');
    expect(oplogLevel(59 * 60, undefined)).toBe('ok');
  });

  it('is fine from 24 h', () => {
    expect(oplogLevel(24 * 3600, 1)).toBe('ok');
  });
});

describe('worstHealth', () => {
  it('orders Down, Degraded, No data, Healthy', () => {
    expect(worstHealth(['healthy', 'noData'])).toBe('noData');
    expect(worstHealth(['noData', 'degraded'])).toBe('degraded');
    expect(worstHealth(['degraded', 'down', 'healthy'])).toBe('down');
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
    groupClusters(snapshot.services ?? [], snapshot.metrics),
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
      groupClusters(snapshot.services ?? [], metrics),
      {
        capabilities: { operations: false },
      }
    ).find((node) => node.name === 'billing');

    expect(om?.health).toBe('unknown');
    expect(hub?.health).toBe('degraded');
    expect(hub?.members).toEqual({ up: 2, total: 3, noData: 1 });
  });
});
