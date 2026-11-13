import type { CombinedSettings } from 'contexts/settings';
import {
  EXTENSIONS_ATW_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
  OM_PATH,
} from 'lib/constants';
import type { HAInfo } from 'types/ha.types';
import type { NavItem } from 'types/navigation.types';
import { ServiceType } from 'types/services.types';
import { UpdateStatus } from 'types/updates.types';
import { findActiveNavItem } from 'utils/navigation.utils';
import {
  TEST_USER_ADMIN,
  TEST_USER_ANONYMOUS,
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
      'advisors',
      'alerts',
      'extensions-atw',
      'explore',
      'qan',
      'realtime-queries',
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
      'app-operations-for-mongodb',
      'extensions-mysql-backups',
      'section-backups',
      'backup-inventory',
      'scheduled-backups',
      'restore-history',
      'storage-locations',
    ]);
    expect(findById(apps, 'app-operations-for-mongodb')?.url).toBe(OM_PATH);
  });

  it.each([
    ['app-operations-for-mongodb', `${OM_PATH}/automations`],
    ['scheduled-backups', '/graph/backup/scheduled'],
    ['backup-inventory', '/graph/backup/new'],
    ['extensions-mysql-backups', `${EXTENSIONS_MYSQL_BACKUPS_PATH}/backups/1`],
  ])('keeps %s highlighted on its own pages', (id, path) => {
    expect(findActiveNavItem(build(), path)?.id).toBe(id);
  });

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

  it('offers a quick add on the inventory rows', () => {
    const rows = findById(build(), 'inventory')?.children;

    expect(rows?.map((row) => row.action?.label)).toEqual([
      'Add a service',
      'Add a node',
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

    const ha = findById(
      build({
        haInfo: {
          enabled: true,
          health: 'healthy',
          nodes: [],
          namespace: 'pmm',
        },
      }),
      'high-availability'
    );

    expect(ha?.badge).toBeDefined();
    expect(ha?.url).toContain('var-namespace=pmm');
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
      'app-operations-for-mongodb',
      'extensions-mysql-backups',
    ]);
  });

  it('offers sign in instead of the account when logged out', () => {
    const tree = build({ user: TEST_USER_ANONYMOUS, isLoggedIn: false });

    expect(ids(tree)).toContain('sign-in');
    expect(ids(tree)).not.toContain('account');
    expect(ids(tree)).not.toContain('apps');
    expect(ids(tree)).not.toContain('extensions-atw');
  });
});
