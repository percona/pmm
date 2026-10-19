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
    MONGODB_FIXTURES[scenario](NOW, operations),
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

    it('matches the design: 3 clusters, none need attention, last good backup 6h ago', () => {
      const result = view('healthy', true);

      expect(result.status).toMatchObject({
        state: 'healthy',
        clusters: 3,
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
          ['reports', 'notApplicable'],
        ]
      );
      expect(result.queries.slower).toEqual([]);
      expect(result.queries.runningLong).toEqual([]);
      expect(result.dashboards.recommended.map((link) => link.label)).toContain(
        'Backup status'
      );
      expect(result.apps[0].summary?.text).toBe(
        '15 of 15 services up · last scan 4m ago'
      );
    });
  });

  describe('wrong', () => {
    it('has at least one item of every kind', () => {
      const kinds = new Set(
        view('wrong', true).attention.map((item) => item.kind)
      );

      expect([...kinds].sort()).toEqual([...MONGODB_ATTENTION_ORDER].sort());
    });

    it('ranks exactly as the design, 10 rows with Operations off and 12 on', () => {
      const off = view('wrong', false).attention;
      const on = view('wrong', true).attention;

      expect(off.map((item) => item.kind)).toEqual([
        'noPrimary',
        'memberDown',
        'memberNoData',
        'backupFailed',
        'pitrStopped',
        'oplogShort',
        'behind',
        'alert',
        'querySlower',
        'queryRunning',
      ]);
      expect(on.map((item) => item.kind)).toEqual([
        ...off.map((item) => item.kind),
        'operationsFailed',
        'operationsFailed',
      ]);
      expect(
        on.slice(10).every((item) => item.door.preview === 'developer')
      ).toBe(true);
    });

    it('shows the first five and folds 5 more off, 7 more on', () => {
      const off = capAttention(view('wrong', false).attention, 5);
      const on = capAttention(view('wrong', true).attention, 5);

      expect(off.visible.map((item) => item.kind)).toEqual([
        'noPrimary',
        'memberDown',
        'memberNoData',
        'backupFailed',
        'pitrStopped',
      ]);
      expect(off.hidden).toHaveLength(5);
      expect(on.hidden).toHaveLength(7);
    });

    it('matches the design’s tree and status line', () => {
      const result = view('wrong', false);
      const [billing, orders, reports] = result.clusters;

      expect(result.status).toMatchObject({
        state: 'down',
        clusters: 3,
        needAttention: 3,
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
        '11 of 15 services up · last scan 4m ago'
      );
      expect(result.apps[0].summary?.text).toBe(
        '3 schedules · 1 failed in the last 24h'
      );
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
      expect(Object.keys(view('empty', true).actions)).toEqual([
        'addService',
        'install',
      ]);
      expect(Object.keys(view('empty', false).actions)).toEqual(['addService']);
      expect(view('empty', true, 'viewer').actions).toEqual({});
    });

    it('points the Operations card at Nodes', () => {
      expect(view('empty', true).apps[0].summary?.text).toBe(
        'No MongoDB yet. Install one from Nodes.'
      );
    });
  });
});
