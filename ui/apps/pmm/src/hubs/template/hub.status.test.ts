import { describe } from 'vitest';
import { type HubStatusContext, statusFacets } from './hub.status';
import type { HubStatus } from './hub.types';

const hoursAgo = (hours: number) =>
  new Date(Date.now() - hours * 3600_000).toISOString();

const SET_UP_ALERTS = {
  label: 'Set up alerts',
  to: '/graph/alerting/alert-rule-templates',
};
const SET_UP_BACKUPS = { label: 'Set up backups', to: '/graph/backup/new' };
const OPEN_ALERTS = { label: 'Open alerts', to: '/alerting/status' };
const OPEN_BACKUPS = { label: 'Open backups', to: '/graph/backup/inventory' };

const HEALTHY: HubStatus = {
  state: 'healthy',
  clusters: 5,
  health: { healthy: 5, degraded: 0, down: 0, upgrading: 0, noData: 0 },
  alerts: 0,
  alertRules: 'some',
  lastGoodBackup: { at: hoursAgo(6), cluster: 'billing' },
  backupsFailed: 0,
  backupCoverage: { protected: 4, protectable: 4 },
  checkedAt: hoursAgo(0),
  doors: {
    alerts: OPEN_ALERTS,
    backups: OPEN_BACKUPS,
    setUpAlerts: SET_UP_ALERTS,
    setUpBackups: SET_UP_BACKUPS,
  },
};

const CONTEXT: HubStatusContext = {
  name: 'MongoDB',
  unit: ['cluster', 'clusters'],
  loading: false,
  matchesNothing: false,
  issues: 0,
};

const facets = (
  status: Partial<HubStatus> = {},
  context: Partial<HubStatusContext> = {}
) => statusFacets({ ...HEALTHY, ...status }, { ...CONTEXT, ...context });

