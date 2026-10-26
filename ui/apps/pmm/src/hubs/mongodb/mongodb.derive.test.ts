import { HUB_NO_ENVIRONMENT } from 'hubs/template/hub.types';
import { emptySnapshot } from './fixtures/empty';
import { healthySnapshot } from './fixtures/healthy';
import { incompleteSnapshot } from './fixtures/incomplete';
import { unhealthySnapshot } from './fixtures/unhealthy';
import { deriveMongoHub } from './mongodb.derive';
import type { MongoSnapshot } from './mongodb.types';
import {
  type ContextOptions,
  deriveContext,
  NOW,
  sourcesWith,
} from './mongodb.testUtils';

const derive = (snapshot: MongoSnapshot, options: ContextOptions = {}) =>
  deriveMongoHub(snapshot, deriveContext(options));

const hours = (count: number) => new Date(NOW - count * 3600_000).toISOString();

describe('deriveMongoHub status', () => {
  const ready = sourcesWith('ready');

  it('counts a healthy estate, every cluster PBM can back up protected', () => {
    const { status, attention } = derive(healthySnapshot(NOW, false), {
      scenario: 'healthy',
      sources: ready,
    });

    expect(status).toEqual({
      state: 'healthy',
      clusters: 5,
      health: { healthy: 5, degraded: 0, down: 0, upgrading: 0, noData: 0 },
      patches: { behind: 2, unknown: 0 },
      alerts: 0,
      alertsCritical: 0,
      alertRules: 'some',
      lastGoodBackup: { at: hours(6), cluster: 'billing' },
      backupsFailed: 0,
      // reports is a standalone, which PBM cannot back up.
      backupCoverage: { protected: 4, protectable: 4 },
      checkedAt: new Date(NOW).toISOString(),
      doors: {
        alerts: { label: 'Open alerts', to: '/alerting/status' },
        backups: { label: 'Open backups', to: '/graph/backup/inventory' },
        setUpAlerts: {
          label: 'Set up alerts',
          to: '/graph/alerting/alert-rule-templates',
        },
        setUpBackups: { label: 'Set up backups', to: '/graph/backup/new' },
      },
    });
    expect(attention).toEqual([]);
  });

  it('counts an incomplete setup: two clusters degraded, no alert rule, catalog unprotected', () => {
    expect(
      derive(incompleteSnapshot(NOW, false), {
        scenario: 'incomplete',
        sources: ready,
      }).status
    ).toMatchObject({
      state: 'degraded',
      health: { healthy: 3, degraded: 2, down: 0, upgrading: 0, noData: 0 },
      alerts: 0,
      alertRules: 'none',
      backupsFailed: 0,
      backupCoverage: { protected: 3, protectable: 4 },
    });
  });

  it('counts the unhealthy scenario by health, with its failed backups and firing alerts', () => {
    const { status, attention } = derive(unhealthySnapshot(NOW, false), {
      sources: ready,
    });

    expect(status).toMatchObject({
      state: 'down',
      clusters: 5,
      // Without Operations, payments' upgrade reads as drift, like catalog's.
      health: { healthy: 0, degraded: 3, down: 1, upgrading: 0, noData: 1 },
      alerts: 3,
      alertsCritical: 1,
      // orders' snapshot failed; billing's stopped point-in-time recovery is not a failed backup.
      backupsFailed: 1,
      backupsFailedOldestGood: hours(32),
      lastGoodBackup: { at: hours(6), cluster: 'billing' },
    });
    // Every finding but the slower query, which stays in Queries.
    expect(attention).toHaveLength(13);
  });

  it('counts a cluster Operations is upgrading as upgrading, not as a problem', () => {
    expect(
      derive(unhealthySnapshot(NOW, true), { operations: true }).status.health
    ).toEqual({ healthy: 0, degraded: 2, down: 1, upgrading: 1, noData: 1 });
  });

  it('says nothing is monitored for an empty estate', () => {
    expect(derive(emptySnapshot(NOW, false)).status).toMatchObject({
      state: 'empty',
      clusters: 0,
    });
  });

  it('cannot count health when metrics failed', () => {
    const view = derive(
      { ...unhealthySnapshot(NOW, false), metrics: null },
      {
        sources: sourcesWith('ready', {
          metrics: { status: 'error', message: 'Request failed with 403' },
        }),
      }
    );

    expect(view.status.state).toBe('noData');
    expect(view.status.health).toBeUndefined();
    expect(view.clusters).toEqual([]);
    expect(view.sections.clusters).toEqual({
      status: 'error',
      source: 'metrics',
      message: 'Request failed with 403',
    });
    // Backups, alerts and queries are not health rows; they still show, and the
    // critical down alert has no member row left to merge into.
    expect(view.attention.map((item) => [item.severity, item.source])).toEqual([
      ['error', 'Alert rules'],
      ['warning', 'Backups'],
      ['warning', 'Backups'],
      ['warning', 'Alert rules'],
      ['warning', 'Alert rules'],
      ['warning', 'Real-time queries'],
    ]);
  });

  it('cannot count clusters, health or backups when the inventory failed', () => {
    const view = derive(
      { ...unhealthySnapshot(NOW, false), services: null },
      {
        sources: sourcesWith('ready', {
          inventory: { status: 'error', message: 'Request failed with 502' },
        }),
      }
    );

    expect(view.status).toMatchObject({
      state: 'noData',
      clusters: null,
      lastGoodBackup: 'unknown',
    });
    expect(view.status.health).toBeUndefined();
    expect(view.status.backupCoverage).toBeUndefined();
    expect(view.sections.clusters).toMatchObject({
      status: 'error',
      source: 'inventory',
    });
  });

  it('opens alerts and backups for everyone, but offers only admins to set them up', () => {
    expect(
      Object.keys(
        derive(healthySnapshot(NOW, false), { role: 'viewer' }).status.doors
      )
    ).toEqual(['alerts', 'backups']);
    expect(
      Object.keys(
        derive(healthySnapshot(NOW, false), { backupManagementEnabled: false })
          .status.doors
      )
    ).toEqual(['alerts', 'backups', 'setUpAlerts']);
  });
});

