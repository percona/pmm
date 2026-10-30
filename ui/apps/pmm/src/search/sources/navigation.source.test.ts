import { NAV_MYSQL } from 'contexts/navigation/navigation.constants';
import { buildProposedNavTree } from 'contexts/navigation/navigation.proposal';
import type { NavItem } from 'types/navigation.types';
import type { User } from 'types/user.types';
import { UpdateStatus } from 'types/updates.types';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { navTreeFor } from '../search.testFixtures';
import { flattenNavTree, technologyDashboards } from './navigation.source';

const Icon = () => null;
const OtherIcon = () => null;

const TREE: NavItem[] = [
  { id: 'home', text: 'Home', url: '/graph/', icon: Icon },
  { id: 'section-a', type: 'menu-section', text: 'Section A' },
  {
    id: 'group',
    text: 'Group',
    icon: OtherIcon,
    url: '/group',
    children: [
      { id: 'group-main', text: 'Main', url: '/group/' },
      { id: 'divider', type: 'menu-divider' },
      { id: 'label', type: 'menu-text', text: 'Label' },
      { id: 'sub-section', type: 'menu-section', text: 'Sub' },
      { id: 'group-child', text: 'Child', url: '/group/child', icon: Icon },
      { id: 'hidden', text: 'Hidden', url: '/hidden', hidden: true },
    ],
  },
  {
    id: 'folder',
    text: 'Folder',
    children: [{ id: 'folder-child', text: 'Folder child', url: '/folder' }],
  },
];

describe('flattenNavTree', () => {
  it('lists every page with its section and ancestors as the breadcrumb', () => {
    const { pages } = flattenNavTree(TREE, { isAdmin: true });

    expect(
      pages.map(({ id, label, description }) => [id, label, description])
    ).toEqual([
      ['page:home', 'Home', undefined],
      ['page:group-main', 'Main', ['Section A', 'Group']],
      ['page:group-child', 'Child', ['Section A', 'Group', 'Sub']],
      ['page:folder-child', 'Folder child', ['Section A', 'Folder']],
    ]);
  });

  it('gives a group’s place to the child that opens the same page', () => {
    const { pages } = flattenNavTree(TREE, { isAdmin: true });
    const main = pages.find((page) => page.id === 'page:group-main');

    expect(pages.find((page) => page.id === 'page:group')).toBeUndefined();
    expect(main?.keywords).toEqual(['Group']);
  });

  it('takes the top ancestor’s icon when an entry has none', () => {
    const { pages } = flattenNavTree(TREE, { isAdmin: true });

    expect(pages.find((page) => page.id === 'page:group-main')?.navIcon).toBe(
      OtherIcon
    );
    expect(pages.find((page) => page.id === 'page:group-child')?.navIcon).toBe(
      Icon
    );
  });

  it('keys Overview by the home dashboard that Grafana redirects its root to', () => {
    const overviewKey = (user: User) =>
      flattenNavTree(navTreeFor(user), { isAdmin: true }).pages.find(
        (page) => page.id === 'page:home-page'
      )?.dedupeKey;

    expect(overviewKey(TEST_USER_ADMIN)).toBe('dashboard:pmm-home');
    expect(
      overviewKey({
        ...TEST_USER_ADMIN,
        preferences: { homeDashboardUID: 'custom' },
      })
    ).toBe('dashboard:custom');
    expect(flattenNavTree(TREE, { isAdmin: true }).pages[0].dedupeKey).toBe(
      '/graph'
    );
  });

  it('mirrors the proposed tree, with Account and its name cut', () => {
    const { pages } = flattenNavTree(navTreeFor(TEST_USER_ADMIN), {
      isAdmin: true,
    });
    const byId = (id: string) => pages.find((page) => page.id === id);

    expect(TEST_USER_ADMIN.name).toBeTruthy();
    expect(byId('page:account')).toBeUndefined();
    expect(byId('page:profile')).toMatchObject({
      label: 'Profile',
      description: ['Administration', 'Account'],
      keywords: ['Account'],
    });
    expect(byId('page:configuration')).toBeUndefined();
    expect(byId('page:configuration-settings')?.keywords).toEqual([
      'Configuration',
    ]);
    expect(byId('page:alerts-status')?.keywords).toEqual(['Alerts']);
    expect(byId('page:users')?.description).toEqual([
      'Administration',
      'Configuration',
      'Users and access',
    ]);
    expect(byId('page:inventory')).toMatchObject({ label: 'Inventory' });
    expect(byId('page:pinned-mongodb-backups')).toMatchObject({
      starred: true,
      description: ['Starred'],
    });
    expect(byId('page:qan')).toMatchObject({
      to: '/graph/d/pmm-qan/pmm-query-analytics',
      carryVariables: true,
      dedupeKey: 'dashboard:pmm-qan',
    });
    expect(pages.some((page) => page.label.includes(':'))).toBe(false);
  });

  it('derives actions from the tree, with entry actions for admins only', () => {
    const tree = navTreeFor(TEST_USER_ADMIN);
    const admin = flattenNavTree(tree, { isAdmin: true }).actions;
    const viewer = flattenNavTree(navTreeFor(TEST_USER_VIEWER), {
      isAdmin: false,
    }).actions;

    expect(admin.map((action) => action.label)).toEqual([
      'Add a service',
      'Add a node',
      'Switch to dark mode',
      'Sign out',
    ]);
    expect(viewer.map((action) => action.label)).toEqual([
      'Switch to dark mode',
      'Sign out',
    ]);
    expect(admin[0]).toMatchObject({
      id: 'action:inventory-services-action',
      to: '/graph/add-instance',
      dedupeKey: '/graph/add-instance',
    });
    expect(admin[2]).toMatchObject({ id: 'action:theme-toggle' });
    expect(typeof admin[2].perform).toBe('function');
    expect(admin[3]).toMatchObject({
      id: 'action:sign-out',
      href: '/graph/logout',
      target: '_self',
      keywords: ['log out', 'logout'],
    });
  });

  it('offers Sign in when nobody is signed in', () => {
    const tree = buildProposedNavTree({
      isLoggedIn: false,
      updateStatus: UpdateStatus.UpToDate,
      colorMode: 'light',
      toggleColorMode: () => {},
    });
    const { actions } = flattenNavTree(tree, { isAdmin: false });

    expect(actions).toEqual([
      expect.objectContaining({
        id: 'action:sign-in',
        href: '/graph/login',
        target: '_self',
      }),
    ]);
  });

  it('lists Starred dashboards under Dashboards', () => {
    const tree = buildProposedNavTree({
      user: TEST_USER_ADMIN,
      isLoggedIn: true,
      updateStatus: UpdateStatus.UpToDate,
      colorMode: 'light',
      toggleColorMode: () => {},
      starredDashboards: [
        {
          uid: 'node-disk',
          title: 'Disk details',
          url: '/graph/d/node-disk/disk-details',
        },
      ],
    });
    const { pages, starredDashboards } = flattenNavTree(tree, {
      isAdmin: true,
    });

    expect(starredDashboards).toEqual([
      expect.objectContaining({
        id: 'dashboard:node-disk',
        kind: 'dashboard',
        group: 'dashboards',
        label: 'Disk details',
        starred: true,
        carryVariables: true,
        description: ['Starred'],
      }),
    ]);
    expect(pages.some((page) => page.label === 'Disk details')).toBe(false);
  });
});

