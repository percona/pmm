import AddLinkOutlined from '@mui/icons-material/AddLinkOutlined';
import ArrowOutwardOutlined from '@mui/icons-material/ArrowOutwardOutlined';
import DownloadOutlined from '@mui/icons-material/DownloadOutlined';
import { Graph4Icon } from '@percona/peak-ui';
import type {
  HubDoor,
  HubNeedsAttentionItem,
  HubSectionState,
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

/** A Needs attention row: a warning on one cluster, with a dashboard to open. */
export const attentionItem = (
  title: string,
  overrides: Partial<HubNeedsAttentionItem> = {}
): HubNeedsAttentionItem => ({
  severity: 'warning',
  title,
  impact: 'Reads from it may return stale data.',
  location: 'orders',
  environment: 'prod',
  since: '18m',
  issueCount: 1,
  alertFiring: false,
  source: 'PMM health monitoring',
  destination: {
    label: 'ReplSet summary',
    route: '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary',
    params: { 'var-cluster': 'orders' },
    kind: 'page',
  },
  ...overrides,
});

export const testView = (
  overrides: Partial<HubViewModel> = {}
): HubViewModel => ({
  technology: { id: 'mongodb', name: 'MongoDB' },
  scenario: 'unhealthy',
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
  shortcuts: [
    { label: 'Clusters', route: '/mongodb/clusters', kind: 'page' },
    {
      label: 'Instances overview',
      route: '/graph/d/mongodb-instance-overview/mongodb-instances-overview',
      kind: 'page',
    },
    {
      label: 'Query analytics',
      route: '/graph/d/pmm-qan/pmm-query-analytics',
      params: { 'var-service_type': 'mongodb' },
      kind: 'page',
    },
    { label: 'Real-time queries', route: '/rta', kind: 'page' },
    {
      label: 'MongoDB backups',
      route: '/graph/backup/inventory',
      kind: 'page',
    },
  ],
  status: {
    state: 'down',
    clusters: 3,
    health: { healthy: 1, degraded: 1, down: 1, upgrading: 0, noData: 0 },
    alerts: 2,
    alertRules: 'some',
    lastGoodBackup: { at: minutesAgo(360), cluster: 'billing' },
    backupsFailed: 0,
    backupCoverage: { protected: 3, protectable: 3 },
    checkedAt: minutesAgo(0),
    doors: {
      alerts: testDoor('Open alerts', '/alerting/status'),
      backups: testDoor('Open backups', '/graph/backup/inventory'),
      setUpAlerts: testDoor(
        'Set up alerts',
        '/graph/alerting/alert-rule-templates'
      ),
      setUpBackups: testDoor('Set up backups', '/graph/backup/new'),
    },
  },
  attention: [],
  recommendations: [],
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
    slower: [],
  },
  dashboards: {
    recommended: [],
    forDevelopers: [],
    recent: [],
    all: testDoor('All MongoDB dashboards', '/graph/dashboards?query=MongoDB'),
  },
  actions: {
    add: [
      {
        ...testDoor(
          'Monitor an existing MongoDB service',
          '/graph/add-instance/mongodb'
        ),
        icon: AddLinkOutlined,
      },
      {
        ...testDoor(
          'Install a new MongoDB cluster',
          '/operations/nodes?filter=unmonitored'
        ),
        icon: DownloadOutlined,
      },
    ],
    more: [
      {
        ...testDoor('Manage services', '/graph/inventory/services'),
        icon: Graph4Icon,
      },
      {
        ...testDoor('Documentation', 'https://docs.percona.com/mongodb.html'),
        icon: ArrowOutwardOutlined,
        external: true,
      },
    ],
  },
  sectionDoors: {
    apps: testDoor('All apps', '/prototype/all-apps'),
  },
  sections: {
    attention: READY,
    recommendations: READY,
    backups: READY,
    queries: READY,
    dashboards: READY,
  },
  ...overrides,
});