describe('deriveMongoHub environment filter', () => {
  it('offers every environment of the estate whatever the filter', () => {
    expect(
      derive(unhealthySnapshot(NOW, false), { environment: 'prod' }).environment
    ).toEqual({
      selected: 'prod',
      options: [
        { value: 'prod', label: 'prod' },
        { value: 'staging', label: 'staging' },
      ],
      matchesNothing: false,
    });
  });

  it('narrows clusters, attention and backups to the environment', () => {
    const view = derive(unhealthySnapshot(NOW, false), {
      environment: 'staging',
    });

    expect(view.clusters.map((node) => node.name)).toEqual([
      'reports',
      'catalog',
    ]);
    expect(view.status).toMatchObject({
      clusters: 2,
      health: { healthy: 0, degraded: 1, down: 0, upgrading: 0, noData: 1 },
      // catalog is the only staging cluster PBM can back up.
      backupCoverage: { protected: 1, protectable: 1 },
    });
    expect(view.backups.rows.map((row) => row.cluster)).toEqual([
      'catalog',
      'reports',
    ]);
    expect(view.attention.map((item) => item.title)).toEqual([
      'PMM gets no data from reports',
      'catalog runs mixed versions, an upgrade may have stopped partway',
    ]);
  });

  it('shows services without an environment for No environment', () => {
    const snapshot = healthySnapshot(NOW, false);

    snapshot.services = (snapshot.services ?? []).map((service) =>
      service.serviceName === 'reports'
        ? { ...service, environment: undefined }
        : service
    );
    const view = derive(snapshot, { environment: HUB_NO_ENVIRONMENT });

    expect(view.clusters.map((node) => node.name)).toEqual(['reports']);
    expect(
      view.environment.options[view.environment.options.length - 1]
    ).toEqual({
      value: HUB_NO_ENVIRONMENT,
      label: 'No environment',
    });
  });

  it('keeps an entity door on its own environment and a list door on the filter', () => {
    const all = derive(healthySnapshot(NOW, false));
    const prod = derive(healthySnapshot(NOW, false), { environment: 'prod' });

    expect(all.clusters[0].doors.name.to).toContain('var-environment=prod');
    expect(all.sectionDoors.clusters.to).not.toContain('var-environment');
    expect(prod.sectionDoors.clusters.to).toContain('var-environment=prod');
  });

  it('reads a filter that matches nothing as no data, not as empty', () => {
    const view = derive(healthySnapshot(NOW, false), { environment: 'qa' });

    expect(view.status).toMatchObject({ state: 'noData', clusters: 0 });
    expect(view.environment).toEqual({
      selected: 'qa',
      options: [
        { value: 'prod', label: 'prod' },
        { value: 'qa', label: 'qa' },
        { value: 'staging', label: 'staging' },
      ],
      matchesNothing: true,
    });
  });

  it('matches something with services in the environment, and nothing in an empty estate', () => {
    expect(
      derive(healthySnapshot(NOW, false), { environment: 'prod' }).environment
        .matchesNothing
    ).toBe(false);
    expect(
      derive(emptySnapshot(NOW, false), { environment: 'prod' }).environment
        .matchesNothing
    ).toBe(false);
  });
});

