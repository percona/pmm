import type { CombinedSettings } from 'contexts/settings';
import {
  EXTENSIONS_ATW_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
  OM_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
} from 'lib/constants';
import type { HAInfo } from 'types/ha.types';
import type { NavItem } from 'types/navigation.types';
import { ServiceType } from 'types/services.types';
import { UpdateStatus } from 'types/updates.types';
import { findActiveNavItem } from 'utils/navigation.utils';
import {
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { buildNavTree, type NavTreeInput } from './navigation.tree';
import { NAV_MYSQL } from './navigation.constants';

const HA_OFF: HAInfo = { enabled: false, health: 'unknown', nodes: [] };

const ALL_TECHNOLOGIES = [
  ServiceType.mongodb,
  ServiceType.mysql,
  ServiceType.posgresql,
  ServiceType.valkey,
];

const SETTINGS = {
  advisorEnabled: true,
  alertingEnabled: true,
  backupManagementEnabled: true,
  enableAccessControl: true,
  extensionsEnabled: true,
  omEnabled: true,
  frontend: {
    apps: {},
    exploreEnabled: true,
    unifiedAlertingEnabled: true,
    disableLoginForm: false,
    auth: { disableLogin: false },
  },
} as unknown as CombinedSettings;

const build = (overrides: Partial<NavTreeInput> = {}) =>
  buildNavTree({
    user: TEST_USER_ADMIN,
    isLoggedIn: true,
    haInfo: HA_OFF,
    settings: SETTINGS,
    serviceTypes: ALL_TECHNOLOGIES,
    folders: [],
    updateStatus: UpdateStatus.UpToDate,
    colorMode: 'light',
    toggleColorMode: () => {},
    ...overrides,
  });

const ids = (items: NavItem[] = []) => items.map((item) => item.id);

const findById = (items: NavItem[] = [], id: string) =>
  items.find((item) => item.id === id);

describe('buildNavTree', () => {
  it('lays the sidebar out as a flat, sectioned list', () => {
    expect(ids(build())).toEqual([
      'section-technologies',
      'mongo',
      'mysql',
      'postgre',
      'valkey',
      'system',
      'section-analysis',
      'qan',
      'realtime-queries',
      'alerts',
      'advisors',
      'explore',
      'extensions-atw',
      'section-browse',
      'dashboards',
      'apps',
      'inventory',
      'section-administration',
      'configuration',
      'account',
      'help',
    ]);
  });

  it('leaves the home dashboard to the logo', () => {
    expect(ids(build())).not.toContain('home-page');
  });

  it('has no Starred section', () => {
    expect(ids(build())).not.toContain('section-starred');
  });

  it('lists only the technologies PMM monitors, and always the OS', () => {
    const tree = build({ serviceTypes: [ServiceType.mysql] });

    expect(ids(tree)).toContain('mysql');
    expect(ids(tree)).toContain('system');
    expect(ids(tree)).not.toContain('mongo');
    expect(ids(tree)).not.toContain('postgre');
    expect(ids(tree)).not.toContain('valkey');
  });

  it('keeps each technology menu as it was', () => {
    const mysql = findById(build(), 'mysql');

    expect(mysql).toBe(NAV_MYSQL);
    expect(mysql?.children?.length).toBeGreaterThan(0);
  });

  it('links the ClusterSync dashboard from the MongoDB menu', () => {
    expect(
      findById(
        findById(build(), 'mongo')?.children,
        'mongo-clustersync-details'
      )
    ).toEqual({
      id: 'mongo-clustersync-details',
      text: 'ClusterSync',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-clustersync-details/mongodb-clustersync-details`,
    });
  });

  it('highlights the open dashboard inside its technology menu', () => {
    const tree = build();
    const active = findActiveNavItem(
      tree,
      '/graph/d/mysql-table/mysql-table-details'
    );

    expect(active?.id).toBe('mysql-table-details');
    expect(findById(tree, 'mysql')?.children).toContain(active);
  });

  it('adds the technology folder of other dashboards when Grafana has one', () => {
    const tree = build({
      folders: [
        { id: 1, uid: 'abc', title: 'MySQL' },
      ] as NavTreeInput['folders'],
    });
    const children = findById(tree, 'mysql')?.children;

    expect(children?.at(-1)).toMatchObject({
      id: 'mysql-other-dashboards',
      text: 'Other dashboards',
      url: '/graph/dashboards/f/abc/mysql',
    });
    expect(NAV_MYSQL.children).not.toContainEqual(
      expect.objectContaining({ id: 'mysql-other-dashboards' })
    );
  });

  it('lists ProxySQL and HAProxy under Dashboards while monitored', () => {
    const children = ids(
      findById(
        build({ serviceTypes: [ServiceType.proxysql, ServiceType.haproxy] }),
        'dashboards'
      )?.children
    );

    expect(children).toContain('proxysql');
    expect(children).toContain('haproxy');
    expect(ids(findById(build(), 'dashboards')?.children)).not.toContain(
      'proxysql'
    );
  });

  it('highlights a technology folder in its own menu, not under Dashboards', () => {
    const tree = build({
      folders: [
        { id: 1, uid: 'abc', title: 'MySQL' },
      ] as NavTreeInput['folders'],
    });

    expect(findActiveNavItem(tree, '/graph/dashboards/f/abc/mysql')?.id).toBe(
      'mysql-other-dashboards'
    );
  });

  it('splits stored metrics and real-time queries into two entries', () => {
    const tree = build();

    expect(
      findActiveNavItem(tree, '/graph/d/pmm-qan/pmm-query-analytics')?.id
    ).toBe('qan');
    expect(findActiveNavItem(tree, '/rta')?.id).toBe('realtime-queries');
    expect(findActiveNavItem(tree, '/rta/overview')?.id).toBe(
      'realtime-queries'
    );
  });

  it('opens the analysis section with the same entries for every role', () => {
    const analysis = (tree: NavItem[]) => {
      const all = ids(tree);

      return all.slice(
        all.indexOf('section-analysis') + 1,
        all.indexOf('section-browse')
      );
    };
    const shared = ['qan', 'realtime-queries', 'alerts'];

    expect(analysis(build({ user: TEST_USER_VIEWER }))).toEqual([
      ...shared,
      'extensions-atw',
    ]);
    expect(analysis(build({ user: TEST_USER_EDITOR })).slice(0, 3)).toEqual(
      shared
    );
    expect(analysis(build()).slice(0, 3)).toEqual(shared);
  });

  it('keeps the Advisors pages as a group', () => {
    expect(ids(findById(build(), 'advisors')?.children)).toEqual([
      'advisors-insights',
      'advisors-checks',
      'advisors-runs',
      'advisors-scheduler',
    ]);
  });

  it('lists the real apps only', () => {
    const apps = findById(build(), 'apps')?.children;

    expect(ids(apps)).toEqual([
      'extensions-mysql-backups',
      'section-operations-for-mongodb',
      'om-fleet',
      'om-nodes',
      'om-automations',
      'om-settings',
      'section-backups',
      'backup-inventory',
      'scheduled-backups',
      'restore-history',
      'storage-locations',
    ]);
    expect(findById(apps, 'om-fleet')?.url).toBe(OM_PATH);
  });

  it('marks the Operations for MongoDB heading as a developer preview', () => {
    const heading = findById(
      findById(build(), 'apps')?.children,
      'section-operations-for-mongodb'
    );

    expect(heading?.badge).toBeDefined();
    expect(heading?.badgeAlwaysVisible).toBe(true);
  });

  it.each([
    ['om-fleet', OM_PATH],
    ['om-nodes', `${OM_PATH}/nodes/install`],
    ['om-automations', `${OM_PATH}/automations`],
    ['om-settings', `${OM_PATH}/settings`],
    ['scheduled-backups', '/graph/backup/scheduled'],
    ['backup-inventory', '/graph/backup/new'],
    ['extensions-mysql-backups', `${EXTENSIONS_MYSQL_BACKUPS_PATH}/backups/1`],
  ])('keeps %s highlighted on its own pages', (id, path) => {
    expect(findActiveNavItem(build(), path)?.id).toBe(id);
  });

  it.each(['extensions-atw', 'extensions-mysql-backups'])(
    'marks the PMM Extensions app %s as a technical preview',
    (id) => {
      const tree = build();
      const item =
        findById(tree, id) ?? findById(findById(tree, 'apps')?.children, id);

      expect(item?.badge).toBeDefined();
      expect(item?.badgeAlwaysVisible).toBe(true);
    }
  );

  it('drops Apps when no app is available', () => {
    const tree = build({
      settings: {
        ...SETTINGS,
        omEnabled: false,
        backupManagementEnabled: false,
        extensionsEnabled: false,
      },
    });

    expect(ids(tree)).not.toContain('apps');
  });

  it('offers a quick add only on Services, since nodes come with services', () => {
    const rows = findById(build(), 'inventory')?.children;

    expect(rows?.map((row) => row.action?.label)).toEqual([
      'Add a service',
      undefined,
    ]);
  });

  it('lists Diagnostics with the analysis tools', () => {
    const tree = build();

    expect(findById(tree, 'extensions-atw')?.text).toBe('Diagnostics');
    expect(
      findActiveNavItem(tree, `${EXTENSIONS_ATW_PATH}/incidents/abc`)?.id
    ).toBe('extensions-atw');
  });

  it('lists the Org. management and Users and access pages directly under Configuration, with headings', () => {
    const tree = build();
    const children = findById(tree, 'configuration')?.children;

    expect(ids(children)).toEqual([
      'configuration-settings',
      'updates',
      'section-organization',
      'organizations',
      'stats-and-licenses',
      'default-preferences',
      'section-users-and-access',
      'users',
      'teams',
      'service-accounts',
      'rbac-roles',
    ]);
    expect(children?.every((child) => !child.children)).toBe(true);
    expect(ids(tree)).not.toContain('users-and-access');
  });

  it('lists Access roles only while access control is on', () => {
    const children = findById(
      build({ settings: { ...SETTINGS, enableAccessControl: false } }),
      'configuration'
    )?.children;

    expect(ids(children)).not.toContain('rbac-roles');
  });

  it('keeps the Account at the top level, named after the signed-in user', () => {
    const account = findById(build(), 'account');

    expect(account?.text).toBe(
      `Account: ${TEST_USER_ADMIN.name.split(' ')[0]}`
    );
    expect(ids(account?.children)).toContain('sign-out');
  });

  it('lists PMM HA only when the instance runs in HA, with its live status', () => {
    expect(ids(build())).not.toContain('high-availability');

    const tree = build({
      haInfo: {
        enabled: true,
        health: 'healthy',
        nodes: [],
        namespace: 'pmm',
      },
    });
    const ha = findById(tree, 'high-availability');

    expect(ha?.badge).toBeDefined();
    expect(ha?.url).toContain('var-namespace=pmm');
    expect(
      findActiveNavItem(
        tree,
        '/graph/d/pmm-ha-health-overview/pmm-ha-health-overview'
      )?.id
    ).toBe('high-availability');
    expect(ha?.children).toBeUndefined();
  });

  it('gives an editor the analysis tools but no administration', () => {
    const tree = ids(build({ user: TEST_USER_EDITOR }));

    expect(tree).toContain('advisors');
    expect(tree).toContain('explore');
    expect(tree).not.toContain('inventory');
    expect(tree).not.toContain('section-administration');
    expect(tree).not.toContain('configuration');
  });

  it('gives a viewer neither the editor tools nor the admin entries', () => {
    const tree = build({ user: TEST_USER_VIEWER });

    expect(ids(tree)).not.toContain('advisors');
    expect(ids(tree)).not.toContain('explore');
    expect(ids(tree)).not.toContain('inventory');
    expect(ids(findById(tree, 'apps')?.children)).toEqual([
      'extensions-mysql-backups',
      'section-operations-for-mongodb',
      'om-fleet',
      'om-nodes',
      'om-automations',
      'om-settings',
    ]);
  });

  it('offers an anonymous visitor sign in and nothing that needs PMM APIs', () => {
    const tree = ids(build({ user: undefined, isLoggedIn: false }));

    expect(tree).toContain('sign-in');
    for (const id of [
      'qan',
      'realtime-queries',
      'alerts',
      'extensions-atw',
      'apps',
      'account',
      'help',
    ]) {
      expect(tree).not.toContain(id);
    }
  });
});
