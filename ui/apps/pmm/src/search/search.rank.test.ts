import {
  buildSearchGroups,
  buildZeroQueryGroups,
  mergeIndex,
  prepareItems,
  prepareQuery,
  scoreItem,
} from './search.rank';
import { ROLE_USERS, staticIndexFor, testItem } from './search.testFixtures';
import type { RemoteState, SearchItem } from './search.types';
import { mapDashboardHits } from './sources/dashboards.source';

const baseOf = (item: SearchItem, query: string) =>
  scoreItem(prepareItems([item])[0], prepareQuery(query)).base;

const labels = (items: { label: string }[]) => items.map((item) => item.label);

const groupsFor = (
  items: SearchItem[],
  query: string,
  context?: Parameters<typeof buildSearchGroups>[2]
) => buildSearchGroups(prepareItems(items), query, context);

const errorState: RemoteState = { status: 'error', refetch: vi.fn() };

describe('scoreItem tiers', () => {
  it.each([
    ['an exact title', testItem('Telemetry'), 'telemetry', 1000],
    [
      'an exact keyword phrase',
      testItem('Query analytics', { keywords: ['qan'] }),
      'qan',
      1000,
    ],
    ['a title prefix', testItem('MySQL summary'), 'mysql sum', 900],
    ['word prefixes', testItem('MySQL InnoDB details'), 'inno', 800],
    ['word prefixes in any order', testItem('Query analytics'), 'ana qu', 800],
    ['the initials', testItem('Real-time queries'), 'rtq', 750],
    [
      'keyword word prefixes',
      testItem('Contact points', { keywords: ['slack'] }),
      'sla',
      700,
    ],
    [
      'an alias, which matches as a word only',
      testItem('MySQL summary', { aliases: ['percona server', 'galera'] }),
      'galera',
      700,
    ],
    ['a title substring', testItem('Disk details'), 'etai', 500],
    [
      'substrings across title, keywords and description',
      testItem('Silences', { description: ['Analysis and alerting'] }),
      'lenc ert',
      400,
    ],
    ['a tight subsequence', testItem('Organizations'), 'orgzt', 196],
    ['nothing', testItem('Organizations'), 'xyz', 0],
  ])('scores %s', (_, item, query, base) => {
    expect(baseOf(item, query)).toBe(base);
  });

  it('needs every term to match', () => {
    expect(baseOf(testItem('MySQL summary'), 'mysql xyz')).toBe(0);
  });

  it('counts only prefix tiers for one-character terms', () => {
    expect(baseOf(testItem('Data retention'), 'd')).toBe(900);
    expect(baseOf(testItem('Data retention'), 'a')).toBe(0);
    expect(baseOf(testItem('Disk a b'), 'a b')).toBe(800);
    expect(baseOf(testItem('Xa bc'), 'a b')).toBe(0);
  });

  it('keeps boosts within a tier', () => {
    const boosted = testItem('Contact points', {
      keywords: ['mysql server'],
      starred: true,
    });
    const prepared = prepareItems([boosted])[0];
    const score = scoreItem(prepared, prepareQuery('mysql'), 0);

    expect(score.base).toBe(700);
    expect(score.total).toBe(700 + 25 + 10 + 6);
    expect(score.total).toBeLessThan(750);
  });
});

