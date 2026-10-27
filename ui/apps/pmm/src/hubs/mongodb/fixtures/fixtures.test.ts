import { attentionIssues } from 'hubs/template/hub.attention';
import { statusFacets } from 'hubs/template/hub.status';
import type { HubViewModel } from 'hubs/template/hub.types';
import { deriveMongoHub } from '../mongodb.derive';
import { describeMongoDestination } from '../mongodb.destinations';
import { deriveContext, NOW, sourcesWith } from '../mongodb.testUtils';
import { clusterHealthOf, groupClusters } from '../mongodb.topology';
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

/** Each cluster's health by name, as the banner counts it: an upgrade Operations runs decides first. */
const healthByName = (
  scenario: (typeof SCENARIOS)[number],
  operations: boolean
) => {
  const snapshot = MONGODB_FIXTURES[scenario](NOW, operations, null);
  const upgrades = new Map(
    (snapshot.operations?.upgrades ?? []).map((upgrade) => [
      upgrade.clusterKey,
      upgrade,
    ])
  );

  return new Map(
    groupClusters(snapshot.services ?? [], snapshot.metrics, NOW).map(
      (cluster) => {
        const upgrade = upgrades.get(cluster.key);

        return [
          cluster.name,
          clusterHealthOf(upgrade ? { ...cluster, upgrade } : cluster),
        ];
      }
    )
  );
};

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
  });

  it('runs every scenario on the 60-cluster fleet', () => {
    (['healthy', 'incomplete', 'unhealthy'] as const).forEach((scenario) => {
      expect(view(scenario, true).status.clusters).toBe(60);
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
      const health = healthByName(scenario, true);
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

  it.each<[(typeof SCENARIOS)[number], string[]]>([
    [
      'healthy',
      [
        'info Point-in-time recovery is off on 9 production clusters',
        'info Advisors flagged 7 clusters for security settings',
        'info 14 clusters are behind on patches',
        'info 6 clusters run MongoDB 6.0',
        'info Query analytics can’t see queries on 12 clusters',
        'info Replication lag and oplog window have no alert rules',
      ],
    ],
    [
      'incomplete',
      [
        'notset No alert rules cover MongoDB',
        'notset 19 clusters have no backups',
        'info Point-in-time recovery is off on 9 production clusters',
        'info 14 clusters are behind on patches',
        'info 6 clusters run MongoDB 6.0',
        'info Query analytics can’t see queries on 12 clusters',
      ],
    ],
    [
      'unhealthy',
      [
        'notset 4 clusters have no backups',
        'info Point-in-time recovery is off on 9 production clusters',
        'info Advisors flagged 7 clusters for security settings',
        'info 14 clusters are behind on patches',
        'info 6 clusters run MongoDB 6.0',
        'info Query analytics can’t see queries on 12 clusters',
      ],
    ],
  ])('recommends the %s scenario’s six', (scenario, expected) => {
    expect(
      view(scenario, true).recommendations.map(
        (item) => `${item.kind} ${item.title}`
      )
    ).toEqual(expected);
  });

  it('asks to turn on Operations, first among what is not set up, while it is off; version advice from PMM’s own data stays', () => {
    const titles = (ops: boolean) =>
      view('unhealthy', ops).recommendations.map((item) => item.title);

    expect(titles(false)).toEqual([
      '4 clusters have no backups',
      'Patch levels aren’t tracked',
      'Point-in-time recovery is off on 9 production clusters',
      'Advisors flagged 7 clusters for security settings',
      '6 clusters run MongoDB 6.0',
      'Query analytics can’t see queries on 12 clusters',
    ]);
    expect(view('unhealthy', false).recommendations[1].destination).toEqual({
      label: 'Turn on Operations',
      route: '/settings/advanced-settings',
      kind: 'page',
    });
  });

  it('matches the backup gaps it recommends filling to the data', () => {
    expect(view('incomplete', true).status.backupCoverage).toEqual({
      protected: 41,
      protectable: 60,
    });
    expect(view('unhealthy', true).status.backupCoverage).toEqual({
      protected: 56,
      protectable: 60,
    });
    expect(view('healthy', true).status.backupCoverage).toEqual({
      protected: 60,
      protectable: 60,
    });
  });

  it('gives every recommendation a destination the sidebar has an icon for', () => {
    (['healthy', 'incomplete', 'unhealthy'] as const).forEach((scenario) =>
      OPS.forEach((ops) =>
        view(scenario, ops).recommendations.forEach(({ destination }) =>
          expect(describeMongoDestination(destination).icon).toBeTruthy()
        )
      )
    );
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
        expect(result.status.health).toEqual({
          healthy: 60,
          degraded: 0,
          down: 0,
          upgrading: 0,
          noData: 0,
        });
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

      expect(staging.status.clusters).toBe(9);
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

        expect(result.status.clusters).toBe(0);
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

    it('keeps every shortcut tile, Fleet included, with nothing monitored', () => {
      expect(view('empty', true).shortcuts.map((tile) => tile.label)).toEqual([
        'Clusters',
        'Instances overview',
        'Query analytics',
        'Real-time queries',
        'MongoDB backups',
        'Fleet',
      ]);
    });
  });
});
