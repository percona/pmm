import { attentionIssues } from 'hubs/template/hub.attention';
import { statusFacets } from 'hubs/template/hub.status';
import type { HubTreeNode, HubViewModel } from 'hubs/template/hub.types';
import { deriveMongoHub } from '../mongodb.derive';
import { describeMongoDestination } from '../mongodb.destinations';
import { deriveContext, NOW, sourcesWith } from '../mongodb.testUtils';
import { MONGODB_FIXTURES } from './index';

const SCENARIOS = ['empty', 'healthy', 'incomplete', 'unhealthy'] as const;
const OPS = [true, false];
const ROLES = ['admin', 'viewer'] as const;

const view = (
  scenario: (typeof SCENARIOS)[number],
  operations: boolean,
  role: (typeof ROLES)[number] = 'admin',
  environment: string | null = null
) =>
  deriveMongoHub(
    MONGODB_FIXTURES[scenario](NOW, operations, environment),
    deriveContext({
      scenario,
      operations,
      role,
      environment,
      sources: sourcesWith('ready'),
    })
  );

const flatten = (nodes: HubTreeNode[]): HubTreeNode[] =>
  nodes.flatMap((node) => [node, ...flatten(node.children ?? [])]);

/** The status banner as it reads: each facet's label, and its action's. */
const banner = (result: HubViewModel) =>
  statusFacets(result.status, {
    name: 'MongoDB',
    unit: ['cluster', 'clusters'],
    loading: false,
    matchesNothing: false,
    issues: attentionIssues(result.attention),
  }).map(({ label, action }) => (action ? [label, action.label] : [label]));