describe('buildSearchGroups', () => {
  it('ranks the static index as expected', () => {
    const index = staticIndexFor();
    const first = (query: string) => groupsFor(index, query)[0].items[0].label;
    const groupOf = (query: string, id: string) =>
      groupsFor(index, query).find((group) => group.id === id);

    expect(first('qan')).toBe('Query analytics');
    expect(first('rtq')).toBe('Real-time queries');
    expect(groupOf('ana', 'pages')?.items[0].label).toBe('Query analytics');
    expect(groupOf('mysql sum', 'dashboards')?.items[0].label).toBe(
      'MySQL summary'
    );
    expect(groupOf('inno', 'dashboards')?.items[0].label).toBe(
      'MySQL InnoDB details'
    );
    expect(labels(groupOf('data ret', 'settings')?.items ?? [])).toEqual([
      'Data retention',
    ]);
  });

  it('ranks a sub-technology’s own dashboards above those its alias only names', () => {
    const index = staticIndexFor();
    const dashboardsFor = (query: string) =>
      labels(
        groupsFor(index, query).find((group) => group.id === 'dashboards')
          ?.items ?? []
      );

    ['galera', 'pxc'].forEach((query) => {
      expect(labels(groupsFor(index, query)[0].items)).toEqual(['MySQL']);
      expect(dashboardsFor(query).slice(0, 3)).toEqual([
        'MySQL PXC/Galera node',
        'MySQL PXC/Galera nodes',
        'MySQL PXC/Galera cluster',
      ]);
    });
    expect(dashboardsFor('mariadb')[0]).toBe('MySQL summary');
  });

  it('ranks a service named for an alias above those the alias only describes', () => {
    const service = (name: string) =>
      testItem(name, {
        id: `service:${name}`,
        kind: 'service',
        group: 'inventory',
        aliases: ['pxc', 'galera'],
      });

    expect(
      labels(
        groupsFor([service('mysql-1'), service('galera-1')], 'galera')[0].items
      )
    ).toEqual(['galera-1', 'mysql-1']);
  });

  it('drops fuzzy matches once anything matches well, reading base scores', () => {
    const groups = groupsFor(
      [testItem('Organizations', { starred: true }), testItem('Orgzt')],
      'orgzt',
      { recentIds: ['page:Organizations'] }
    );

    expect(labels(groups[0].items)).toEqual(['Orgzt']);
  });

  it('keeps fuzzy matches when nothing matches well', () => {
    expect(
      labels(groupsFor([testItem('Organizations')], 'orgzt')[0].items)
    ).toEqual(['Organizations']);
  });

  it('breaks ties by the shorter label, then by order', () => {
    const groups = groupsFor(
      [
        testItem('Alert rules long', { order: 0 }),
        testItem('Alert b', { order: 2 }),
        testItem('Alert a', { order: 1 }),
      ],
      'alert'
    );

    expect(labels(groups[0].items)).toEqual([
      'Alert a',
      'Alert b',
      'Alert rules long',
    ]);
  });

  it('caps each group and adds Show all past six dashboards', () => {
    const pages = Array.from({ length: 9 }, (_, i) => testItem(`Node ${i}`));
    const dashboards = Array.from({ length: 8 }, (_, i) =>
      testItem(`Node dashboard ${i}`, {
        id: `dashboard:${i}`,
        kind: 'dashboard',
        group: 'dashboards',
        dedupeKey: `dashboard:${i}`,
      })
    );
    const groups = groupsFor([...pages, ...dashboards], ' node ');
    const dashboardGroup = groups.find((group) => group.id === 'dashboards');

    expect(groups[0].items).toHaveLength(6);
    expect(dashboardGroup?.items).toHaveLength(7);
    expect(dashboardGroup?.items[6]).toMatchObject({
      id: 'link:all-dashboards',
      label: 'Show all 8 dashboards',
      to: '/graph/dashboards?query=node',
    });
  });

  it('adds retry rows after the cap for failed remote sources', () => {
    const dashboards = Array.from({ length: 7 }, (_, i) =>
      testItem(`Disk ${i}`, {
        id: `dashboard:${i}`,
        kind: 'dashboard',
        group: 'dashboards',
        dedupeKey: `dashboard:${i}`,
      })
    );
    const groups = groupsFor(dashboards, 'disk', {
      dashboards: errorState,
      inventory: errorState,
    });
    const [dashboardGroup, inventoryGroup] = groups;

    expect(dashboardGroup.items.map((item) => item.id).slice(6)).toEqual([
      'link:all-dashboards',
      'retry:dashboards',
    ]);
    expect(dashboardGroup.items[7]).toMatchObject({
      status: 'error',
      label: "Couldn't load dashboards",
      description: 'Select to try again',
      perform: errorState.refetch,
    });
    expect(inventoryGroup).toMatchObject({
      id: 'inventory',
      items: [{ id: 'retry:inventory', label: "Couldn't load inventory" }],
    });
  });

  it('shows remote items and retry rows only from two characters', () => {
    const items = [
      testItem('MySQL summary', {
        id: 'dashboard:local',
        kind: 'dashboard',
        group: 'dashboards',
        curated: true,
      }),
      testItem('MySQL remote', {
        id: 'dashboard:remote',
        kind: 'dashboard',
        group: 'dashboards',
        remote: true,
      }),
      testItem('mysql-1', {
        id: 'service:1',
        kind: 'service',
        group: 'inventory',
        remote: true,
      }),
    ];

    expect(
      groupsFor(items, 'm', { dashboards: errorState }).map((group) => [
        group.id,
        labels(group.items),
      ])
    ).toEqual([['dashboards', ['MySQL summary']]]);
    expect(
      groupsFor(items, 'my').map((group) => [group.id, labels(group.items)])
    ).toEqual([
      ['dashboards', ['MySQL summary', 'MySQL remote']],
      ['inventory', ['mysql-1']],
    ]);
  });

  it('moves an exact Settings match first when Pages has none', () => {
    expect(
      groupsFor(staticIndexFor(), 'telemetry').map((group) => group.id)
    ).toEqual(['settings']);
    expect(
      groupsFor(staticIndexFor(), 'data retention').map((group) => group.id)[0]
    ).toBe('settings');
  });

  it('moves a title prefix first over weaker matches in earlier groups', () => {
    const groups = groupsFor(staticIndexFor(), 'add');

    expect(groups.map((group) => group.id)).toEqual(['actions', 'settings']);
    expect(groups[0].items[0].label).toBe('Add a service');
  });

  it('moves promoted groups first by their best match', () => {
    const groups = groupsFor(
      [
        testItem('Telemetry settings', {
          id: 'setting:telemetry',
          kind: 'setting',
          group: 'settings',
        }),
        testItem('Telemetry', {
          id: 'action:telemetry',
          kind: 'action',
          group: 'actions',
        }),
        testItem('Telemetry dashboard', {
          id: 'dashboard:telemetry',
          kind: 'dashboard',
          group: 'dashboards',
        }),
      ],
      'telemetry'
    );

    expect(groups.map((group) => group.id)).toEqual([
      'actions',
      'settings',
      'dashboards',
    ]);
  });

  it('keeps the fixed order when a title prefix only ties with Pages', () => {
    const groups = groupsFor(
      [
        testItem('Alert me', {
          id: 'action:alert',
          kind: 'action',
          group: 'actions',
        }),
        testItem('Alert rules'),
      ],
      'alert'
    );

    expect(groups.map((group) => group.id)).toEqual(['pages', 'actions']);
  });

  it('keeps Pages first when it holds the exact match', () => {
    const groups = groupsFor(staticIndexFor(), 'help');

    expect(groups[0].id).toBe('pages');
    expect(groups[0].items[0].label).toBe('Help');
    expect(groups.map((group) => group.id)).toContain('help');
  });

  it('returns nothing for a blank query', () => {
    expect(groupsFor(staticIndexFor(), '  ')).toEqual([]);
  });
});

