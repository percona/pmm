import { HUB_NO_ENVIRONMENT } from 'hubs/template/hub.types';
import { emptySnapshot } from './fixtures/empty';
import { healthySnapshot } from './fixtures/healthy';
import { wrongSnapshot } from './fixtures/wrong';
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
  it('reads a healthy estate as healthy, none need attention', () => {
    const { status, attention } = derive(healthySnapshot(NOW, false), {
      scenario: 'healthy',
    });

    expect(status).toMatchObject({
      state: 'healthy',
      clusters: 3,
      needAttention: 0,
      otherIssues: 0,
      lastGoodBackup: { at: hours(6), cluster: 'billing' },
      checkedAt: new Date(NOW).toISOString(),
    });
    expect(status.attentionSeverity).toBeUndefined();
    expect(attention).toEqual([]);
  });

  it('carries the worst row severity and counts rows outside any cluster', () => {
    const snapshot = healthySnapshot(NOW, true);

    snapshot.alerts = [
      {
        name: 'MongoDB restarted',
        templateName: 'pmm_mongodb_restarted',
        severity: 'critical',
        serviceName: 'billing-m1',
        cluster: 'billing',
        environment: 'prod',
        activeAt: hours(1),
      },
    ];
    snapshot.qan = wrongSnapshot(NOW, true).qan;
    snapshot.operations = wrongSnapshot(NOW, true).operations;

    expect(
      derive(snapshot, { scenario: 'healthy', operations: true }).status
    ).toMatchObject({
      state: 'healthy',
      needAttention: 1,
      otherIssues: 3,
      attentionSeverity: 'error',
    });
  });

  it('counts the clusters that need attention, not the rows', () => {
    const { status, attention } = derive(wrongSnapshot(NOW, false));

    expect(status).toMatchObject({
      state: 'down',
      clusters: 3,
      needAttention: 3,
      otherIssues: 1,
      attentionSeverity: 'error',
      lastGoodBackup: { at: hours(6), cluster: 'billing' },
    });
    expect(attention).toHaveLength(11);
  });

  it('says nothing is monitored for an empty estate', () => {
    expect(derive(emptySnapshot(NOW, false)).status).toMatchObject({
      state: 'empty',
      clusters: 0,
      needAttention: 0,
    });
  });

  it('cannot say what needs attention when metrics failed', () => {
    const view = derive(
      { ...wrongSnapshot(NOW, false), metrics: null },
      {
        sources: sourcesWith('ready', {
          metrics: { status: 'error', message: 'Request failed with 403' },
        }),
      }
    );

    expect(view.status).toMatchObject({ state: 'noData', needAttention: null });
    expect(view.clusters).toEqual([]);
    expect(view.sections.clusters).toEqual({
      status: 'error',
      source: 'metrics',
      message: 'Request failed with 403',
    });
    // Backups, alerts and queries are not health rows; they still show, and the
    // critical down alert has no member row left to merge into.
    expect(view.attention.map((item) => item.kind)).toEqual([
      'alert',
      'backupFailed',
      'pitrStopped',
      'alert',
      'querySlower',
      'queryRunning',
    ]);
  });

  it('links its figures back into the page, keeping the hub params', () => {
    const { status } = derive(wrongSnapshot(NOW, false), {
      environment: 'prod',
    });

    expect(status.doors.clusters.to).toBe(
      '?environment=prod&hubState=wrong&hubOps=off&hubRole=admin#clusters'
    );
    expect(status.doors.attention.to).toBe(
      '?environment=prod&hubState=wrong&hubOps=off&hubRole=admin#needs-attention'
    );
    expect(status.doors.backup.to).toBe('/graph/backup/inventory');
  });

  it('offers admins to set up a backup when none is', () => {
    const snapshot = {
      ...healthySnapshot(NOW, false),
      backups: { enabled: true, artifacts: [], scheduled: [] },
    };

    expect(derive(snapshot).status.doors.backup).toEqual({
      label: 'Set up a backup',
      to: '/graph/backup/new',
    });
    expect(derive(snapshot, { role: 'viewer' }).status.doors.backup.to).toBe(
      '/graph/backup/inventory'
    );
  });
});

describe('deriveMongoHub environment filter', () => {
  it('offers every environment of the estate whatever the filter', () => {
    expect(
      derive(wrongSnapshot(NOW, false), { environment: 'prod' }).environment
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
    const view = derive(wrongSnapshot(NOW, false), { environment: 'staging' });

    expect(view.clusters.map((node) => node.name)).toEqual(['reports']);
    expect(view.status).toMatchObject({ clusters: 1, needAttention: 1 });
    // Every schedule and artifact belongs to prod.
    expect(view.backups.state).toBe('notSetUp');
    // Query analytics is filtered by the server, so sample queries stay whatever the filter.
    expect(
      view.attention
        .filter((item) => item.kind !== 'querySlower')
        .map((item) => item.kind)
    ).toEqual(['memberNoData']);
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

    expect(view.actions).toEqual({
      addService: {
        label: 'Add MongoDB service',
        to: '/graph/add-instance/mongodb',
      },
      install: {
        label: 'Install MongoDB',
        to: '/operations/nodes?filter=unmonitored',
        preview: 'developer',
      },
    });
    expect(view.sectionDoors.clusters).toEqual({
      label: 'Fleet in Operations',
      to: '/operations?tab=clusters',
      preview: 'developer',
    });
    expect(view.apps.map((app) => app.id)).toEqual([
      'operations',
      'backups',
      'diagnostics',
    ]);
  });

  it('opens Instances overview without Operations and drops its card', () => {
    const view = derive(healthySnapshot(NOW, false));

    expect(view.actions.install).toBeUndefined();
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

    expect(view.actions).toEqual({});
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
      { ...wrongSnapshot(NOW, false), alerts: null },
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
      kind: 'noPrimary',
      title: "billing has no primary, it can't take writes",
    });
    expect(view.attention.some((item) => item.kind === 'alert')).toBe(false);
  });

  it('fails the attention section outright when metrics failed', () => {
    expect(
      derive(
        { ...wrongSnapshot(NOW, false), metrics: null, alerts: null },
        {
          sources: sourcesWith('ready', {
            metrics: { status: 'error' },
            alerts: { status: 'error' },
          }),
        }
      ).sections.attention
    ).toEqual({ status: 'error', source: 'metrics' });
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
  it('expands the problem path', () => {
    expect(derive(wrongSnapshot(NOW, false)).expanded).toMatchObject({
      'cluster:prod/billing': true,
      'cluster:prod/orders': true,
    });
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
        (item) => item.kind
      )
    ).toEqual(['alert']);
  });
});
