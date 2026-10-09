import {
  EXTENSIONS_ATW_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
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
      'section-my-navigation',
      'pinned-mongodb-backups',
      'pinned-mysql-innodb-details',
      'pinned-postgresql-query-analytics',
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

  it('marks the sample entries as pinned under a customizable heading', () => {
    const tree = build();
    const pinned = tree.filter((item) => item.id.startsWith('pinned-'));

    expect(findById(tree, 'section-my-navigation')?.action?.url).toBe(
      `${PROTOTYPE_PLACEHOLDER_PATH}/customize-navigation`
    );
    expect(pinned).toHaveLength(3);
    expect(pinned.every((item) => item.pinned)).toBe(true);
  });

  it('makes each technology a single entry that opens its overview dashboard', () => {
    const mysql = findById(build(), 'mysql');

    expect(mysql?.children).toBeUndefined();
    expect(mysql?.url).toBe(
      '/graph/d/mysql-instance-overview/mysql-instances-overview'
    );
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

  it('lets a pinned entry claim its page ahead of the technology', () => {
    expect(
      findActiveNavItem(build(), '/graph/d/mysql-innodb/mysql-innodb-details')
        ?.id
    ).toBe('pinned-mysql-innodb-details');
  });

  it('gives the pinned query analytics view only to its filter', () => {
    const tree = build();
    const qan = '/graph/d/pmm-qan/pmm-query-analytics';

    expect(findActiveNavItem(tree, qan)?.id).toBe('qan');
    expect(
      findActiveNavItem(tree, qan, '?var-service_type=postgresql')?.id
    ).toBe('pinned-postgresql-query-analytics');
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

  it('lists the projected app catalog, with MySQL backups opening its route', () => {
    const apps = findById(build(), 'apps')?.children;

    expect(apps?.map((app) => app.text)).toEqual([
      'Data archiving',
      'MySQL backups',
      'PostgreSQL backups',
      'ProxySQL manager',
      'Replication checksums',
      'Schema changes',
      'Valkey backups',
      'Get more apps',
    ]);
    expect(findById(apps, 'extensions-mysql-backups')?.url).toBe(
      EXTENSIONS_MYSQL_BACKUPS_PATH
    );
    expect(findById(apps, 'app-schema-changes')?.url).toBe(
      `${PROTOTYPE_PLACEHOLDER_PATH}/schema-changes`
    );
  });

  it('offers a quick add on the inventory rows', () => {
    const rows = findById(build(), 'inventory')?.children;

    expect(rows?.map((row) => row.action?.label)).toEqual([
      'Add service',
      'Add node',
    ]);
  });

  it('keeps a deep link into an app active and inside its group', () => {
    const tree = build();
    const backups = findActiveNavItem(
      tree,
      `${EXTENSIONS_MYSQL_BACKUPS_PATH}/backups/123`
    );

    expect(backups?.id).toBe('extensions-mysql-backups');
    // The sidebar expands a group when its active child is the very object
    // held in `children`, so identity has to match, not just the id.
    expect(findById(tree, 'apps')?.children).toContain(backups);
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