describe('mergeIndex', () => {
  it('keeps the first item per destination and folds the rest into it', () => {
    const page = testItem('Query analytics', {
      id: 'page:qan',
      dedupeKey: 'dashboard:pmm-qan',
    });
    const dashboard = testItem('PMM Query Analytics', {
      id: 'dashboard:pmm-qan',
      kind: 'dashboard',
      group: 'dashboards',
      dedupeKey: 'dashboard:pmm-qan',
      keywords: ['qan'],
      starred: true,
      remote: true,
    });
    const merged = mergeIndex([[page], [dashboard]]);

    expect(merged).toHaveLength(1);
    expect(merged[0]).toMatchObject({
      id: 'page:qan',
      starred: true,
      keywords: ['PMM Query Analytics', 'qan'],
    });
    expect(merged[0].remote).toBeUndefined();
    expect(page.keywords).toEqual([]);
  });

  it('keeps a dropped item’s aliases as aliases', () => {
    const page = testItem('MySQL', { dedupeKey: 'dashboard:mysql' });
    const dashboard = testItem('MySQL overview', {
      id: 'dashboard:mysql',
      kind: 'dashboard',
      group: 'dashboards',
      dedupeKey: 'dashboard:mysql',
      aliases: ['pxc'],
    });
    const [merged] = mergeIndex([[page], [dashboard]]);

    expect(merged).toMatchObject({
      keywords: ['MySQL overview'],
      aliases: ['pxc'],
    });
    expect(page.aliases).toBeUndefined();
  });

  it('folds Grafana’s home dashboard into Overview, which redirects to it', () => {
    const merged = mergeIndex([
      staticIndexFor(),
      mapDashboardHits([
        {
          uid: 'pmm-home',
          title: 'Home Dashboard',
          url: '/graph/d/pmm-home/home-dashboard',
          type: 'dash-db',
        },
      ]),
    ]);
    const overview = merged.find((item) => item.id === 'page:home-page');

    expect(merged.some((item) => item.id === 'dashboard:pmm-home')).toBe(false);
    expect(overview?.keywords).toContain('Home Dashboard');
    expect(overview?.to).toBe('/graph/');
  });

  it('numbers items in merged order', () => {
    const merged = mergeIndex([
      [testItem('A'), testItem('B')],
      [testItem('C')],
    ]);

    expect(merged.map((item) => item.order)).toEqual([0, 1, 2]);
  });

  it('keeps every settings field entry next to its tab', () => {
    const settings = staticIndexFor().filter(
      (item) => item.group === 'settings'
    );

    expect(labels(settings)).toEqual([
      'Metrics resolution',
      'Advanced settings',
      'SSH key',
      'ServiceNow connection',
      'Data retention',
      'Public address',
      'Telemetry',
      'Check for updates',
      'Advisor run intervals',
      'Microsoft Azure monitoring',
      'Access control',
      'Feature management',
    ]);
  });
});