describe('deriveMongoHub capabilities', () => {
  it('gives an admin with Operations every action and Operations doors', () => {
    const view = derive(healthySnapshot(NOW, true), { operations: true });

    expect(
      view.actions.add.map(({ label, to, description }) => [
        label,
        to,
        description,
      ])
    ).toEqual([
      [
        'Monitor an existing MongoDB service',
        '/graph/add-instance/mongodb',
        undefined,
      ],
      [
        'Install a new MongoDB cluster',
        '/operations/nodes?filter=unmonitored',
        undefined,
      ],
    ]);
    expect(
      view.actions.more.map(({ label, to, external }) => [
        label,
        to,
        !!external,
      ])
    ).toEqual([
      ['Manage services', '/graph/inventory/services', false],
      ['Configure backups', '/graph/backup/scheduled', false],
      ['Set up alert rules', '/graph/alerting/alert-rule-templates', false],
      ['Run Advisors checks', '/graph/advisors/insights', false],
      [
        'Documentation',
        'https://docs.percona.com/percona-monitoring-and-management/3/install-pmm/install-pmm-client/connect-database/mongodb.html',
        true,
      ],
    ]);
    expect(view.sectionDoors.clusters).toEqual({
      label: 'Fleet in Operations',
      to: '/operations?tab=clusters',
    });
    expect(view.apps.map((app) => app.id)).toEqual([
      'operations',
      'backups',
      'diagnostics',
    ]);
  });

  it('opens Instances overview without Operations and drops its card', () => {
    const view = derive(healthySnapshot(NOW, false));

    expect(view.actions.add.map((action) => action.label)).toEqual([
      'Monitor an existing MongoDB service',
    ]);
    expect(view.sectionDoors.clusters).toEqual({
      label: 'Instances overview',
      to: '/graph/d/mongodb-instance-overview/mongodb-instances-overview',
    });
    expect(view.apps.map((app) => app.id)).toEqual(['backups', 'diagnostics']);
  });

  it('gives a viewer no actions and no Operations, whatever the switch', () => {
    const view = derive(healthySnapshot(NOW, true), {
      role: 'viewer',
      operations: true,
    });

    // The docs are for everyone; the rest is for admins.
    expect(view.actions.add).toEqual([]);
    expect(view.actions.more.map((action) => action.label)).toEqual([
      'Documentation',
    ]);
    expect(view.capabilities.operations).toBe(false);
    expect(view.apps.map((app) => app.id)).toEqual(['backups', 'diagnostics']);
  });

  it('shows Diagnostics only with PMM Extensions', () => {
    expect(
      derive(healthySnapshot(NOW, false), { extensions: false }).apps.map(
        (app) => app.id
      )
    ).toEqual(['backups']);
  });
});