describe('technologyDashboards', () => {
  const dashboards = technologyDashboards();
  const byId = (id: string) => dashboards.find((item) => item.id === id);

  it('labels each child with its technology first', () => {
    expect(byId('dashboard:mysql-instance-summary')).toMatchObject({
      label: 'MySQL summary',
      description: ['Technologies', 'MySQL'],
      kind: 'dashboard',
      group: 'dashboards',
      curated: true,
      carryVariables: true,
      keywords: ['Summary'],
      aliases: ['mariadb', 'percona server', 'pxc', 'galera'],
    });
    expect(byId('dashboard:valkey-slowlog')?.label).toBe('Valkey slow log');
    expect(byId('dashboard:node-memory-numa')?.label).toBe(
      'Operating system NUMA'
    );
  });

  it('drops the overview child, which opens the technology page itself', () => {
    expect(byId('dashboard:mysql-instance-overview')).toBeUndefined();
    expect(byId('dashboard:node-instance-overview')).toBeUndefined();
  });

  it('gives a high availability group’s place to its first child', () => {
    const groupReplication = dashboards.filter(
      (item) => item.id === 'dashboard:mysql-group-replicaset-summary'
    );

    expect(groupReplication).toHaveLength(1);
    expect(groupReplication[0]).toMatchObject({
      label: 'MySQL group replication',
      description: ['Technologies', 'MySQL', 'High availability'],
    });
    expect(groupReplication[0].keywords).toContain('High availability');
    expect(byId('dashboard:postgresql-patroni-details')?.label).toBe(
      'PostgreSQL Patroni'
    );
  });

  it('uses the child’s icon, or the technology’s', () => {
    expect(byId('dashboard:mysql-innodb')?.navIcon).toBe(NAV_MYSQL.icon);
    expect(byId('dashboard:mysql-instance-summary')?.navIcon).toBe(
      NAV_MYSQL.children?.[1].icon
    );
  });
});
