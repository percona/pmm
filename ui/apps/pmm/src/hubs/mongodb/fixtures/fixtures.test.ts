import { capAttention } from 'hubs/template/hub.attention';
import type { HubTreeNode } from 'hubs/template/hub.types';
import { MONGODB_ATTENTION_ORDER } from '../mongodb.attention';
import { deriveMongoHub } from '../mongodb.derive';
import { deriveContext, NOW } from '../mongodb.testUtils';
import { MONGODB_FIXTURES } from './index';

const SCENARIOS = ['empty', 'healthy', 'wrong'] as const;
const OPS = [true, false];
const ROLES = ['admin', 'viewer'] as const;

const view = (
  scenario: (typeof SCENARIOS)[number],
  operations: boolean,
  role: (typeof ROLES)[number] = 'admin'
) =>
  deriveMongoHub(
    MONGODB_FIXTURES[scenario](NOW, operations, null),
    deriveContext({ scenario, operations, role })
  );

const flatten = (nodes: HubTreeNode[]): HubTreeNode[] =>
  nodes.flatMap((node) => [node, ...flatten(node.children ?? [])]);

describe('fixtures', () => {
  it.each(
    SCENARIOS.flatMap((scenario) =>
      OPS.flatMap((ops) => ROLES.map((role) => [scenario, ops, role] as const))
    )
  )('%s with Operations %s as %s derives a view', (scenario, ops, role) => {
    const result = view(scenario, ops, role);

    expect(result.technology).toEqual({ id: 'mongodb', name: 'MongoDB' });
    expect(result.scenario).toBe(scenario);
    expect(
      Object.values(result.sections).every((s) => s.status === 'ready')
    ).toBe(true);
  });

  describe('healthy', () => {
    it('has no attention item and no node that is not healthy', () => {
      OPS.forEach((ops) => {
        const result = view('healthy', ops);

        expect(result.attention).toEqual([]);
        expect(
          flatten(result.clusters).filter((node) => node.health !== 'healthy')
        ).toEqual([]);
        expect(result.expanded).toEqual({});
      });
    });

    it('matches the design: 5 clusters, none need attention, last good backup 6h ago', () => {
      const result = view('healthy', true);

      expect(result.status).toMatchObject({
        state: 'healthy',
        clusters: 5,
        needAttention: 0,
        lastGoodBackup: {
          at: new Date(NOW - 6 * 3600_000).toISOString(),
          cluster: 'billing',
        },
      });
      expect(result.backups.rows.map((row) => [row.cluster, row.pitr])).toEqual(
        [
          ['billing', 'on'],
          ['orders', 'off'],
          ['payments', 'off'],
          ['catalog', 'off'],
          ['reports', 'notApplicable'],
        ]
      );
      expect(result.queries.slower).toEqual([]);
      expect(result.queries.runningLong).toEqual([]);
      expect(result.dashboards.recommended.map((link) => link.label)).toContain(
        'Backup status'
      );
      expect(result.apps[0].summary?.text).toBe(
        '21 of 21 services up · last scan 4m ago'
      );
    });
  });

  describe('wrong', () => {
    it('has at least one item of every kind but a lost member, which unit tests cover', () => {
      const kinds = new Set(
        view('wrong', true).attention.map((item) => item.kind)
      );

      expect([...kinds].sort()).toEqual(
        MONGODB_ATTENTION_ORDER.filter((kind) => kind !== 'memberLost').sort()
      );
    });

    it('ranks every kind, 14 rows with Operations off and 15 on', () => {
      const off = view('wrong', false).attention;
      const on = view('wrong', true).attention;
      const steady = [
        'noPrimary',
        'memberDown',
        'memberNoData',
        'backupFailed',
        'pitrStopped',
        'oplogShort',
        'memberNotServing',
      ];
      const after = ['behind', 'alert', 'alert', 'querySlower', 'queryRunning'];

      // Without Operations nothing says payments is upgrading, so its mixed versions read as drift too.
      expect(off.map((item) => [item.kind, item.where])).toEqual(
        expect.arrayContaining([
          ['versionDrift', 'payments · prod'],
          ['versionDrift', 'catalog · staging'],
        ])
      );
      expect(off.map((item) => item.kind)).toEqual([
        ...steady,
        'versionDrift',
        'versionDrift',
        ...after,
      ]);
      expect(on.map((item) => item.kind)).toEqual([
        ...steady,
        'versionDrift',
        ...after,
        'operationsFailed',
        'operationsFailed',
      ]);
      expect(
        on
          .slice(13)
          .every((item) => item.door.to.startsWith('/operations/automations'))
      ).toBe(true);
    });

    it('shows the first five and folds 9 more off, 10 more on', () => {
      const off = capAttention(view('wrong', false).attention, 5);
      const on = capAttention(view('wrong', true).attention, 5);

      expect(off.visible.map((item) => item.kind)).toEqual([
        'noPrimary',
        'memberDown',
        'memberNoData',
        'backupFailed',
        'pitrStopped',
      ]);
      expect(off.hidden).toHaveLength(9);
      expect(on.hidden).toHaveLength(10);
    });

    it('matches the design’s tree and status line', () => {
      const result = view('wrong', false);
      const [billing, reports, , orders] = result.clusters;

      expect(result.clusters.map((node) => node.name)).toEqual([
        'billing',
        'reports',
        'catalog',
        'orders',
        'payments',
      ]);
      expect(result.status).toMatchObject({
        state: 'down',
        clusters: 5,
        needAttention: 5,
        lastGoodBackup: {
          at: new Date(NOW - 6 * 3600_000).toISOString(),
          cluster: 'billing',
        },
      });
      expect([billing.name, billing.health, billing.members]).toEqual([
        'billing',
        'down',
        { up: 1, total: 3, noData: 0 },
      ]);
      expect([orders.name, orders.health, orders.members]).toEqual([
        'orders',
        'degraded',
        { up: 10, total: 11, noData: 0 },
      ]);
      expect(orders.measures.lag).toMatchObject({
        seconds: 40,
        level: 'warning',
      });
      expect(orders.measures.oplogWindow).toMatchObject({
        seconds: 10800,
        level: 'warning',
      });
      expect([reports.name, reports.health, reports.members]).toEqual([
        'reports',
        'noData',
        { up: 0, total: 1, noData: 1 },
      ]);
      expect(result.queries.slower[0].headline).toBe('6× slower than usual');
      expect(result.queries.runningLong[0].headline).toBe('running for 14m');
      expect(view('wrong', true).apps[0].summary?.text).toBe(
        '17 of 21 services up · last scan 4m ago'
      );
      expect(result.apps[0].summary?.text).toBe(
        '4 schedules · 1 failed in the last 24h'
      );
    });

    it('reads payments as Upgrading while Operations runs its upgrade, and keeps it out of the count', () => {
      const on = view('wrong', true);
      const payments = on.clusters.find((node) => node.name === 'payments');

      expect(payments).toMatchObject({
        health: 'upgrading',
        since: new Date(NOW - 9 * 60_000).toISOString(),
      });
      // billing, reports, orders and catalog; payments' mixed versions are its upgrade's.
      expect(on.status.needAttention).toBe(4);
      expect(on.expanded['cluster:prod/payments']).toBeUndefined();
      // Without Operations nothing says an upgrade runs: the restarted member is inside its grace period, but the versions differ.
      expect(
        view('wrong', false).clusters.find((node) => node.name === 'payments')
          ?.health
      ).toBe('degraded');
    });

    it('narrows queries, Operations counts and installs to the environment like live data', () => {
      const staging = deriveMongoHub(
        MONGODB_FIXTURES.wrong(NOW, true, 'staging'),
        deriveContext({ operations: true, environment: 'staging' })
      );

      expect(staging.clusters.map((node) => node.name)).toEqual([
        'reports',
        'catalog',
      ]);
      expect(staging.queries.slower).toEqual([]);
      expect(staging.apps[0].summary?.text).toBe(
        '3 of 4 services up · last scan 4m ago'
      );
      expect(staging.attention.map((item) => item.title)).toEqual([
        'PMM gets no data from reports',
        'catalog runs mixed versions, an upgrade may have stopped partway',
        'Scan failed on 2 nodes',
      ]);
    });

    it('reads the orders backup as failed after an old success', () => {
      const orders = view('wrong', false).backups.rows.find(
        (row) => row.cluster === 'orders'
      );

      expect(orders).toMatchObject({
        failedAt: new Date(NOW - 8 * 3600_000).toISOString(),
        lastGood: {
          at: new Date(NOW - 32 * 3600_000).toISOString(),
          level: 'warning',
        },
      });
    });
  });

  describe('empty', () => {
    it('has no clusters, no services to back up and only the dashboards door', () => {
      OPS.forEach((ops) => {
        const result = view('empty', ops);

        expect(result.clusters).toEqual([]);
        expect(result.status.state).toBe('empty');
        expect(result.backups.state).toBe('noServices');
        expect(result.dashboards.recommended).toEqual([]);
        expect(result.dashboards.recent).toEqual([]);
        expect(result.dashboards.all.count).toBe(13);
      });
    });

    it('offers both actions to an admin with Operations, Add only without, none to viewers', () => {
      const labels = (ops: boolean, role?: 'viewer') =>
        view('empty', ops, role).actions.add.map((action) => action.label);

      expect(labels(true)).toEqual([
        'Monitor an existing service',
        'Install new cluster',
      ]);
      expect(labels(false)).toEqual(['Monitor an existing service']);
      expect(labels(true, 'viewer')).toEqual([]);
    });

    it('points the Operations card at Nodes', () => {
      expect(view('empty', true).apps[0].summary?.text).toBe(
        'No MongoDB yet. Install one from Nodes.'
      );
    });
  });
});
