import {
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import {
  addAlerting,
  addSection,
  addExtensionsApps,
  addDashboardItems,
  addOm,
} from './navigation.utils';
import {
  NAV_EXTENSIONS_ATW,
  NAV_EXTENSIONS_MYSQL_BACKUPS,
  NAV_MONGO,
} from './navigation.constants';
import {
  DEFAULT_EXTENSIONS_APP_ICON,
  type ExtensionsApp,
} from 'extensions/extensionsApps';
import { ServiceType } from 'types/services.types';
import {
  EXTENSIONS_ATW_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
  EXTENSIONS_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
} from 'lib/constants';
import { PreviewChip } from 'components/preview-chip';

const childIds = (item: ReturnType<typeof addAlerting>) =>
  (item.children || []).map((c) => c.id);

describe('addAlerting', () => {
  it('always includes alert rules', () => {
    expect(childIds(addAlerting())).toContain('alerts-rules');
  });

  it('does not add user-gated pages without a user', () => {
    const ids = childIds(addAlerting(true, true));

    expect(ids).toContain('alerts-rules');
    expect(ids).toContain('alerts-status');
    expect(ids).not.toContain('alerts-silences');
    expect(ids).not.toContain('alerts-groups');
    expect(ids).not.toContain('alerts-contact-points');
    expect(ids).not.toContain('alerts-policies');
    expect(ids).not.toContain('alerts-settings');
  });

  it('adds silences and groups for authenticated user when UA is off', () => {
    const ids = childIds(addAlerting(false, false, TEST_USER_VIEWER));

    expect(ids).toContain('alerts-silences');
    expect(ids).toContain('alerts-groups');
    expect(ids).not.toContain('alerts-contact-points');
    expect(ids).not.toContain('alerts-policies');
    expect(ids).not.toContain('alerts-settings');
  });

  it('adds contact points and policies for authenticated user when UA is on', () => {
    const ids = childIds(addAlerting(false, true, TEST_USER_VIEWER));

    expect(ids).toContain('alerts-silences');
    expect(ids).toContain('alerts-groups');
    expect(ids).toContain('alerts-contact-points');
    expect(ids).toContain('alerts-policies');
  });

  it('adds alert settings only for PMM admin when UA is on', () => {
    expect(childIds(addAlerting(false, true, TEST_USER_ADMIN))).toContain(
      'alerts-settings'
    );
    expect(childIds(addAlerting(false, true, TEST_USER_VIEWER))).not.toContain(
      'alerts-settings'
    );
    expect(childIds(addAlerting(false, false, TEST_USER_ADMIN))).not.toContain(
      'alerts-settings'
    );
  });

  it('adds status when alerting is enabled and templates only for editors', () => {
    expect(childIds(addAlerting(true, false))).toContain('alerts-status');
    expect(childIds(addAlerting(false, false))).not.toContain('alerts-status');

    expect(childIds(addAlerting(true, false, TEST_USER_EDITOR))).toContain(
      'alerts-templates'
    );
    expect(childIds(addAlerting(true, false, TEST_USER_VIEWER))).not.toContain(
      'alerts-templates'
    );
    expect(childIds(addAlerting(false, false, TEST_USER_EDITOR))).not.toContain(
      'alerts-templates'
    );
  });
});

const extensionsApp = (overrides: Partial<ExtensionsApp>): ExtensionsApp => ({
  key: 'pg_backups',
  slug: 'pg-backups',
  label: 'PostgreSQL Backups',
  icon: DEFAULT_EXTENSIONS_APP_ICON,
  order: 10,
  enabled: true,
  customUi: false,
  blockedBy: [],
  ...overrides,
});

const LISTED_ATW = extensionsApp({
  key: 'atw',
  slug: 'atw',
  label: 'ATW',
  order: 3,
  customUi: true,
});
const LISTED_MYSQL_BACKUPS = extensionsApp({
  key: 'mysql_backups',
  slug: 'mysql-backups',
  label: 'MySQL backups',
  order: 8,
});

const childIdsOf = (items: ReturnType<typeof addExtensionsApps>) =>
  (items[0]?.children || []).map((child) => child.id);

describe('addExtensionsApps', () => {
  it('returns one Apps section instead of two flat entries', () => {
    const items = addExtensionsApps();

    expect(items).toHaveLength(1);
    expect(items[0]).toMatchObject({ id: 'management', text: 'Apps' });
  });

  it('marks the section as a tech preview', () => {
    expect(addExtensionsApps()[0]).toMatchObject({
      badgeAlwaysVisible: true,
      badge: { type: PreviewChip, props: { level: 'tech' } },
    });
  });

  it('gives the section no page of its own', () => {
    expect(addExtensionsApps()[0].url).toBeUndefined();
  });

  describe('before the app list arrives', () => {
    it('shows the apps with a dedicated route, in nav order', () => {
      const children = addExtensionsApps()[0].children || [];

      expect(children.map((child) => child.id)).toEqual([
        'extensions-atw',
        'extensions-mysql-backups',
      ]);
      expect(children[0]).toMatchObject({
        url: EXTENSIONS_ATW_PATH,
        matches: ['*'],
      });
      expect(children[1]).toMatchObject({
        url: EXTENSIONS_MYSQL_BACKUPS_PATH,
        matches: ['*'],
      });
    });
  });

  describe('from the app list', () => {
    it('adds a listed schema-driven app under its slug', () => {
      const items = addExtensionsApps([extensionsApp({})]);

      expect(items[0].children).toEqual([
        {
          id: 'extensions-pg-backups',
          icon: DEFAULT_EXTENSIONS_APP_ICON,
          text: 'PostgreSQL Backups',
          url: `${EXTENSIONS_PATH}/pg-backups`,
          matches: ['*'],
        },
      ]);
    });

    it('follows the order of the list', () => {
      const items = addExtensionsApps([
        LISTED_ATW,
        LISTED_MYSQL_BACKUPS,
        extensionsApp({}),
      ]);

      expect(childIdsOf(items)).toEqual([
        'extensions-atw',
        'extensions-mysql-backups',
        'extensions-pg-backups',
      ]);
    });

    it('keeps the label and icon of an app with a dedicated route', () => {
      const [atw, mysqlBackups] =
        addExtensionsApps([LISTED_ATW, LISTED_MYSQL_BACKUPS])[0].children || [];

      expect(atw).toBe(NAV_EXTENSIONS_ATW);
      expect(mysqlBackups).toBe(NAV_EXTENSIONS_MYSQL_BACKUPS);
    });

    it('drops a disabled app, including one with a dedicated route', () => {
      const items = addExtensionsApps([
        { ...LISTED_ATW, enabled: false },
        LISTED_MYSQL_BACKUPS,
        extensionsApp({ enabled: false }),
      ]);

      expect(childIdsOf(items)).toEqual(['extensions-mysql-backups']);
    });

    it('drops a custom-page app that has no dedicated route', () => {
      const items = addExtensionsApps([
        LISTED_MYSQL_BACKUPS,
        extensionsApp({ customUi: true }),
      ]);

      expect(childIdsOf(items)).toEqual(['extensions-mysql-backups']);
    });

    it('drops the section when no listed app has an item', () => {
      expect(addExtensionsApps([])).toEqual([]);
      expect(addExtensionsApps([{ ...LISTED_ATW, enabled: false }])).toEqual(
        []
      );
    });
  });
});

describe('addSection', () => {
  it('wraps the children in the section', () => {
    const items = addSection({ id: 'section' }, [{ id: 'child' }]);

    expect(items).toEqual([{ id: 'section', children: [{ id: 'child' }] }]);
  });

  it('contributes nothing rather than an empty expandable shell', () => {
    expect(addSection({ id: 'section' }, [])).toEqual([]);
  });
});

describe('addOm', () => {
  it('is the Operations item, marked as a Preview', () => {
    const om = addOm();

    expect(om.text).toBe('Operations');
    expect(om.badge).toMatchObject({ label: 'Preview' });
    expect(om.badgeAlwaysVisible).toBe(true);
  });
});

describe('addDashboardItems', () => {
  const mongoMenu = (items: ReturnType<typeof addDashboardItems>) =>
    items.find((item) => item.id === NAV_MONGO.id);

  it('nests the extras at the end of the MongoDB menu', () => {
    const menu = mongoMenu(
      addDashboardItems([ServiceType.mongodb], [], undefined, [addOm()])
    );

    expect(menu?.children?.at(-1)?.id).toBe('om');
    expect(menu?.children?.length).toBe((NAV_MONGO.children?.length ?? 0) + 1);
  });

  it('leaves the shared MongoDB constant untouched', () => {
    const before = NAV_MONGO.children?.length;

    addDashboardItems([ServiceType.mongodb], [], undefined, [addOm()]);
    addDashboardItems([ServiceType.mongodb], [], undefined, [addOm()]);

    expect(NAV_MONGO.children?.length).toBe(before);
  });

  it('keeps the MongoDB menu for the extras when no MongoDB is monitored', () => {
    const menu = mongoMenu(addDashboardItems([], [], undefined, [addOm()]));

    expect(menu?.children?.map((c) => c.id)).toEqual(['om']);
    expect(menu?.url).toBe(addOm().url);
  });

  it('has no MongoDB menu with neither MongoDB nor extras', () => {
    expect(mongoMenu(addDashboardItems([], []))).toBeUndefined();
  });

  it('links the ClusterSync dashboard from the MongoDB section', () => {
    const clusterSync = mongoMenu(
      addDashboardItems([ServiceType.mongodb], [])
    )?.children?.find((c) => c.id === 'mongo-clustersync-details');

    expect(clusterSync).toEqual({
      id: 'mongo-clustersync-details',
      text: 'ClusterSync',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-clustersync-details/mongodb-clustersync-details`,
    });
  });

  it('shows the MongoDB section, and so ClusterSync, only with a MongoDB service', () => {
    const mongoChildIds = (types: ServiceType[]) =>
      mongoMenu(addDashboardItems(types, []))?.children?.map((c) => c.id);

    expect(mongoChildIds([ServiceType.external])).toBeUndefined();
    expect(mongoChildIds([ServiceType.mongodb])).toContain(
      'mongo-clustersync-details'
    );
  });
});
