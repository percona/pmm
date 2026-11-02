import type { CombinedSettings } from 'contexts/settings';
import type { VersionResponse } from 'types/version.types';
import { KEYWORDS } from '../search.keywords';
import {
  ALL_FEATURES_ON,
  AMI_VERSION,
  staticIndexFor,
} from '../search.testFixtures';
import {
  actionItems,
  extraPages,
  helpItems,
  settingsItems,
  type StaticSourceContext,
} from './static.source';

const context = (
  overrides: Partial<StaticSourceContext> = {}
): StaticSourceContext => ({
  isAdmin: true,
  isEditor: true,
  settings: ALL_FEATURES_ON,
  version: AMI_VERSION,
  startTour: vi.fn(),
  ...overrides,
});

const labels = (items: { label: string }[]) => items.map((item) => item.label);

const FEATURES_OFF = {
  advisorEnabled: false,
  backupManagementEnabled: false,
  extensionsEnabled: false,
  omEnabled: false,
} as CombinedSettings;

const OPERATIONS_PAGES = [
  'Clusters',
  'Services',
  'Nodes',
  'Installs',
  'Scans',
  'Operations settings',
];

describe('extraPages', () => {
  it('lists the curated pages for an admin with every feature on', () => {
    expect(extraPages(context()).map(({ label, to }) => [label, to])).toEqual([
      ['Client updates', '/updates/clients'],
      ['Configuration advisors', '/graph/advisors/configuration'],
      ['Performance advisors', '/graph/advisors/performance'],
      ['Query advisors', '/graph/advisors/query'],
      ['Security advisors', '/graph/advisors/security'],
      ['Backup inventory', '/graph/backup/inventory'],
      ['Scheduled backups', '/graph/backup/scheduled'],
      ['Restores', '/graph/backup/restore'],
      ['Storage locations', '/graph/backup/locations'],
      ['MySQL backup schedule', '/extensions/mysql-backups/schedule'],
      ['Clusters', '/operations?tab=clusters'],
      ['Services', '/operations?tab=services'],
      ['Nodes', '/operations/nodes'],
      ['Installs', '/operations/automations?tab=installs'],
      ['Scans', '/operations/automations?tab=scans'],
      ['Operations settings', '/operations/settings'],
      ['Plugins', '/graph/plugins'],
      ['Data sources', '/graph/connections/datasources'],
    ]);
  });

  it.each([
    ['an admin with Operations on', context(), OPERATIONS_PAGES],
    ['an admin with Operations off', context({ settings: FEATURES_OFF }), []],
    ['an admin before settings load', context({ settings: null }), []],
    ['an editor with Operations on', context({ isAdmin: false }), []],
  ])('lists the Operations pages for %s only', (_, ctx, expected) => {
    expect(
      labels(extraPages(ctx).filter((item) => item.id.startsWith('page:om-')))
    ).toEqual(expected);
  });

  it('files the Operations pages under the app, and each tab under its page', () => {
    const pages = extraPages(context());
    const descriptionOf = (id: string) =>
      pages.find((item) => item.id === id)?.description;

    expect(descriptionOf('page:om-clusters')).toEqual([
      'Apps',
      'Operations for MongoDB',
      'Fleet',
    ]);
    expect(descriptionOf('page:om-scans')).toEqual([
      'Apps',
      'Operations for MongoDB',
      'Automations',
    ]);
    expect(descriptionOf('page:om-nodes')).toEqual([
      'Apps',
      'Operations for MongoDB',
    ]);
  });

  it.each([
    ['an admin with PMM Extensions on', context(), ['MySQL backup schedule']],
    [
      'an admin with PMM Extensions off',
      context({ settings: FEATURES_OFF }),
      [],
    ],
    ['an admin before settings load', context({ settings: null }), []],
    ['an editor with PMM Extensions on', context({ isAdmin: false }), []],
  ])('lists the MySQL backup schedule for %s only', (_, ctx, expected) => {
    expect(
      labels(extraPages(ctx).filter((item) => item.label.includes('MySQL')))
    ).toEqual(expected);
  });

  it('keys a page addressed by its query apart from the page it opens on', () => {
    const keyOf = (id: string) =>
      extraPages(context()).find((item) => item.id === id)?.dedupeKey;

    expect(keyOf('page:om-clusters')).toBe('page:om-clusters');
    expect(keyOf('page:om-installs')).toBe('page:om-installs');
    expect(keyOf('page:om-nodes')).toBe('/operations/nodes');
    expect(keyOf('page:backup-inventory')).toBe('/graph/backup/inventory');
  });

  it('counts a feature as on unless the server turns it off', () => {
    expect(labels(extraPages(context({ settings: null })))).toContain(
      'Query advisors'
    );
    expect(
      labels(extraPages(context({ isAdmin: false, settings: FEATURES_OFF })))
    ).toEqual(['Client updates']);
  });
});