describe('buildZeroQueryGroups', () => {
  const summary = (groups: ReturnType<typeof buildZeroQueryGroups>) =>
    groups.map((group) => [group.id, labels(group.items)]);

  it.each([
    ['admin', ['Add a service', 'Create an alert rule', 'Switch to dark mode']],
    [
      'editor',
      ['Create an alert rule', 'Switch to dark mode', 'Start the product tour'],
    ],
    ['viewer', ['Switch to dark mode', 'Start the product tour']],
  ] as const)('starts an %s with pages, actions and help', (role, actions) => {
    const index = staticIndexFor(ROLE_USERS[role]);

    expect(summary(buildZeroQueryGroups(index, []))).toEqual([
      ['pages', ['Overview', 'Query analytics', 'All dashboards', 'Advisors']],
      ['actions', [...actions]],
      ['help', ['Help', 'PMM documentation']],
    ]);
  });

  it('shows Recent instead of the starters, deduped from the rest', () => {
    const index = staticIndexFor();
    const help = index.find((item) => item.id === 'page:help');
    const recent = [testItem('Disk details'), help as SearchItem];

    expect(summary(buildZeroQueryGroups(index, recent))).toEqual([
      ['recent', ['Disk details', 'Help']],
      [
        'actions',
        ['Add a service', 'Create an alert rule', 'Switch to dark mode'],
      ],
      ['help', ['PMM documentation']],
    ]);
  });
});
