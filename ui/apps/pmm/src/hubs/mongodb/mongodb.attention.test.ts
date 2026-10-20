import { rankAttention } from 'hubs/template/hub.attention';
import { ALERTS_SERVICE_FILTER_KEY } from 'hubs/template/hub.doors';
import type { HubAttentionItem } from 'hubs/template/hub.types';
import { wrongSnapshot } from './fixtures/wrong';
import {
  alertSeverity,
  type AttentionInput,
  deriveAttention,
  MONGODB_ATTENTION_ORDER,
} from './mongodb.attention';
import { deriveBackups } from './mongodb.backups';
import { deriveQueries } from './mongodb.queries';
import { groupClusters } from './mongodb.topology';
import type {
  MongoAlertInput,
  MongoMetrics,
  MongoServiceInput,
} from './mongodb.types';
import { mergeMetrics, NOW, replicaSet, service } from './mongodb.testUtils';

const input = (
  services: MongoServiceInput[],
  metrics: MongoMetrics,
  overrides: Partial<AttentionInput> = {}
): AttentionInput => ({
  clusters: groupClusters(services, metrics, NOW),
  alerts: [],
  backupFindings: [],
  slower: [],
  running: [],
  operations: null,
  filter: null,
  now: NOW,
  ...overrides,
});

const items = (attention: AttentionInput): HubAttentionItem[] =>
  deriveAttention(attention).map((entry) => entry.item);

const kinds = (attention: AttentionInput) =>
  items(attention).map((item) => item.kind);

// The oplog is half full, so a short window raises no row of its own.
const lagging = (lag: number, sustained = lag, window?: number) => {
  const set = replicaSet('rs', { window: window ?? 40 * 3600, fill: 0.5 });

  set.metrics.lag[set.services[2].serviceId] = { now: lag, sustained };

  return input(set.services, set.metrics);
};

const shortOplog = (window: number, fill: number) => {
  const set = replicaSet('rs', { window, fill });

  return input(set.services, set.metrics);
};

describe('behind', () => {
  it('flags sustained lag from 10s as a warning', () => {
    expect(kinds(lagging(9))).toEqual([]);
    expect(items(lagging(11))).toMatchObject([
      {
        kind: 'behind',
        severity: 'warning',
        title: 'rs-m3 is 11s behind its primary',
        detail: '11s sustained for 2m',
      },
    ]);
  });

  it('stays a warning however far behind, as PMM-15299 has it', () => {
    expect(items(lagging(601))[0].severity).toBe('warning');
    expect(items(lagging(200, 200, 300))[0].severity).toBe('warning');
  });

  it('ignores a momentary spike that is not sustained', () => {
    expect(kinds(lagging(700, 5))).toEqual([]);
  });

  it('opens the set’s ReplSet summary', () => {
    expect(items(lagging(40))[0].door).toEqual({
      label: 'Open ReplSet summary',
      to: '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-cluster=rs&var-rs_nm=rs',
      context: 'for rs',
    });
  });
});

describe('oplogShort', () => {
  it('flags a window under 24 h only once the oplog is 90 % full', () => {
    expect(kinds(shortOplog(23 * 3600, 0.89))).toEqual([]);
    expect(items(shortOplog(23 * 3600, 0.91))).toMatchObject([
      {
        kind: 'oplogShort',
        severity: 'warning',
        title: 'rs oplog window is 23h and the oplog is full',
        detail: '91% full',
      },
    ]);
  });

  it('stays a warning under an hour', () => {
    expect(items(shortOplog(59 * 60, 0.95))).toMatchObject([
      { kind: 'oplogShort', severity: 'warning' },
    ]);
  });

  it('opens Oplog details on the set’s primary', () => {
    expect(items(shortOplog(3600 * 3, 0.95))[0].door.to).toBe(
      '/graph/d/mongodb-oplog-details/mongodb-oplog-details?var-environment=prod&var-cluster=rs&var-replset=rs&var-service_name=rs-m1'
    );
  });
});

