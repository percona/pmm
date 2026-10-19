import type {
  HubAttentionItem,
  HubDoor,
  HubSectionState,
  HubTreeColumn,
  HubTreeNode,
  HubViewModel,
} from './hub.types';

// View-model pieces for the template's component tests; the MongoDB derivation has its own fixtures.

export const READY: HubSectionState = { status: 'ready' };

export const minutesAgo = (minutes: number) =>
  new Date(Date.now() - minutes * 60_000).toISOString();

export const testDoor = (
  label: string,
  to: string,
  extra: Partial<HubDoor> = {}
): HubDoor => ({ label, to, ...extra });

export const TEST_TREE_COLUMNS: HubTreeColumn[] = [
  {
    id: 'lag',
    header: 'Replication lag',
    description: 'How far each secondary is behind its primary.',
  },
  {
    id: 'oplogWindow',
    header: 'Oplog window',
    description: 'How much history the primary keeps for its secondaries.',
  },
];

const summary = (name: string) =>
  testDoor(name, `/graph/d/summary?var-name=${name}`);

const member = (
  name: string,
  overrides: Partial<HubTreeNode> = {}
): HubTreeNode => ({
  id: `member:svc-${name}`,
  kind: 'member',
  name,
  type: 'Secondary',
  health: 'healthy',
  showHealth: false,
  measures: {},
  doors: { name: summary(name), menu: [] },
  ...overrides,
});

export const TEST_CLUSTERS: HubTreeNode[] = [
  {
    id: 'cluster:prod/billing',
    kind: 'cluster',
    name: 'billing',
    type: 'Replica set',
    environment: 'prod',
    health: 'down',
    showHealth: true,
    since: minutesAgo(18),
    members: { up: 1, total: 3, noData: 0 },
    measures: {
      lag: { kind: 'notApplicable', reason: 'No secondary is up' },
    },
    doors: {
      name: summary('billing'),
      members: testDoor('1/3', '/graph/d/replset?var-rs=billing', {
        context: 'members of billing',
      }),
      menu: [
        testDoor('Open summary', '/graph/d/replset?var-rs=billing'),
        testDoor('Manage in Operations', '/operations?cluster=om-billing', {
          preview: 'developer',
        }),
      ],
    },
    children: [
      member('billing-m1', {
        type: 'Member',
        health: 'down',
        showHealth: true,
        since: minutesAgo(18),
      }),
      member('billing-m3'),
    ],
  },
  {
    id: 'cluster:prod/orders',
    kind: 'cluster',
    name: 'orders',
    type: 'Sharded cluster',
    environment: 'prod',
    health: 'degraded',
    showHealth: true,
    members: { up: 5, total: 5, noData: 0 },
    measures: {
      lag: {
        kind: 'value',
        seconds: 40,
        level: 'warning',
        rule: 'Warning above 30s sustained for 2m',
        door: testDoor('40s', '/graph/d/replset?var-rs=orders-shard1', {
          context: 'for orders-shard1',
        }),
      },
    },
    doors: { name: summary('orders'), menu: [] },
    children: [
      {
        id: 'group:prod/orders/routers',
        kind: 'group',
        name: 'Routers (mongos)',
        type: 'Routers',
        health: 'healthy',
        showHealth: true,
        members: { up: 2, total: 2, noData: 0 },
        measures: {},
        doors: { name: summary('routers'), menu: [] },
        children: [member('orders-mongos', { type: 'Router' })],
      },
      {
        id: 'set:prod/orders/orders-shard1',
        kind: 'set',
        name: 'orders-shard1',
        type: 'Shard',
        health: 'degraded',
        showHealth: true,
        members: { up: 3, total: 3, noData: 0 },
        measures: {},
        doors: { name: summary('orders-shard1'), menu: [] },
        children: [
          member('orders-shard1-m1', { type: 'Primary' }),
          member('orders-shard1-m3', {
            measures: {
              lag: { kind: 'value', seconds: 40, level: 'warning' },
            },
          }),
        ],
      },
    ],
  },
  {
    id: 'cluster:staging/reports',
    kind: 'cluster',
    name: 'reports',
    type: 'Standalone',
    environment: 'staging',
    health: 'noData',
    showHealth: true,
    since: minutesAgo(9),
    members: { up: 0, total: 1, noData: 1 },
    measures: {},
    doors: { name: summary('reports'), menu: [] },
  },
];

export const TEST_EXPANDED: Record<string, true> = {
  'cluster:prod/billing': true,
  'cluster:prod/orders': true,
  'set:prod/orders/orders-shard1': true,
};

export const attentionItem = (
  id: string,
  overrides: Partial<HubAttentionItem> = {}
): HubAttentionItem => ({
  id,
  kind: 'memberDown',
  severity: 'warning',
  title: `${id} needs a look`,
  where: 'orders · prod',
  door: testDoor('Open Instance summary', `/graph/d/instance?var-name=${id}`, {
    context: `for ${id}`,
  }),
  ...overrides,
});

export const testView = (
  overrides: Partial<HubViewModel> = {}
): HubViewModel => ({
  technology: { id: 'mongodb', name: 'MongoDB' },
  scenario: 'wrong',
  role: 'admin',
  capabilities: {
    addService: true,
    install: true,
    operations: true,
    setUpBackup: true,
    extensions: true,
  },
  environment: {
    selected: null,
    options: [
      { value: 'prod', label: 'prod' },
      { value: 'staging', label: 'staging' },
    ],
    matchesNothing: false,
  },
  status: {
    state: 'down',
    clusters: 3,
    needAttention: 3,
    lastGoodBackup: { at: minutesAgo(360), cluster: 'billing' },
    checkedAt: minutesAgo(0),
    doors: {
      clusters: testDoor('3 clusters', '/mongodb#clusters'),
      attention: testDoor('3 need attention', '/mongodb#needs-attention'),
      backup: testDoor('last good backup', '/graph/backup/inventory'),
    },
  },
  attention: [],
  clusters: TEST_CLUSTERS,
  expanded: TEST_EXPANDED,
  backups: {
    state: 'ok',
    rows: [],
    doors: {
      app: testDoor('MongoDB backups', '/graph/backup/inventory'),
      dashboard: testDoor(
        'Backup status dashboard',
        '/graph/d/mongodb-backup-details'
      ),
    },
  },
  queries: {
    links: {
      analytics: testDoor('Query analytics', '/graph/d/pmm-qan'),
      realtime: testDoor('Real-time queries', '/rta'),
    },
    slower: [],
    runningLong: [],
  },
  dashboards: {
    recommended: [],
    forDevelopers: [],
    recent: [],
    all: testDoor('All MongoDB dashboards', '/graph/dashboards?query=MongoDB'),
  },
  apps: [],
  actions: {
    addService: testDoor('Add MongoDB service', '/graph/add-instance/mongodb'),
    install: testDoor(
      'Install MongoDB',
      '/operations/nodes?filter=unmonitored',
      {
        preview: 'developer',
      }
    ),
  },
  sectionDoors: {
    attention: testDoor('Alerts', '/alerting/status'),
    clusters: testDoor('Fleet in Operations', '/operations?tab=clusters', {
      preview: 'developer',
    }),
  },
  sections: {
    attention: READY,
    clusters: READY,
    backups: READY,
    queries: READY,
    dashboards: READY,
    apps: READY,
  },
  ...overrides,
});
