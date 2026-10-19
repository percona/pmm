import {
  EXTENSIONS_ATW_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
  MONGODB_HUB_PATH,
  PROTOTYPE_PLACEHOLDER_PATH,
} from 'lib/constants';
import { HAInfo } from 'types/ha.types';
import { NavItem } from 'types/navigation.types';
import { UpdateStatus } from 'types/updates.types';
import { findActiveNavItem } from 'utils/navigation.utils';
import {
  TEST_USER_ADMIN,
  TEST_USER_ANONYMOUS,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import {
  buildProposedNavTree,
  ProposedNavTreeInput,
} from './navigation.proposal';

const HA_OFF: HAInfo = { enabled: false, health: 'unknown', nodes: [] };

const INNODB = {
  uid: 'mysql-innodb',
  title: 'MySQL InnoDB details',
  url: '/graph/d/mysql-innodb/mysql-innodb-details',
};
const DISK = {
  uid: 'node-disk',
  title: 'Disk details',
  url: '/graph/d/node-disk/disk-details',
};

const build = (overrides: Partial<ProposedNavTreeInput> = {}) =>
  buildProposedNavTree({
    user: TEST_USER_ADMIN,
    isLoggedIn: true,
    haInfo: HA_OFF,
    updateStatus: UpdateStatus.UpToDate,
    colorMode: 'light',
    toggleColorMode: () => {},
    ...overrides,
  });

const ids = (items: NavItem[] = []) => items.map((item) => item.id);

const findById = (items: NavItem[] = [], id: string) =>
  items.find((item) => item.id === id);

describe('buildProposedNavTree', () => {
  it('lays the sidebar out as the proposed flat, sectioned list', () => {
    expect(ids(build())).toEqual([
      'home-page',
      'section-starred',
      'pinned-mongodb-backups',
      'pinned-mysql-backups',
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
      'high-availability',
      'configuration',
      'account',
      'help',
    ]);
  });

  it('opens on Overview', () => {
    expect(build()[0]).toMatchObject({ id: 'home-page', text: 'Overview' });
  });

  it('marks the sample entries as pinned', () => {
    const pinned = build().filter((item) => item.id.startsWith('pinned-'));

    expect(pinned).toHaveLength(2);
    expect(pinned.every((item) => item.pinned)).toBe(true);
  });

  it('makes each technology a single entry that opens its overview dashboard', () => {
    const mysql = findById(build(), 'mysql');

    expect(mysql?.children).toBeUndefined();
    expect(mysql?.url).toBe(
      '/graph/d/mysql-instance-overview/mysql-instances-overview'
    );
  });

  it('opens the MongoDB hub and keeps MongoDB highlighted on it and on its dashboards', () => {
    const tree = build();

    expect(findById(tree, 'mongo')?.url).toBe(MONGODB_HUB_PATH);
    expect(findActiveNavItem(tree, MONGODB_HUB_PATH)?.id).toBe('mongo');
    expect(
      findActiveNavItem(
        tree,
        '/graph/d/mongodb-instance-overview/mongodb-instances-overview'
      )?.id
    ).toBe('mongo');
    expect(
      findActiveNavItem(
        tree,
        '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary'
      )?.id
    ).toBe('mongo');
  });

  it('keeps the other technologies on their overview dashboards', () => {
    const tree = build();

    expect(findById(tree, 'postgre')?.url).toBe(
      '/graph/d/postgresql-instance-overview/postgresql-instances-overview'
    );
    expect(findById(tree, 'system')?.url).toBe(
      '/graph/d/node-instance-overview/nodes-overview'
    );
    expect(
      ['mysql', 'postgre', 'valkey', 'system'].map(
        (id) => findById(tree, id)?.matches?.includes(MONGODB_HUB_PATH) ?? false
      )
    ).toEqual([false, false, false, false]);
  });

  it('keeps a technology highlighted on any of its dashboards', () => {
    const tree = build();

    expect(
      findActiveNavItem(tree, '/graph/d/mysql-table/mysql-table-details')?.id
    ).toBe('mysql');
    expect(findActiveNavItem(tree, '/graph/d/node-disk/disk-details')?.id).toBe(
      'system'
    );
  });

  it('lists the dashboards starred in Grafana after the starred apps', () => {
    const tree = build({ starredDashboards: [INNODB, DISK] });
    const start = tree.findIndex((item) => item.id === 'section-starred');

    expect(ids(tree.slice(start + 1, start + 5))).toEqual([
      'pinned-mongodb-backups',
      'pinned-mysql-backups',
      'starred-dashboard-mysql-innodb',
      'starred-dashboard-node-disk',
    ]);
    expect(findById(tree, 'starred-dashboard-mysql-innodb')).toMatchObject({
      text: 'MySQL InnoDB details',
      url: '/graph/d/mysql-innodb/mysql-innodb-details',
      pinned: true,
    });
  });

  it('lets a starred dashboard claim its page ahead of the technology, whatever its slug', () => {
    const tree = build({ starredDashboards: [INNODB] });

    expect(
      findActiveNavItem(tree, '/graph/d/mysql-innodb/mysql-innodb-details')?.id
    ).toBe('starred-dashboard-mysql-innodb');
    expect(findActiveNavItem(tree, '/graph/d/mysql-innodb/renamed')?.id).toBe(
      'starred-dashboard-mysql-innodb'
    );
    expect(
      findActiveNavItem(build(), '/graph/d/mysql-innodb/mysql-innodb-details')
        ?.id
    ).toBe('mysql');
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

  it('lists the alert pages under one Alerts entry', () => {
    expect(ids(findById(build(), 'alerts')?.children)).toEqual([
      'alerts-status',
      'alerts-rules',
      'alerts-templates',
      'alerts-contact-points',
      'alerts-policies',
      'alerts-silences',
      'alerts-groups',
      'alerts-settings',
    ]);
  });

  it('lists the projected app catalog without the pinned apps', () => {
    const apps = findById(build(), 'apps')?.children;

    expect(apps?.map((app) => app.text)).toEqual([
      'Data archiving',
      'Operations for MongoDB',
      'PostgreSQL backups',
      'ProxySQL manager',
      'Replication checksums',
      'Schema changes',
      'Valkey backups',
      'All apps',
    ]);
    expect(findById(apps, 'app-schema-changes')?.url).toBe(
      `${PROTOTYPE_PLACEHOLDER_PATH}/schema-changes`
    );
  });

  it('offers a quick add on the inventory rows', () => {
    const rows = findById(build(), 'inventory')?.children;

    expect(rows?.map((row) => row.action?.label)).toEqual([
      'Add a service',
      'Add a node',
    ]);
  });

  it('keeps a deep link into a pinned app active on its starred entry', () => {
    const tree = build();
    const backups = findActiveNavItem(
      tree,
      `${EXTENSIONS_MYSQL_BACKUPS_PATH}/backups/123`
    );

    expect(backups?.id).toBe('pinned-mysql-backups');
    expect(backups?.url).toBe(EXTENSIONS_MYSQL_BACKUPS_PATH);
    expect(tree).toContain(backups);
  });

  it('lists Diagnostics with the analysis tools and leaves Help a single entry', () => {
    const tree = build();
    const diagnostics = findById(tree, 'extensions-atw');

    expect(diagnostics?.text).toBe('Diagnostics');
    expect(diagnostics?.url).toBe(EXTENSIONS_ATW_PATH);
    expect(
      findActiveNavItem(tree, `${EXTENSIONS_ATW_PATH}/incidents/abc`)?.id
    ).toBe('extensions-atw');
    expect(ids(findById(tree, 'apps')?.children)).not.toContain(
      'extensions-atw'
    );
    expect(findById(tree, 'help')?.children).toBeUndefined();
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
    expect(findById(children, 'section-organization')?.type).toBe(
      'menu-section'
    );
    expect(ids(tree)).not.toContain('users-and-access');
  });

  it('keeps the Account at the top level, named after the signed-in user', () => {
    const account = findById(build(), 'account');

    expect(account?.text).toBe(
      `Account: ${TEST_USER_ADMIN.name.split(' ')[0]}`
    );
    expect(account?.icon).toBeDefined();
    expect(ids(account?.children)).toContain('sign-out');
  });

  it('shows PMM HA even when the instance does not run in HA', () => {
    const ha = findById(build(), 'high-availability');

    expect(ha?.text).toBe('PMM HA');
    expect(ha?.badge).toBeUndefined();
  });

  it('adds the live status badge when the instance runs in HA', () => {
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

  it('shows a viewer the same structure as an admin', () => {
    expect(ids(build({ user: TEST_USER_VIEWER }))).toEqual(ids(build()));
  });

  it('offers sign in instead of the account when logged out', () => {
    const tree = build({ user: TEST_USER_ANONYMOUS, isLoggedIn: false });

    expect(ids(tree)).toContain('sign-in');
    expect(ids(tree)).not.toContain('account');
  });
});