describe('member rows', () => {
  it('folds the down members of a set without a primary into one error', () => {
    const set = replicaSet('billing');
    const [m1, m2] = set.services;

    set.metrics.up[m1.serviceId] = { value: 0, clusterRole: 'mongod' };
    set.metrics.up[m2.serviceId] = { value: 0, clusterRole: 'mongod' };
    set.metrics.lastUpAt[m1.serviceId] = (NOW - 18 * 60_000) / 1000;
    set.metrics.lastUpAt[m2.serviceId] = (NOW - 17 * 60_000) / 1000;

    const result = items(input(set.services, set.metrics));

    expect(result).toEqual([
      {
        id: 'noPrimary:prod/billing/billing',
        kind: 'noPrimary',
        severity: 'error',
        title: "billing has no primary, it can't take writes",
        where: 'billing · prod',
        detail: '2 of 3 members down',
        since: new Date(NOW - 18 * 60_000).toISOString(),
        door: {
          label: 'Open ReplSet summary',
          to: `/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-cluster=billing&var-rs_nm=billing&from=${NOW - 48 * 60_000}&to=now`,
          context: 'for billing',
        },
      },
    ]);
  });

  it('makes a down secondary of a healthy set a warning', () => {
    const set = replicaSet('rs');

    set.metrics.up[set.services[2].serviceId] = {
      value: 0,
      clusterRole: 'mongod',
    };

    expect(items(input(set.services, set.metrics))).toMatchObject([
      { kind: 'memberDown', severity: 'warning', title: 'rs-m3 is down' },
    ]);
  });

  it('makes a down standalone an error', () => {
    const lone = service('lone', { cluster: 'lone' });
    const metrics = mergeMetrics();

    metrics.up[lone.serviceId] = { value: 0, clusterRole: 'mongod' };

    expect(items(input([lone], metrics))).toMatchObject([
      { kind: 'memberDown', severity: 'error', where: 'lone · prod' },
    ]);
  });

  it('reports a member PMM gets no data from as a warning with its door framed on the incident', () => {
    const lone = service('lone', { cluster: 'lone' });
    const metrics = mergeMetrics();

    metrics.lastUpAt[lone.serviceId] = (NOW - 9 * 60_000) / 1000;

    expect(items(input([lone], metrics))).toEqual([
      {
        id: `memberNoData:${lone.serviceId}`,
        kind: 'memberNoData',
        severity: 'warning',
        title: 'PMM gets no data from lone',
        where: 'lone · prod',
        since: new Date(NOW - 9 * 60_000).toISOString(),
        door: {
          label: 'Open Instance summary',
          to: `/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-environment=prod&var-cluster=lone&var-service_name=lone&from=${NOW - 39 * 60_000}&to=now`,
          context: 'for lone',
        },
      },
    ]);
  });
});

describe('members that are up but not serving', () => {
  const inState = (state: string, minutes = 60) => {
    const set = replicaSet('rs');
    const m3 = set.services[2].serviceId;

    set.metrics.self[m3] = { ...set.metrics.self[m3], state };
    set.metrics.stateSince[m3] = (NOW - minutes * 60_000) / 1000;
    delete set.metrics.lag[m3];

    return { set, m3 };
  };

  it.each([
    ['RECOVERING', 'recovering'],
    ['STARTUP2', 'in initial sync'],
  ])(
    'warns about a member %s past the grace period and opens its set',
    (state, phrase) => {
      const { set, m3 } = inState(state);

      expect(items(input(set.services, set.metrics))).toEqual([
        {
          id: `memberNotServing:${m3}`,
          kind: 'memberNotServing',
          severity: 'warning',
          title: `rs-m3 has been ${phrase} for 1h, it can't serve reads`,
          where: 'rs · prod',
          door: {
            label: 'Open ReplSet summary',
            to: '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-cluster=rs&var-rs_nm=rs',
            context: 'for rs',
          },
        },
      ]);
    }
  );

  it('says nothing about a member inside the 10m grace period', () => {
    const { set } = inState('RECOVERING', 9);

    expect(items(input(set.services, set.metrics))).toEqual([]);
  });

  it('makes a member rolling back an error, since its set is down', () => {
    const { set, m3 } = inState('ROLLBACK', 1);

    expect(items(input(set.services, set.metrics))).toMatchObject([
      {
        id: `memberLost:${m3}`,
        kind: 'memberLost',
        severity: 'error',
        title: "rs-m3 is rolling back, it holds none of the set's data",
      },
    ]);
  });

  it('merges the unusual state alert of its set', () => {
    const set = replicaSet('rs');
    const m3 = set.services[2].serviceId;

    set.metrics.self[m3] = { ...set.metrics.self[m3], state: 'RECOVERING' };

    expect(
      items(
        input(set.services, set.metrics, {
          alerts: [
            {
              name: 'MongoDB member is in Unusual state',
              templateName: 'pmm_mongodb_unusual_state',
              severity: 'critical',
              cluster: 'rs',
              replicaSet: 'rs',
              activeAt: new Date(NOW).toISOString(),
            },
          ],
        })
      )
    ).toMatchObject([
      { kind: 'memberNotServing', severity: 'error', source: 'alert' },
    ]);
  });
});