describe('statusFacets', () => {
  it('reads Healthy as the Figma banner does', () => {
    expect(facets()).toEqual([
      {
        id: 'health',
        state: 'healthy',
        label: 'All 5 clusters are healthy',
      },
      { id: 'alerts', state: 'default', label: 'No alerts firing' },
      {
        id: 'backups',
        state: 'default',
        label: 'Backups on schedule, last 6h ago',
      },
    ]);
  });

  it('reads Problematic with the filled state for clusters, errors for alerts and backups, and a way into each', () => {
    expect(
      facets(
        {
          state: 'down',
          health: {
            healthy: 1,
            degraded: 2,
            down: 1,
            upgrading: 0,
            noData: 1,
          },
          alerts: 3,
          alertsCritical: 1,
          backupsFailed: 2,
          backupsFailedOldestGood: hoursAgo(32),
        },
        { issues: 15 }
      )
    ).toEqual([
      {
        id: 'health',
        state: 'mixedProblems',
        label: '4 of 5 clusters need attention',
        action: { label: 'Review 15 issues', to: '#needs-attention' },
      },
      {
        id: 'alerts',
        state: 'errors',
        label: '3 alerts firing, 1 critical',
        action: OPEN_ALERTS,
      },
      {
        id: 'backups',
        state: 'errors',
        label: '2 backups failed, oldest good backup 1d 8h ago',
        action: OPEN_BACKUPS,
      },
    ]);
  });

  it('reviews a cluster problem down the page, only while Needs attention lists one', () => {
    const degraded = {
      health: { healthy: 4, degraded: 1, down: 0, upgrading: 0, noData: 0 },
    };
    const upgrading = {
      health: { healthy: 4, degraded: 0, down: 0, upgrading: 1, noData: 0 },
    };

    expect(facets(degraded, { issues: 1 })[0].action).toEqual({
      label: 'Review 1 issue',
      to: '#needs-attention',
    });
    expect(facets(degraded, { issues: 0 })[0].action).toBeUndefined();
    expect(facets({}, { issues: 3 })[0].action).toBeUndefined();
    expect(facets(upgrading, { issues: 3 })[0].action).toBeUndefined();
  });

  it('reads Incomplete with a way to set up alerts and backups', () => {
    expect(
      facets(
        {
          state: 'degraded',
          health: {
            healthy: 3,
            degraded: 2,
            down: 0,
            upgrading: 0,
            noData: 0,
          },
          alertRules: 'none',
          backupCoverage: { protected: 3, protectable: 4 },
        },
        { issues: 2 }
      )
    ).toEqual([
      {
        id: 'health',
        state: 'degraded',
        label: '2 of 5 clusters degraded',
        action: { label: 'Review 2 issues', to: '#needs-attention' },
      },
      {
        id: 'alerts',
        state: 'notSetUp',
        label: 'No alerts set up',
        action: SET_UP_ALERTS,
      },
      {
        id: 'backups',
        state: 'notSetUp',
        label: '3 of 4 clusters protected',
        action: SET_UP_BACKUPS,
      },
    ]);
  });

  it('leaves the actions out for a reader who cannot set anything up', () => {
    const viewer = facets({
      alertRules: 'none',
      lastGoodBackup: 'notSetUp',
      doors: { alerts: OPEN_ALERTS, backups: OPEN_BACKUPS },
    });

    expect(viewer.map((facet) => facet.action)).toEqual([
      undefined,
      undefined,
      undefined,
    ]);
    expect(viewer[2].label).toBe('No backups set up');
  });

  it('checks every facet while loading', () => {
    expect(facets({}, { loading: true })).toEqual([
      { id: 'health', state: 'loading', label: 'Checking clusters...' },
      { id: 'alerts', state: 'loading', label: 'Checking alerts...' },
      { id: 'backups', state: 'loading', label: 'Checking backups...' },
    ]);
  });

  it('says only that nothing is monitored on a first run, or in an empty environment', () => {
    expect(facets({ state: 'empty', clusters: 0 })).toEqual([
      { id: 'health', state: 'notSetUp', label: 'No MongoDB monitored yet' },
    ]);
    expect(facets({}, { matchesNothing: true })).toEqual([
      {
        id: 'health',
        state: 'default',
        label: 'No MongoDB in this environment',
      },
    ]);
  });

  it('counts no data with degraded, as a warning, until a cluster is down', () => {
    expect(
      facets({
        health: { healthy: 3, degraded: 1, down: 0, upgrading: 0, noData: 1 },
      })[0]
    ).toEqual({
      id: 'health',
      state: 'degraded',
      label: '2 of 5 clusters degraded',
    });
    expect(
      facets({
        health: { healthy: 0, degraded: 3, down: 1, upgrading: 0, noData: 1 },
      })[0]
    ).toEqual({
      id: 'health',
      state: 'mixedProblems',
      label: 'All 5 clusters need attention',
    });
  });

  it('says how many firing alerts are critical, if any', () => {
    expect(facets({ alerts: 17, alertsCritical: 4 })[1].label).toBe(
      '17 alerts firing, 4 critical'
    );
    expect(facets({ alerts: 2, alertsCritical: 2 })[1].label).toBe(
      '2 alerts firing, all critical'
    );
    expect(facets({ alerts: 2, alertsCritical: 0 })[1].label).toBe(
      '2 alerts firing'
    );
  });

  it('reads an upgrade in progress as information, not a problem', () => {
    expect(
      facets({
        health: { healthy: 4, degraded: 0, down: 0, upgrading: 1, noData: 0 },
      })[0]
    ).toEqual({
      id: 'health',
      state: 'information',
      label: '1 of 5 clusters upgrading',
    });
  });

  it('speaks of a single cluster in the singular', () => {
    const one = (health: HubStatus['health']) =>
      facets({ clusters: 1, health })[0].label;

    expect(
      one({ healthy: 1, degraded: 0, down: 0, upgrading: 0, noData: 0 })
    ).toBe('1 cluster is healthy');
    expect(
      one({ healthy: 0, degraded: 0, down: 1, upgrading: 0, noData: 0 })
    ).toBe('1 cluster needs attention');
    expect(
      one({ healthy: 0, degraded: 1, down: 0, upgrading: 0, noData: 0 })
    ).toBe('1 cluster is degraded');
    expect(facets({ alerts: 1 })[1].label).toBe('1 alert firing');
    expect(
      facets({ backupsFailed: 1, backupsFailedOldestGood: hoursAgo(6) })[2]
        .label
    ).toBe('1 backup failed, last good backup 6h ago');
  });

  it('says what it cannot tell instead of guessing', () => {
    expect(
      facets({ health: undefined, alerts: null, lastGoodBackup: 'unknown' })
    ).toEqual([
      { id: 'health', state: 'default', label: 'Cluster health unknown' },
      { id: 'alerts', state: 'default', label: 'Alerts unavailable' },
      { id: 'backups', state: 'default', label: 'Backups unavailable' },
    ]);
  });

  it('leaves alerts out while alerting is off, and says so for backups', () => {
    expect(
      facets({ alerts: undefined, lastGoodBackup: 'featureOff' }).map(
        ({ id, state, label }) => [id, state, label]
      )
    ).toEqual([
      ['health', 'healthy', 'All 5 clusters are healthy'],
      ['backups', 'notSetUp', 'Backup management is off'],
    ]);
  });

  it('puts a failed backup before missing protection, and says when none is good yet', () => {
    expect(
      facets({
        backupsFailed: 1,
        lastGoodBackup: 'none',
        backupCoverage: { protected: 1, protectable: 4 },
      })[2]
    ).toEqual({
      id: 'backups',
      state: 'errors',
      label: '1 backup failed, no good backup yet',
      action: OPEN_BACKUPS,
    });
    expect(facets({ lastGoodBackup: 'none' })[2]).toEqual({
      id: 'backups',
      state: 'default',
      label: 'No backup taken yet',
    });
  });
});