describe('deriveMongoHub sections', () => {
  it('reads every section ready for sample data', () => {
    expect(Object.values(derive(healthySnapshot(NOW, false)).sections)).toEqual(
      Array(6).fill({ status: 'ready' })
    );
  });

  it('shows loading while a section’s source loads', () => {
    const { sections } = derive(healthySnapshot(NOW, false), {
      sources: sourcesWith('ready', { qan: { status: 'loading' } }),
    });

    expect(sections.queries).toEqual({ status: 'loading' });
    expect(sections.clusters).toEqual({ status: 'ready' });
  });

  it('fails only the section of the failed source', () => {
    const { sections } = derive(healthySnapshot(NOW, false), {
      sources: sourcesWith('ready', {
        backups: { status: 'error', message: 'Access denied' },
      }),
    });

    expect(sections.backups).toEqual({
      status: 'error',
      source: 'backups',
      message: 'Access denied',
    });
    expect(sections.attention).toEqual({ status: 'ready' });
  });

  it('keeps every derived attention row when only alerts failed', () => {
    const view = derive(
      { ...unhealthySnapshot(NOW, false), alerts: null },
      {
        sources: sourcesWith('ready', {
          alerts: {
            status: 'error',
            message: 'Request failed with status code 502',
          },
        }),
      }
    );

    expect(view.sections.attention).toEqual({
      status: 'error',
      source: 'alerts',
      message:
        'Firing alerts could not be read (Request failed with status code 502), so none is listed here.',
      partial: true,
    });
    expect(view.attention[0]).toMatchObject({
      severity: 'critical',
      title: "billing has no primary, it can't take writes",
    });
    expect(view.attention.some((item) => item.alertFiring)).toBe(false);
  });

  it('fails the attention section outright when metrics failed', () => {
    expect(
      derive(
        { ...unhealthySnapshot(NOW, false), metrics: null, alerts: null },
        {
          sources: sourcesWith('ready', {
            metrics: { status: 'error' },
            alerts: { status: 'error' },
          }),
        }
      ).sections.attention
    ).toEqual({ status: 'error', source: 'metrics' });
  });

  it('states nothing about backups it could not read', () => {
    const snapshot = healthySnapshot(NOW, false);

    // Without PBM configured, only a backup the hub can see recommends Backup status.
    if (snapshot.metrics) {
      snapshot.metrics.pbmConfiguredByCluster = {};
    }
    const view = derive(
      { ...snapshot, backups: null },
      {
        scenario: 'healthy',
        sources: sourcesWith('ready', {
          backups: { status: 'error', message: 'Request failed with 403' },
        }),
      }
    );

    expect(view.status.lastGoodBackup).toBe('unknown');
    expect(view.backups.state).toBe('unknown');
    expect(view.apps.find((app) => app.id === 'backups')?.summary).toEqual({
      text: 'Backup status unknown',
    });
    expect(view.dashboards.recommended.map((link) => link.label)).not.toContain(
      'Backup status'
    );
  });

  it('never reads an empty estate while the inventory is still loading', () => {
    expect(
      derive(emptySnapshot(NOW, false), {
        sources: sourcesWith('ready', { inventory: { status: 'loading' } }),
      }).status.state
    ).not.toBe('empty');
  });
});

describe('deriveMongoHub tree', () => {
  it('starts every cluster as one row, the unhealthy ones too', () => {
    expect(derive(unhealthySnapshot(NOW, false)).expanded).toEqual({});
  });

  it('drops alerts of another environment', () => {
    const snapshot = healthySnapshot(NOW, false);

    snapshot.alerts = [
      {
        name: 'MongoDB restarted',
        templateName: 'pmm_mongodb_restarted',
        severity: 'warning',
        serviceName: 'reports',
        activeAt: hours(1),
      },
    ];

    expect(derive(snapshot, { environment: 'prod' }).attention).toEqual([]);
    expect(
      derive(snapshot, { environment: 'staging' }).attention.map(
        (item) => item.source
      )
    ).toEqual(['Alert rules']);
  });
});