describe('alerts', () => {
  const downSecondary = () => {
    const set = replicaSet('rs');

    set.metrics.up[set.services[2].serviceId] = {
      value: 0,
      clusterRole: 'mongod',
    };

    return set;
  };
  const alert = (overrides: Partial<MongoAlertInput>): MongoAlertInput => ({
    name: 'MongoDB down',
    templateName: 'pmm_mongodb_down',
    severity: 'critical',
    serviceName: 'rs-m3',
    cluster: 'rs',
    activeAt: new Date(NOW - 60_000).toISOString(),
    summary: 'MongoDB down (rs-m3)',
    ...overrides,
  });

  it('merges a matching alert into its row, adds the Alert door and takes the higher severity', () => {
    const set = downSecondary();
    const [row] = items(
      input(set.services, set.metrics, { alerts: [alert({})] })
    );

    expect(row).toMatchObject({
      kind: 'memberDown',
      severity: 'error',
      source: 'alert',
      secondaryDoor: {
        label: 'Alert',
        to: `/alerting/status?${ALERTS_SERVICE_FILTER_KEY}=rs-m3`,
      },
    });
  });

  it('keeps the derived severity when the alert is milder', () => {
    const set = replicaSet('rs', { members: 4 });

    set.metrics.up[set.services[2].serviceId] = {
      value: 0,
      clusterRole: 'mongod',
    };
    set.metrics.up[set.services[3].serviceId] = {
      value: 0,
      clusterRole: 'mongod',
    };
    const [row] = items(
      input(set.services, set.metrics, {
        alerts: [
          alert({
            templateName: 'pmm_mongodb_down',
            severity: 'warning',
            serviceName: 'rs-m3',
          }),
        ],
      })
    );

    // Half the set is down, so it lost its majority: the rows are errors whatever the alert says.
    expect(row).toMatchObject({
      kind: 'memberDown',
      severity: 'error',
      source: 'alert',
    });
  });

  it('merges a no-primary alert by replica set', () => {
    const set = replicaSet('rs');

    set.metrics.self[set.services[0].serviceId] = {
      rsName: 'rs',
      state: 'SECONDARY',
    };
    const result = items(
      input(set.services, set.metrics, {
        alerts: [
          alert({
            templateName: 'pmm_mongodb_replset_no_primary',
            serviceName: undefined,
            replicaSet: 'rs',
          }),
        ],
      })
    );

    expect(result.map((item) => [item.kind, item.source])).toEqual([
      ['noPrimary', 'alert'],
    ]);
  });

  it('makes an alert with nothing to merge into its own row', () => {
    const set = replicaSet('rs');
    const result = items(
      input(set.services, set.metrics, {
        alerts: [
          alert({
            name: 'MongoDB restarted',
            templateName: 'pmm_mongodb_restarted',
            severity: 'warning',
            serviceName: 'rs-m1',
            summary: 'MongoDB restarted (rs-m1)',
          }),
        ],
      })
    );

    expect(result).toEqual([
      {
        id: 'alert:pmm_mongodb_restarted:rs-m1',
        kind: 'alert',
        severity: 'warning',
        title: 'MongoDB restarted (rs-m1)',
        where: 'rs · prod',
        since: new Date(NOW - 60_000).toISOString(),
        door: {
          label: 'Open Alerts',
          to: `/alerting/status?${ALERTS_SERVICE_FILTER_KEY}=rs-m1`,
          context: 'for rs-m1',
        },
        source: 'alert',
      },
    ]);
  });

  it('does not merge an alert about another cluster', () => {
    const set = downSecondary();
    const result = items(
      input(set.services, set.metrics, {
        alerts: [alert({ cluster: 'other' })],
      })
    );

    expect(result.map((item) => item.kind)).toEqual(['memberDown', 'alert']);
  });

  it('maps alert severities', () => {
    expect(alertSeverity('critical')).toBe('error');
    expect(alertSeverity('emergency')).toBe('error');
    expect(alertSeverity('alert')).toBe('error');
    expect(alertSeverity('error')).toBe('error');
    expect(alertSeverity('warning')).toBe('warning');
    expect(alertSeverity('notice')).toBe('info');
    expect(alertSeverity(undefined)).toBe('info');
  });
});

