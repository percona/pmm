import { healthySnapshot } from './fixtures/healthy';
import { wrongSnapshot } from './fixtures/wrong';
import { type BackupsInput, deriveBackups } from './mongodb.backups';
import { groupClusters } from './mongodb.topology';
import type { MongoBackupsInput, MongoSnapshot } from './mongodb.types';
import { NOW } from './mongodb.testUtils';

const HOUR = 3600_000;
const ago = (hours: number) => new Date(NOW - hours * HOUR).toISOString();

const derive = (
  snapshot: MongoSnapshot,
  overrides: Partial<BackupsInput> = {}
) =>
  deriveBackups({
    clusters: groupClusters(snapshot.services ?? [], snapshot.metrics),
    input: snapshot.backups,
    metrics: snapshot.metrics,
    capabilities: { setUpBackup: true },
    role: 'admin',
    filter: null,
    now: NOW,
    ...overrides,
  });

const withBackups = (backups: MongoBackupsInput | null) => ({
  ...healthySnapshot(NOW, false),
  backups,
});

describe('deriveBackups', () => {
  it('says backup management is off, with a way to turn it on for admins', () => {
    const { backups, lastGoodBackup } = derive(
      withBackups({ enabled: false, artifacts: [], scheduled: [] })
    );

    expect(backups.state).toBe('featureOff');
    expect(backups.doors.settings).toEqual({
      label: 'Turn it on in Settings',
      to: '/settings/advanced-settings',
    });
    expect(backups.doors.setUp).toBeUndefined();
    expect(lastGoodBackup).toBe('featureOff');
    expect(
      derive(withBackups({ enabled: false, artifacts: [], scheduled: [] }), {
        role: 'viewer',
        capabilities: { setUpBackup: false },
      }).backups.doors.settings
    ).toBeUndefined();
  });

  it('says there is nothing to back up without services', () => {
    const { backups } = derive({
      ...withBackups(healthySnapshot(NOW, false).backups),
      services: [],
    });

    expect(backups).toMatchObject({ state: 'noServices', rows: [] });
  });

  it('says how to start when nothing is set up, with the PBM hint', () => {
    const snapshot = withBackups({
      enabled: true,
      artifacts: [],
      scheduled: [],
    });

    if (snapshot.metrics) {
      snapshot.metrics.pbmConfiguredByCluster.catalog = false;
    }
    const { backups, lastGoodBackup, coverage } = derive(snapshot);

    expect(backups.state).toBe('notSetUp');
    expect(backups.doors.setUp).toEqual({
      label: 'Set up a backup',
      to: '/graph/backup/new',
    });
    expect(lastGoodBackup).toBe('notSetUp');
    // reports is a standalone, which PBM cannot back up.
    expect(backups.hint).toBe('PBM is not configured on catalog');
    expect(coverage).toEqual({ protected: 0, protectable: 4 });
  });

  it('lists clusters where PBM is not configured, never standalones', () => {
    const snapshot = withBackups({
      enabled: true,
      artifacts: [],
      scheduled: [],
    });

    if (snapshot.metrics) {
      snapshot.metrics.pbmConfiguredByCluster = {
        orders: false,
        billing: false,
        reports: false,
      };
    }

    expect(derive(snapshot).backups.hint).toBe(
      'PBM is not configured on billing, orders'
    );
  });

  it('reads healthy backups per cluster: last good, PITR, next run', () => {
    const { backups, lastGoodBackup, findings, schedules, coverage } = derive(
      healthySnapshot(NOW, false)
    );

    expect(backups.state).toBe('ok');
    expect(
      backups.rows.map((row) => [
        row.cluster,
        row.lastGood?.at,
        row.lastGood?.level,
        row.pitr,
        row.nextRun,
      ])
    ).toEqual([
      ['billing', ago(6), 'ok', 'on', expect.stringMatching(/T03:00:00/)],
      ['catalog', ago(9), 'ok', 'off', expect.stringMatching(/T05:00:00/)],
      ['orders', ago(7), 'ok', 'off', expect.stringMatching(/T02:00:00/)],
      ['payments', ago(8), 'ok', 'off', expect.stringMatching(/T04:00:00/)],
      ['reports', undefined, undefined, 'notApplicable', undefined],
    ]);
    expect(lastGoodBackup).toEqual({ at: ago(6), cluster: 'billing' });
    expect(findings).toEqual([]);
    expect(schedules).toBe(5);
    expect(coverage).toEqual({ protected: 4, protectable: 4 });
  });

  it('finds a failure after the last success, and marks the stale success', () => {
    const { backups, findings } = derive(wrongSnapshot(NOW, false));
    const orders = backups.rows.find((row) => row.cluster === 'orders');

    expect(orders).toMatchObject({
      failedAt: ago(8),
      lastGood: { at: ago(32), level: 'warning' },
    });
    expect(findings).toContainEqual({
      kind: 'backupFailed',
      clusterKey: 'prod/orders',
      cluster: 'orders',
      environment: 'prod',
      name: 'nightly backup',
      since: ago(8),
    });
  });

  it('reports PITR stopped when its newest artifact failed', () => {
    const { backups, findings, lastGoodBackup } = derive(
      wrongSnapshot(NOW, false)
    );

    expect(backups.rows.find((row) => row.cluster === 'billing')?.pitr).toBe(
      'stopped'
    );
    expect(findings.map((finding) => finding.kind)).toEqual([
      'backupFailed',
      'pitrStopped',
    ]);
    expect(lastGoodBackup).toEqual({ at: ago(6), cluster: 'billing' });
  });

  it('puts problems first, then clusters without a backup, then by name', () => {
    expect(
      derive(wrongSnapshot(NOW, false)).backups.rows.map((row) => row.cluster)
    ).toEqual(['billing', 'orders', 'catalog', 'payments', 'reports']);
  });

  it('ignores a failure older than 24 hours for Needs attention but keeps it in the row', () => {
    const snapshot = withBackups({
      enabled: true,
      scheduled: [],
      artifacts: [
        {
          artifactId: 'a',
          name: 'manual',
          serviceId: 'svc-billing-m1',
          serviceName: 'billing-m1',
          mode: 'snapshot',
          status: 'error',
          createdAt: ago(30),
        },
      ],
    });
    const { backups, findings } = derive(snapshot);

    expect(findings).toEqual([]);
    expect(backups.rows.find((row) => row.cluster === 'billing')).toMatchObject(
      {
        failedAt: ago(30),
        lastGood: null,
      }
    );
  });

  it('says PITR has not started, never On, before its schedule makes an artifact', () => {
    const snapshot = healthySnapshot(NOW, false);

    if (snapshot.backups) {
      snapshot.backups.artifacts = snapshot.backups.artifacts.filter(
        (artifact) => artifact.mode !== 'pitr'
      );
    }
    const { backups, findings } = derive(snapshot);

    expect(backups.rows.find((row) => row.cluster === 'billing')?.pitr).toBe(
      'notStarted'
    );
    expect(findings).toEqual([]);
  });

  it('turns PITR off when its schedule is disabled', () => {
    const snapshot = healthySnapshot(NOW, false);

    snapshot.backups?.scheduled.forEach((schedule) => {
      schedule.enabled = schedule.mode !== 'pitr';
    });

    expect(
      derive(snapshot).backups.rows.find((row) => row.cluster === 'billing')
        ?.pitr
    ).toBe('off');
  });

  it('maps artifacts to clusters through their service and drops other services', () => {
    const snapshot = withBackups({
      enabled: true,
      scheduled: [],
      artifacts: [
        {
          artifactId: 'x',
          name: 'elsewhere',
          serviceId: 'svc-not-shown',
          serviceName: 'gone',
          mode: 'snapshot',
          status: 'success',
          createdAt: ago(1),
        },
      ],
    });

    expect(derive(snapshot).backups.state).toBe('notSetUp');
  });

  it('opens each cell on its page', () => {
    const [billing] = derive(healthySnapshot(NOW, false)).backups.rows;

    expect(billing.doors).toEqual({
      cluster: {
        label: 'billing',
        to: '/graph/d/mongodb-backup-details/mongodb-backup-details?var-environment=prod&var-cluster=billing',
      },
      lastGood: {
        label: 'Last good backup',
        to: '/graph/backup/inventory',
        context: 'for billing',
      },
      schedule: {
        label: 'Open scheduled backups',
        to: '/graph/backup/scheduled',
        context: 'for billing',
      },
    });
  });

  it('carries the environment filter into the backup dashboard door', () => {
    expect(
      derive(healthySnapshot(NOW, false), { filter: 'prod' }).backups.doors
        .dashboard
    ).toEqual({
      label: 'Backup status dashboard',
      to: '/graph/d/mongodb-backup-details/mongodb-backup-details?var-environment=prod',
    });
  });

  it('knows nothing when the source failed', () => {
    expect(derive(withBackups(null))).toMatchObject({
      backups: { state: 'unknown', rows: [] },
      lastGoodBackup: 'unknown',
      schedules: 0,
      failedRecently: 0,
    });
  });
});