describe('fixtures', () => {
  // The banner says ages against the clock, so it reads the sample data's now.
  beforeAll(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(NOW);
  });
  afterAll(() => {
    vi.useRealTimers();
  });

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
    // Every cluster starts as one row.
    expect(result.expanded).toEqual({});
  });

  it('runs every scenario on the 60-cluster fleet', () => {
    (['healthy', 'incomplete', 'unhealthy'] as const).forEach((scenario) => {
      expect(view(scenario, true).clusters).toHaveLength(60);
    });
  });

  it('writes every row in the requirements format, with a destination the sidebar has an icon for', () => {
    (['incomplete', 'unhealthy'] as const).forEach((scenario) =>
      OPS.forEach((ops) =>
        view(scenario, ops).attention.forEach((item) => {
          expect(Object.keys(item).sort()).toEqual(
            [
              'severity',
              'title',
              'impact',
              'location',
              'environment',
              'since',
              'issueCount',
              'alertFiring',
              ...(item.alertService ? ['alertService'] : []),
              'source',
              'destination',
            ].sort()
          );
          expect(describeMongoDestination(item.destination).icon).toBeTruthy();
        })
      )
    );
  });

  it('sends each grouped row to the Clusters page, filtered to clusters that are not healthy', () => {
    (['incomplete', 'unhealthy'] as const).forEach((scenario) => {
      const result = view(scenario, true);
      const health = new Map(
        result.clusters.map((node) => [node.name, node.health])
      );
      const affected = result.attention.filter(
        (item) => item.destination.label === 'Affected clusters'
      );

      expect(affected.length).toBeGreaterThan(0);
      affected.forEach(({ destination }) => {
        expect(destination).toMatchObject({
          route: '/mongodb/clusters',
          kind: 'page',
        });
        expect(describeMongoDestination(destination).path).toEqual([
          'MongoDB',
          'Clusters, filtered',
        ]);
        destination.params?.clusters.split(',').forEach((cluster) => {
          expect([cluster, health.get(cluster)]).not.toEqual([
            cluster,
            'healthy',
          ]);
          expect(health.has(cluster)).toBe(true);
        });
      });
    });
  });

  describe('healthy', () => {
    it('reads as the Healthy banner, with nothing that needs attention', () => {
      OPS.forEach((ops) => {
        const result = view('healthy', ops);

        expect(banner(result)).toEqual([
          ['All 60 clusters are healthy'],
          ['No alerts firing'],
          ['Backups on schedule, last 25m ago'],
        ]);
        expect(result.attention).toEqual([]);
        expect(
          flatten(result.clusters).filter((node) => node.health !== 'healthy')
        ).toEqual([]);
      });
    });
  });

  describe('incomplete', () => {
    it('reads as the Incomplete banner', () => {
      const result = view('incomplete', true);

      expect(banner(result)).toEqual([
        ['12 of 60 clusters degraded', 'Review 15 issues'],
        ['No alerts set up', 'Set up alerts'],
        ['41 of 60 clusters protected', 'Set up backups'],
      ]);
      expect(result.status.health).toEqual({
        healthy: 48,
        degraded: 10,
        down: 0,
        upgrading: 0,
        noData: 2,
      });
    });

    it('lists 15 issues in 6 warning rows, none with an alert, as none is set up', () => {
      const { attention } = view('incomplete', true);

      expect(attention).toHaveLength(6);
      expect(attentionIssues(attention)).toBe(15);
      expect(attention.every((item) => item.severity === 'warning')).toBe(true);
      expect(attention.some((item) => item.alertFiring)).toBe(false);
    });
  });

  describe('unhealthy', () => {
    it('reads as the Unhealthy banner', () => {
      const result = view('unhealthy', true);

      expect(banner(result)).toEqual([
        ['23 of 60 clusters need attention', 'Review 39 issues'],
        ['17 alerts firing, 4 critical', 'Open alerts'],
        ['3 backups failed, oldest good backup 1d 8h ago', 'Open backups'],
      ]);
      expect(result.status.health).toEqual({
        healthy: 37,
        degraded: 15,
        down: 4,
        upgrading: 0,
        noData: 4,
      });
    });

    it('lists 39 issues in 12 rows, critical ones first and each on its own', () => {
      const { attention } = view('unhealthy', true);

      expect(attention).toHaveLength(12);
      expect(attentionIssues(attention)).toBe(39);
      expect(attention.map((item) => item.severity)).toEqual([
        ...Array(4).fill('critical'),
        ...Array(2).fill('error'),
        ...Array(6).fill('warning'),
      ]);
      expect(
        attention
          .filter((item) => item.severity === 'critical')
          .every((item) => item.issueCount === 1)
      ).toBe(true);
    });

    it('sends mixed versions to Fleet, or to Instances overview while Operations is off', () => {
      const drift = (ops: boolean) =>
        view('unhealthy', ops).attention.find((item) =>
          item.title.includes('mixed versions')
        )?.destination;

      expect(drift(true)).toEqual({
        label: 'Fleet',
        route: '/operations',
        kind: 'page',
      });
      expect(drift(false)).toEqual({
        label: 'Instances overview',
        route: '/graph/d/mongodb-instance-overview/mongodb-instances-overview',
        kind: 'page',
      });
    });

    it('keeps the rows of one environment, a row that spans two in both', () => {
      const staging = view('unhealthy', true, 'admin', 'staging');

      expect(staging.clusters).toHaveLength(9);
      expect(staging.attention.map((item) => item.title)).toEqual([
        'PMM gets no data from 4 clusters',
        'inventory-apac runs mixed versions',
      ]);
    });

    it('reads a failed nightly backup after a success 1 day 8 hours ago', () => {
      const orders = view('unhealthy', false).backups.rows.find(
        (row) => row.cluster === 'orders-us'
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
        'Monitor an existing MongoDB service',
        'Install a new MongoDB cluster',
      ]);
      expect(labels(false)).toEqual(['Monitor an existing MongoDB service']);
      expect(labels(true, 'viewer')).toEqual([]);
    });

    it('points the Operations card at Nodes', () => {
      expect(view('empty', true).apps[0].summary?.text).toBe(
        'No MongoDB yet. Install one from Nodes.'
      );
    });
  });
});