describe('settingsItems', () => {
  it('is empty for anyone but an admin', () => {
    expect(settingsItems(context({ isAdmin: false }))).toEqual([]);
  });

  it('shows the SSH key tab on AMI and ServiceNow with extensions on', () => {
    const items = settingsItems(context());

    expect(labels(items).slice(0, 4)).toEqual([
      'Metrics resolution',
      'Advanced settings',
      'SSH key',
      'ServiceNow connection',
    ]);
    expect(
      labels(
        settingsItems(
          context({
            settings: FEATURES_OFF,
            version: { distributionMethod: 'x' } as unknown as VersionResponse,
          })
        )
      ).slice(0, 3)
    ).toEqual(['Metrics resolution', 'Advanced settings', 'Data retention']);
  });

  it('keeps each field its own entry on the advanced tab', () => {
    const retention = settingsItems(context()).find(
      (item) => item.id === 'setting:data-retention'
    );

    expect(retention).toMatchObject({
      kind: 'setting',
      group: 'settings',
      to: '/settings/advanced-settings',
      dedupeKey: 'setting:data-retention',
      keywords: ['retention', 'storage'],
      description: ['Configuration', 'Settings', 'Advanced settings'],
    });
  });
});

describe('actionItems', () => {
  it.each([
    [
      'an admin',
      context(),
      [
        'Start the product tour',
        'Create an alert rule',
        'Create a dashboard',
        'Import a dashboard',
        'Create a backup',
        'New MySQL backup',
        'Install MongoDB',
        'Create a PMM dump',
      ],
    ],
    [
      'an admin before settings load',
      context({ settings: null }),
      [
        'Start the product tour',
        'Create an alert rule',
        'Create a dashboard',
        'Import a dashboard',
        'Create a backup',
        'Create a PMM dump',
      ],
    ],
    [
      'an admin without backups',
      context({ settings: FEATURES_OFF }),
      [
        'Start the product tour',
        'Create an alert rule',
        'Create a dashboard',
        'Import a dashboard',
        'Create a PMM dump',
      ],
    ],
    [
      'an editor',
      context({ isAdmin: false }),
      [
        'Start the product tour',
        'Create an alert rule',
        'Create a dashboard',
        'Import a dashboard',
      ],
    ],
    [
      'a viewer',
      context({ isAdmin: false, isEditor: false }),
      ['Start the product tour'],
    ],
  ])('offers %s only what they can do', (_, ctx, expected) => {
    expect(labels(actionItems(ctx))).toEqual(expected);
  });

  it('opens Nodes on the unmonitored ones and the MySQL backup form', () => {
    const actions = actionItems(context());
    const find = (id: string) => actions.find((item) => item.id === id);

    expect(find('action:om-install-mongodb')).toMatchObject({
      to: '/operations/nodes?filter=unmonitored',
      dedupeKey: 'action:om-install-mongodb',
      keywords: ['bootstrap', 'new replica set'],
    });
    expect(find('action:new-mysql-backup')).toMatchObject({
      to: '/extensions/mysql-backups/new',
      dedupeKey: '/extensions/mysql-backups/new',
      keywords: ['backup', 'restore', 'pitr'],
    });
  });

  it('starts the product tour', () => {
    const ctx = context();
    const [tour] = actionItems(ctx);

    tour.perform?.();

    expect(ctx.startTour).toHaveBeenCalledWith('product');
    expect(tour.keywords).toEqual(['tips', 'tour', 'onboarding']);
  });
});

describe('helpItems', () => {
  it('maps the help cards, skipping the tips card the tour action covers', () => {
    expect(
      helpItems(context()).map(({ id, label, to, href, external }) => [
        id,
        label,
        to ?? href,
        external ?? false,
      ])
    ).toEqual([
      [
        'help:pmm-docs',
        'PMM documentation',
        'https://per.co.na/pmm_documentation',
        true,
      ],
      ['help:support', 'Percona support', expect.any(String), true],
      ['help:forum', 'Percona forum', 'https://per.co.na/PMM3_forum', true],
      ['help:pmm-dump', 'PMM dump', '/graph/pmm-dump', false],
      ['help:pmm-logs', 'Download PMM logs', '/logs.zip', true],
      [
        'help:next-chapter',
        "Help shape PMM's next chapter",
        'https://per.co.na/pmm3_feedback',
        true,
      ],
    ]);
  });

  it('describes external items as opening a new tab', () => {
    const [docs] = helpItems(context());

    expect(docs).toMatchObject({
      description: 'Help · opens in a new tab',
      target: '_blank',
    });
  });

  it('hides admin-only cards from others', () => {
    const ids = helpItems(context({ isAdmin: false })).map((item) => item.id);

    expect(ids).not.toContain('help:pmm-dump');
    expect(ids).not.toContain('help:pmm-logs');
    expect(ids).toContain('help:pmm-docs');
  });
});

describe('search.keywords', () => {
  it('names only items of an admin index with every feature on', () => {
    const ids = new Set(staticIndexFor().map((item) => item.id));

    expect(Object.keys(KEYWORDS).filter((id) => !ids.has(id))).toEqual([]);
  });
});

describe('buildStaticIndex', () => {
  it('keeps each Operations tab and filter its own result beside the app', () => {
    const index = staticIndexFor();
    const find = (id: string) => index.find((item) => item.id === id);
    const app = find('page:app-operations-for-mongodb');

    expect(app?.to).toBe('/operations');
    expect(app?.keywords).not.toContain('Clusters');
    expect(app?.keywords).not.toContain('Services');
    expect(find('page:om-nodes')?.keywords).not.toContain('Install MongoDB');
    [
      'page:om-clusters',
      'page:om-services',
      'page:om-installs',
      'page:om-scans',
      'action:om-install-mongodb',
    ].forEach((id) => expect(find(id)).toBeDefined());
  });
});
