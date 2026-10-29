import { DASHBOARD_CATALOG, recentDashboards } from './fixtures/estate';
import { healthySnapshot } from './fixtures/healthy';
import {
  type DashboardsInput,
  deriveDashboards,
  mainFolder,
  recommendedIds,
} from './mongodb.dashboards';
import { groupClusters } from './mongodb.topology';
import { NOW, replicaSet, service } from './mongodb.testUtils';

const estate = () => {
  const snapshot = healthySnapshot(NOW, false);

  return groupClusters(snapshot.services ?? [], snapshot.metrics);
};

const derive = (overrides: Partial<DashboardsInput> = {}) =>
  deriveDashboards({
    clusters: estate(),
    dashboards: { catalog: DASHBOARD_CATALOG, recent: recentDashboards(NOW) },
    hasBackups: true,
    filter: null,
    ...overrides,
  });

describe('recommendedIds', () => {
  it('recommends nothing without MongoDB', () => {
    expect(recommendedIds({ clusters: [], hasBackups: true })).toEqual([]);
  });

  it('recommends the overview and summary for any MongoDB', () => {
    expect(
      recommendedIds({
        clusters: groupClusters([service('lone')], null),
        hasBackups: false,
      })
    ).toEqual(['instancesOverview', 'instanceSummary']);
  });

  it('adds replica set dashboards for a replica set', () => {
    const { services, metrics } = replicaSet('rs');

    expect(
      recommendedIds({
        clusters: groupClusters(services, metrics),
        hasBackups: false,
      })
    ).toEqual([
      'instancesOverview',
      'instanceSummary',
      'replsetSummary',
      'oplogDetails',
    ]);
  });

  it('adds sharded and backup dashboards when PMM sees them', () => {
    expect(recommendedIds({ clusters: estate(), hasBackups: true })).toEqual([
      'instancesOverview',
      'instanceSummary',
      'replsetSummary',
      'oplogDetails',
      'clusterSummary',
      'routerSummary',
      'backupStatus',
    ]);
  });
});

describe('deriveDashboards', () => {
  it('links recommended and developer dashboards with the environment filter', () => {
    const { recommended, forDevelopers } = derive({ filter: 'prod' });

    expect(recommended[0]).toEqual({
      uid: 'mongodb-instance-overview',
      label: 'Instances overview',
      to: '/graph/d/mongodb-instance-overview/mongodb-instances-overview?var-environment=prod',
    });
    expect(forDevelopers.map((link) => link.label)).toEqual([
      'Collections overview',
      'Unused indexes',
    ]);
  });

  it('lists recent visits from the catalog, newest first, without variables', () => {
    expect(derive().recent).toEqual([
      {
        uid: 'mongodb-replicaset-summary',
        label: 'ReplSet summary',
        to: '/graph/d/mongodb-replicaset-summary',
        visitedAt: new Date(NOW - 5 * 60_000).toISOString(),
      },
      {
        uid: 'mongodb-oplog-details',
        label: 'Oplog details',
        to: '/graph/d/mongodb-oplog-details',
        visitedAt: new Date(NOW - 24 * 3600_000).toISOString(),
      },
      {
        uid: 'mongodb-instance-summary',
        label: 'Instance summary',
        to: '/graph/d/mongodb-instance-summary',
        visitedAt: new Date(NOW - 48 * 3600_000).toISOString(),
      },
    ]);
  });

  it('keeps only catalog dashboards, each once, five at most', () => {
    const recent = [
      'prometheus-status',
      'not-mongodb',
      ...DASHBOARD_CATALOG.slice(0, 7).map((hit) => hit.uid),
      'mongodb-backup-details',
    ].map((uid) => ({ uid }));
    const result = derive({
      dashboards: { catalog: DASHBOARD_CATALOG, recent },
    }).recent;

    expect(result).toHaveLength(5);
    expect(result.map((link) => link.uid)).toEqual(
      DASHBOARD_CATALOG.slice(0, 5).map((hit) => hit.uid)
    );
  });

  it('opens the MongoDB folder and counts its dashboards', () => {
    expect(derive().all).toEqual({
      label: 'All MongoDB dashboards',
      to: '/graph/dashboards/f/mongodb-folder/',
      count: 13,
    });
  });

  it('falls back to a search without a folder', () => {
    expect(
      derive({
        dashboards: {
          catalog: DASHBOARD_CATALOG.map(
            ({ folderUid: _folder, ...hit }) => hit
          ),
          recent: [],
        },
      }).all
    ).toEqual({
      label: 'All MongoDB dashboards',
      to: '/graph/dashboards?query=MongoDB',
      count: 14,
    });
  });

  it('keeps the door but no count when the catalog failed', () => {
    expect(derive({ dashboards: null }).all).toEqual({
      label: 'All MongoDB dashboards',
      to: '/graph/dashboards?query=MongoDB',
    });
  });

  it('shows only the door for an empty estate', () => {
    const empty = derive({ clusters: [] });

    expect(empty.recommended).toEqual([]);
    expect(empty.forDevelopers).toEqual([]);
    expect(empty.all.count).toBe(13);
  });
});

describe('mainFolder', () => {
  it('picks the folder most hits share', () => {
    expect(mainFolder(DASHBOARD_CATALOG)).toBe('mongodb-folder');
    expect(mainFolder([])).toBeUndefined();
  });
});
