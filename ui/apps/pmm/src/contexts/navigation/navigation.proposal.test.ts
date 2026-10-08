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
      'explore',
      'qan',
      'section-browse',
      'dashboards',
      'apps',
      'inventory',
      'section-administration',
      'high-availability',
      'configuration',
      'users-and-access',
      'account',
      'help',
    ]);
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

  it('lists the projected app catalog, with the real apps opening their routes', () => {
    const apps = findById(build(), 'apps')?.children;

    expect(apps?.map((app) => app.text)).toEqual([
      'Data archiving',
      'MongoDB backups',
      'MySQL backups',
      'PostgreSQL backups',
      'ProxySQL manager',
      'Replication checksums',
      'Schema changes',
      'Support diagnostics',
      'Valkey backups',
      'Get more apps',
    ]);
    expect(findById(apps, 'extensions-mysql-backups')?.url).toBe(
      EXTENSIONS_MYSQL_BACKUPS_PATH
    );
    expect(findById(apps, 'extensions-atw')?.url).toBe(EXTENSIONS_ATW_PATH);
    expect(findById(apps, 'app-schema-changes')?.url).toBe(
      `${PROTOTYPE_PLACEHOLDER_PATH}/schema-changes`
    );
  });

  it('keeps a deep link into an app active and inside Apps', () => {
    const tree = build();
    const active = findActiveNavItem(tree, `${EXTENSIONS_ATW_PATH}/runs/abc`);

    expect(active?.id).toBe('extensions-atw');
    // The sidebar expands a group when its active child is the very object
    // held in `children`, so identity has to match, not just the id.
    expect(findById(tree, 'apps')?.children).toContain(active);
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
    const tree = ids(build({ user: TEST_USER_ANONYMOUS, isLoggedIn: false }));

    expect(tree).toContain('sign-in');
    expect(tree).not.toContain('account');
  });
});