describe('operations', () => {
  const failedRuns = [
    {
      id: 'r1',
      kind: 'install' as const,
      target: 'pmm-client',
      finishedAt: new Date(NOW - 20 * 60_000).toISOString(),
    },
    {
      id: 'r2',
      kind: 'scan' as const,
      target: '',
      finishedAt: new Date(NOW - 25 * 60_000).toISOString(),
      failingNodes: 2,
    },
    {
      id: 'r3',
      kind: 'scan' as const,
      target: '',
      finishedAt: new Date(NOW - 25 * 3600_000).toISOString(),
    },
  ];

  it('reports failed installs and scans of the last 24 h, with Preview doors', () => {
    const result = items(
      input([], mergeMetrics(), {
        operations: {
          clusterIds: {},
          summary: { services: 0, up: 0 },
          upgrades: [],
          failedRuns,
        },
      })
    );

    expect(result.map((item) => item.title)).toEqual([
      'Install of MongoDB on pmm-client failed',
      'Scan failed on 2 nodes',
    ]);
    expect(result[0]).toMatchObject({
      kind: 'operationsFailed',
      source: 'operations',
      door: {
        to: '/operations/automations?tab=installs&expand=r1',
        preview: 'developer',
      },
    });
  });

  it('reports nothing from Operations when it is off', () => {
    expect(kinds(input([], mergeMetrics()))).toEqual([]);
  });
});

describe('the things-wrong estate', () => {
  it('gives exactly the ranked list of the design', () => {
    const snapshot = wrongSnapshot(NOW, true);
    const services = snapshot.services ?? [];
    // As the derivation does: Operations' upgrade of payments rides on its cluster.
    const clusters = groupClusters(services, snapshot.metrics, NOW).map(
      (cluster) => ({
        ...cluster,
        upgrade: snapshot.operations?.upgrades.find(
          (upgrade) => upgrade.clusterKey === cluster.key
        ),
      })
    );
    const backups = deriveBackups({
      clusters,
      input: snapshot.backups,
      metrics: snapshot.metrics,
      capabilities: { setUpBackup: true },
      role: 'admin',
      filter: null,
      now: NOW,
    });
    const queries = deriveQueries(snapshot, services, null);
    const ranked = rankAttention(
      deriveAttention({
        clusters,
        alerts: snapshot.alerts ?? [],
        backupFindings: backups.findings,
        slower: queries.slower,
        running: queries.running,
        operations: snapshot.operations,
        filter: null,
        now: NOW,
      }).map((entry) => entry.item),
      MONGODB_ATTENTION_ORDER
    );

    expect(
      ranked.map((item) => [item.kind, item.severity, item.title])
    ).toEqual([
      ['noPrimary', 'error', "billing has no primary, it can't take writes"],
      ['memberDown', 'error', 'orders-shard2-m2 is down'],
      ['memberNoData', 'warning', 'PMM gets no data from reports'],
      ['backupFailed', 'warning', 'orders nightly backup failed'],
      ['pitrStopped', 'warning', 'billing point-in-time recovery stopped'],
      [
        'oplogShort',
        'warning',
        'orders-shard2 oplog window is 3h and the oplog is full',
      ],
      [
        'memberNotServing',
        'warning',
        "orders-cfg-m3 has been recovering for 14m, it can't serve reads",
      ],
      [
        'versionDrift',
        'warning',
        'catalog runs mixed versions, an upgrade may have stopped partway',
      ],
      ['behind', 'warning', 'orders-shard1-m3 is 40s behind its primary'],
      ['alert', 'warning', 'MongoDB restarted (orders-shard1-m1)'],
      ['alert', 'warning', 'Node high CPU load (orders-shard2-m1)'],
      [
        'querySlower',
        'warning',
        'db.orders.find({status: ?, createdAt: ?}) · 6× slower than usual',
      ],
      ['queryRunning', 'warning', 'aggregate on shop.orders · running for 14m'],
      ['operationsFailed', 'warning', 'Scan failed on 2 nodes'],
      [
        'operationsFailed',
        'warning',
        'Install of MongoDB on pmm-client failed',
      ],
    ]);
    expect(ranked[1].secondaryDoor?.label).toBe('Alert');
    expect(ranked[1].where).toBe('orders › orders-shard2 · prod');
    expect(new Set(ranked.map((item) => item.id)).size).toBe(ranked.length);
  });

  it('keeps ids stable across refetches', () => {
    const ids = (now: number) => {
      const snapshot = wrongSnapshot(now, false);

      return deriveAttention({
        clusters: groupClusters(snapshot.services ?? [], snapshot.metrics),
        alerts: snapshot.alerts ?? [],
        backupFindings: [],
        slower: [],
        running: [],
        operations: null,
        filter: null,
        now,
      }).map((entry) => entry.item.id);
    };

    expect(ids(NOW + 30_000)).toEqual(ids(NOW));
  });
});