describe('deriveMongoHub patches', () => {
  it('counts the clusters behind on patches and those the check could not answer for', () => {
    const { status } = derive(unhealthySnapshot(NOW, false));

    // payments and catalog run 7.0.24-13; billing's check is three days old.
    expect(status.patches).toEqual({ behind: 2, unknown: 1 });
  });

  it('says nothing about patches while nothing checks for updates', () => {
    const view = derive(
      { ...unhealthySnapshot(NOW, false), updates: null },
      { scenario: 'live' }
    );

    expect(view.status.patches).toBeUndefined();
    expect(
      view.clusters.every((node) => node.version?.update === undefined)
    ).toBe(true);
  });

  it('shows only the clusters behind on patches while the filter is on, and still counts them all', () => {
    const view = derive(healthySnapshot(NOW, false), {
      scenario: 'healthy',
      patches: 'behind',
    });

    expect(view.patchesFilter).toBe(true);
    expect(view.clusters.map((node) => node.name)).toEqual([
      'catalog',
      'payments',
    ]);
    expect(view.status).toMatchObject({
      clusters: 5,
      patches: { behind: 2, unknown: 0 },
    });
  });
});

describe('deriveMongoHub alerts', () => {
  it('counts the firing alerts, host alerts included', () => {
    const view = derive(unhealthySnapshot(NOW, false), {
      sources: sourcesWith('ready'),
    });

    expect(view.status.alerts).toBe(3);
    expect(
      view.attention.find((item) => item.title.startsWith('Node high CPU'))
    ).toMatchObject({
      alertFiring: true,
      location: 'orders › orders-shard2',
      environment: 'prod',
      destination: { label: 'Open Alerts', route: '/alerting/status' },
    });
  });

  it('cannot count alerts it could not read, and says nothing while alerting is off', () => {
    expect(
      derive(
        { ...unhealthySnapshot(NOW, false), alerts: null },
        { sources: sourcesWith('ready', { alerts: { status: 'error' } }) }
      ).status.alerts
    ).toBeNull();
    expect(
      derive(unhealthySnapshot(NOW, false), {
        sources: sourcesWith('ready', { alerts: { status: 'disabled' } }),
      }).status.alerts
    ).toBeUndefined();
  });

  it('says what no alert rule covers, with templates for admins and an administrator for viewers', () => {
    const admin = derive(unhealthySnapshot(NOW, false));

    expect(admin.alertCoverage).toEqual({
      text: 'No rule from the replication lag or oplog window template yet.',
      door: {
        label: 'Create from templates',
        to: '/graph/alerting/alert-rule-templates',
      },
    });

    const none = {
      ...unhealthySnapshot(NOW, false),
      alertRules: { templates: ['pmm_grafana_down'], custom: 0 },
    };

    expect(derive(none).alertCoverage).toMatchObject({
      text: 'No alert rules cover MongoDB yet.',
      detail: '(member down, replication lag, oplog window)',
    });
    expect(derive(none, { role: 'viewer' }).alertCoverage).toEqual({
      text: 'No alert rules cover MongoDB yet.',
      detail:
        'An administrator can create them from templates (member down, replication lag, oplog window).',
    });
  });

  it('says nothing about coverage once the key templates have rules, or without rules to read', () => {
    expect(
      derive(healthySnapshot(NOW, false), { scenario: 'healthy' }).alertCoverage
    ).toBeUndefined();
    expect(
      derive({ ...unhealthySnapshot(NOW, false), alertRules: null })
        .alertCoverage
    ).toBeUndefined();
    expect(derive(emptySnapshot(NOW, false)).alertCoverage).toBeUndefined();
  });
});
