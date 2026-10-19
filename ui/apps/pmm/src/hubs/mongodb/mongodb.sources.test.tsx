import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { getPrometheusAlertRules } from 'api/alerting';
import { getRunningSessions, searchQueries } from 'api/rta';
import { UserContext } from 'contexts/user';
import { listArtifacts, listScheduled } from 'hubs/api/backups';
import { impressionsKey, searchDashboards } from 'hubs/api/grafanaDashboards';
import type * as GrafanaDashboardsModule from 'hubs/api/grafanaDashboards';
import { listManagedServicesQuietly } from 'hubs/api/inventory';
import { getReport } from 'hubs/api/qan';
import { instantQuery } from 'hubs/api/victoriaMetrics';
import type * as VictoriaMetricsModule from 'hubs/api/victoriaMetrics';
import {
  HUB_NO_ENVIRONMENT,
  type HubLiveContext,
} from 'hubs/template/hub.types';
import { request } from '@pmm/plugins-om/src/api';
import { getRecentKey } from 'search/search.recent';
import type {
  PrometheusAlertItem,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import { RealtimeSessionStatus } from 'types/rta.types';
import { ServiceStatus } from 'types/services.types';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { wrapWithSettings } from 'utils/testUtils';
import {
  RECORDED_RUNS,
  RECORDED_SERVICES,
  RECORDED_TOPOLOGY,
  RECORDED_VECTORS,
} from './fixtures/recorded';
import { MONGO_QUERY_IDS, mongoQueries } from './mongodb.promql';
import {
  fetchMongoMetrics,
  fetchQan,
  qanLabels,
  readRecentDashboards,
  toAlertInputs,
  toBackupsInput,
  toOperationsInput,
  toQanRows,
  toRtaQueries,
  toServiceInputs,
  useMongoLiveSnapshot,
} from './mongodb.sources';
import { findRunningQueries, runningAge } from './mongodb.queries';

vi.mock('api/alerting', () => ({ getPrometheusAlertRules: vi.fn() }));
vi.mock('api/rta', () => ({
  getRunningSessions: vi.fn(),
  searchQueries: vi.fn(),
}));
vi.mock('hubs/api/inventory', () => ({ listManagedServicesQuietly: vi.fn() }));
vi.mock('hubs/api/backups', () => ({
  listArtifacts: vi.fn(),
  listScheduled: vi.fn(),
}));
vi.mock('hubs/api/qan', () => ({ getReport: vi.fn() }));
vi.mock('hubs/api/victoriaMetrics', async (importOriginal) => ({
  ...(await importOriginal<typeof VictoriaMetricsModule>()),
  instantQuery: vi.fn(),
}));
vi.mock('hubs/api/grafanaDashboards', async (importOriginal) => ({
  ...(await importOriginal<typeof GrafanaDashboardsModule>()),
  searchDashboards: vi.fn(),
}));
vi.mock('@pmm/plugins-om/src/api', () => ({ request: vi.fn() }));

const NOW = Date.parse('2026-10-19T12:00:00.000Z');

const vectorFor = (query: string) => {
  const queries = mongoQueries(null);
  const id = MONGO_QUERY_IDS.find((key) => queries[key] === query);

  return (id ? RECORDED_VECTORS[id] : []).map(({ metric, value }) => ({
    metric,
    value: Number(value[1]),
    timestamp: value[0],
  }));
};

describe('toServiceInputs', () => {
  it('maps the management services, leaving empty labels out', () => {
    const services = toServiceInputs(RECORDED_SERVICES);
    const mongos = services.find((s) => s.serviceName === 'orders-mongos');

    expect(services).toHaveLength(12);
    expect(services[0]).toEqual({
      serviceId: 'b594af62-61db-443f-bb0b-c48c25c74467',
      serviceName: 'billing-m1',
      nodeName: 'billing-m1',
      environment: 'prod',
      cluster: 'billing',
      replicationSet: 'billing',
      inventoryStatus: 'up',
    });
    expect(mongos).not.toHaveProperty('replicationSet');
  });

  it('reads down and unknown statuses', () => {
    const [down, unknown, missing] = toServiceInputs([
      { ...RECORDED_SERVICES[0], status: ServiceStatus.down },
      { ...RECORDED_SERVICES[0], status: ServiceStatus.unknown },
      { ...RECORDED_SERVICES[0], status: undefined },
    ]);

    expect(down.inventoryStatus).toBe('down');
    expect(unknown.inventoryStatus).toBe('unknown');
    expect(missing.inventoryStatus).toBe('unknown');
  });
});

describe('fetchMongoMetrics', () => {
  afterEach(() => vi.mocked(instantQuery).mockReset());

  it('runs every query and folds the recorded vectors into metrics', async () => {
    vi.mocked(instantQuery).mockImplementation(async (query) =>
      vectorFor(query)
    );

    const metrics = await fetchMongoMetrics(null);

    expect(instantQuery).toHaveBeenCalledTimes(MONGO_QUERY_IDS.length);
    expect(Object.keys(metrics.up)).toHaveLength(12);
    expect(metrics.shardsByCluster).toEqual({ orders: 2 });
  });

  it('fails as a whole when one query fails, so nothing reads as no data', async () => {
    vi.mocked(instantQuery).mockImplementation(async (query) => {
      if (query.includes('mongodb_members_self')) {
        throw new Error('bad gateway');
      }

      return [];
    });

    await expect(fetchMongoMetrics('prod')).rejects.toThrow('bad gateway');
  });
});

describe('toAlertInputs', () => {
  const services = toServiceInputs(RECORDED_SERVICES);
  const rules = (
    alerts: Array<{
      labels: Record<string, string>;
      state?: PrometheusAlertItem['state'];
      annotations?: Record<string, string>;
    }>,
    ruleState?: 'firing' | 'inactive'
  ): PrometheusAlertRulesResponse => ({
    data: {
      groups: [
        {
          name: 'group',
          rules: [
            {
              name: 'Rule',
              state: ruleState,
              alerts: alerts.map((alert) => ({
                annotations: {},
                activeAt: '2026-10-19T11:00:00Z',
                ...alert,
              })),
            },
          ],
        },
      ],
    },
  });

  it('keeps firing MongoDB alerts with their labels', () => {
    expect(
      toAlertInputs(
        rules([
          {
            state: 'Alerting',
            labels: {
              alertname: 'MongoDB down',
              template_name: 'pmm_mongodb_down',
              severity: 'critical',
              service_name: 'billing-m2',
              cluster: 'billing',
              environment: 'prod',
            },
            annotations: { summary: 'MongoDB down (billing-m2)' },
          },
        ]),
        services
      )
    ).toEqual([
      {
        name: 'MongoDB down',
        templateName: 'pmm_mongodb_down',
        severity: 'critical',
        serviceName: 'billing-m2',
        cluster: 'billing',
        environment: 'prod',
        activeAt: '2026-10-19T11:00:00Z',
        summary: 'MongoDB down (billing-m2)',
      },
    ]);
  });

  it('recognises MongoDB by service type, service name or cluster', () => {
    const result = toAlertInputs(
      rules([
        {
          state: 'Alerting',
          labels: { alertname: 'a', service_type: 'mongodb' },
        },
        {
          state: 'Alerting',
          labels: { alertname: 'b', service_name: 'reports' },
        },
        {
          state: 'Alerting',
          labels: { alertname: 'c', cluster: 'orders', rs_nm: 'orders-shard1' },
        },
        {
          state: 'Alerting',
          labels: { alertname: 'd', service_name: 'mysql-shop-1' },
        },
      ]),
      services
    );

    expect(result.map((alert) => alert.name)).toEqual(['a', 'b', 'c']);
    expect(result[2].replicaSet).toBe('orders-shard1');
  });

  it('recognises a set alert by the replica set alone, for services without a cluster label', () => {
    const unlabelled = services.map(({ cluster: _cluster, ...rest }) => rest);
    const result = toAlertInputs(
      rules([
        {
          state: 'Alerting',
          labels: {
            alertname: 'MongoDB replica set has no primary',
            template_name: 'pmm_mongodb_replset_no_primary',
            rs_nm: 'billing',
          },
        },
        {
          state: 'Alerting',
          labels: { alertname: 'other', rs_nm: 'unknown-set' },
        },
      ]),
      unlabelled
    );

    expect(result.map((alert) => [alert.name, alert.replicaSet])).toEqual([
      ['MongoDB replica set has no primary', 'billing'],
    ]);
  });

  it('drops alerts that are not firing', () => {
    expect(
      toAlertInputs(
        rules([
          {
            state: 'Normal',
            labels: { alertname: 'a', service_type: 'mongodb' },
          },
          {
            state: 'Pending',
            labels: { alertname: 'b', service_type: 'mongodb' },
          },
        ]),
        services
      )
    ).toEqual([]);
  });

  it('falls back to the rule’s state when the alert has none', () => {
    expect(
      toAlertInputs(
        rules(
          [{ labels: { alertname: 'a', service_type: 'mongodb' } }],
          'firing'
        ),
        services
      )
    ).toHaveLength(1);
  });
});

describe('toBackupsInput', () => {
  it('keeps MongoDB artifacts and schedules with their mode and status', () => {
    expect(
      toBackupsInput(
        true,
        [
          {
            artifactId: 'a1',
            name: 'nightly',
            vendor: 'mongodb',
            serviceId: 's1',
            serviceName: 'billing-m1',
            status: 'BACKUP_STATUS_ERROR',
            createdAt: '2026-10-19T02:00:00Z',
            mode: 'BACKUP_MODE_PITR',
          },
          {
            artifactId: 'a2',
            name: 'mysql',
            vendor: 'mysql',
            serviceId: 's2',
            serviceName: 'mysql-shop-1',
            status: 'BACKUP_STATUS_SUCCESS',
            createdAt: '2026-10-19T02:00:00Z',
            mode: 'BACKUP_MODE_SNAPSHOT',
          },
        ],
        [
          {
            scheduledBackupId: 'sb1',
            serviceId: 's1',
            serviceName: 'billing-m1',
            cronExpression: '0 3 * * *',
            name: 'daily',
            enabled: true,
            mode: 'BACKUP_MODE_SNAPSHOT',
            vendor: 'mongodb',
            nextRun: '2026-10-20T03:00:00Z',
          },
        ]
      )
    ).toEqual({
      enabled: true,
      artifacts: [
        {
          artifactId: 'a1',
          name: 'nightly',
          serviceId: 's1',
          serviceName: 'billing-m1',
          mode: 'pitr',
          status: 'error',
          createdAt: '2026-10-19T02:00:00Z',
        },
      ],
      scheduled: [
        {
          scheduledBackupId: 'sb1',
          name: 'daily',
          serviceId: 's1',
          serviceName: 'billing-m1',
          mode: 'snapshot',
          enabled: true,
          cronExpression: '0 3 * * *',
          nextRun: '2026-10-20T03:00:00Z',
        },
      ],
    });
  });

  it('reads empty artifact and schedule lists', () => {
    expect(toBackupsInput(true, [], [])).toEqual({
      enabled: true,
      artifacts: [],
      scheduled: [],
    });
  });
});

describe('QAN', () => {
  afterEach(() => vi.mocked(getReport).mockReset());

  it('drops the total row and rows without a number, and reads the average query time', () => {
    expect(
      toQanRows([
        { dimension: '', fingerprint: 'TOTAL', numQueries: 9000 },
        {
          dimension: 'Q1',
          fingerprint: 'db.orders.find({status: ?})',
          numQueries: 1200,
          metrics: { queryTime: { stats: { avg: 0.84 } } },
        },
        {
          dimension: 'Q2',
          numQueries: 10,
          metrics: { queryTime: { stats: { sum: 5 } } },
        },
        { dimension: 'Q3', numQueries: 1 },
        {
          dimension: 'Q4',
          numQueries: 3,
          metrics: {
            queryTime: { stats: { avg: 'NaN' as unknown as number } },
          },
        },
      ])
    ).toEqual([
      {
        queryId: 'Q1',
        fingerprint: 'db.orders.find({status: ?})',
        numQueries: 1200,
        avgQueryTime: 0.84,
      },
      { queryId: 'Q2', fingerprint: 'Q2', numQueries: 10, avgQueryTime: 0.5 },
    ]);
  });

  it('filters by service type and environment', () => {
    expect(qanLabels(null)).toEqual([
      { key: 'service_type', value: ['mongodb'] },
    ]);
    expect(qanLabels('prod')).toEqual([
      { key: 'service_type', value: ['mongodb'] },
      { key: 'environment', value: ['prod'] },
    ]);
    expect(qanLabels(HUB_NO_ENVIRONMENT)[1]).toEqual({
      key: 'environment',
      value: [''],
    });
  });

  it('compares the last hour with the 24 hours before it', async () => {
    vi.mocked(getReport).mockResolvedValue([]);

    await fetchQan('prod', NOW);

    expect(
      vi
        .mocked(getReport)
        .mock.calls.map(([options]) => [
          options.from.toISOString(),
          options.to.toISOString(),
        ])
    ).toEqual([
      ['2026-10-19T11:00:00.000Z', '2026-10-19T12:00:00.000Z'],
      ['2026-10-18T11:00:00.000Z', '2026-10-19T11:00:00.000Z'],
    ]);
  });
});

describe('toRtaQueries', () => {
  it('names the operation and namespace, and reads the duration in seconds', () => {
    expect(
      toRtaQueries([
        {
          serviceId: 's1',
          serviceName: 'orders-shard1-m1',
          queryId: 'r1',
          queryText: 'db.orders.aggregate([])',
          queryExecutionDurationMs: 840,
          queryCollectTime: '',
          clientAddress: '',
          queryRawJson: '',
          mongoDbPayload: {
            dbInstanceAddress: '',
            clientAppName: '',
            databaseName: 'shop',
            collection: 'orders',
            operationStartTime: '',
            planSummary: '',
            operation: 'aggregate',
            username: '',
          },
        },
      ])
    ).toEqual([
      {
        serviceId: 's1',
        serviceName: 'orders-shard1-m1',
        queryText: 'db.orders.aggregate([])',
        operation: 'aggregate',
        namespace: 'shop.orders',
        durationMs: 840000,
      },
    ]);
  });
});

describe('toOperationsInput', () => {
  it('maps the recorded topology and runs', () => {
    expect(
      toOperationsInput(RECORDED_TOPOLOGY, RECORDED_RUNS, [], null)
    ).toEqual({
      clusterIds: {
        'prod/billing': '34f43ecdb19802d9',
        'prod/orders': '39dc72734db365ae',
        'staging/reports': '40cbf687d4830aa8',
      },
      // This topology lists every service down.
      summary: { services: 12, up: 0, lastScanAt: '2026-10-19T15:42:14Z' },
      failedRuns: [
        {
          id: '4585235a-f86b-45a9-8038-2049190bfab3',
          kind: 'scan',
          target: '',
          finishedAt: '2026-10-19T15:42:14Z',
        },
      ],
    });
  });

  it('counts only the selected environment', () => {
    expect(
      toOperationsInput(RECORDED_TOPOLOGY, [], [], 'staging').summary
    ).toEqual({ services: 1, up: 0 });
  });

  it('reports a failed install and skips a scan that succeeded last', () => {
    const result = toOperationsInput(
      RECORDED_TOPOLOGY,
      RECORDED_RUNS.slice(3),
      [
        {
          run_id: 'b1',
          status: 'failed',
          hosts: [{ host: 'pmm-client' }],
          replica_set_name: 'rs-new',
          started_at: '2026-10-19T11:00:00Z',
          finished_at: '2026-10-19T11:40:00Z',
        },
        {
          run_id: 'b2',
          status: 'succeeded',
          hosts: [{ host: 'other' }],
          replica_set_name: 'rs-ok',
          started_at: '2026-10-19T10:00:00Z',
        },
      ],
      null
    );

    expect(result.failedRuns).toEqual([
      {
        id: 'b1',
        kind: 'install',
        target: 'pmm-client',
        finishedAt: '2026-10-19T11:40:00Z',
      },
    ]);
  });
});

describe('toOperationsInput installs', () => {
  const install = (run_id: string, environment?: string) => ({
    run_id,
    status: 'failed' as const,
    hosts: [{ host: run_id }],
    replica_set_name: 'rs-new',
    started_at: '2026-10-19T11:00:00Z',
    finished_at: '2026-10-19T11:40:00Z',
    ...(environment ? { environment } : {}),
  });
  const runs = [install('prod-host', 'prod'), install('bare-host')];
  const targets = (environment: string | null) =>
    toOperationsInput(RECORDED_TOPOLOGY, [], runs, environment).failedRuns.map(
      (run) => run.target
    );

  it('keeps only the failed installs of the selected environment', () => {
    expect(targets(null)).toEqual(['prod-host', 'bare-host']);
    expect(targets('prod')).toEqual(['prod-host']);
    expect(targets('staging')).toEqual([]);
    expect(targets(HUB_NO_ENVIRONMENT)).toEqual(['bare-host']);
  });
});

describe('readRecentDashboards', () => {
  afterEach(() => localStorage.clear());

  it('prefers Grafana’s recently viewed list', () => {
    localStorage.setItem(
      impressionsKey(1),
      JSON.stringify(['mongodb-oplog-details'])
    );

    expect(readRecentDashboards({ orgId: 1, id: 1 })).toEqual([
      { uid: 'mongodb-oplog-details' },
    ]);
  });

  it('falls back to the dashboards PMM search recorded, newest first', () => {
    localStorage.setItem(
      getRecentKey({ orgId: 1, id: 1 }),
      JSON.stringify({
        v: 1,
        items: [
          {
            id: 'dashboard:old',
            kind: 'dashboard',
            label: 'Old',
            to: '/graph/d/old',
            at: 1000,
          },
          {
            id: 'page:mongo',
            kind: 'page',
            label: 'MongoDB',
            to: '/mongodb',
            at: 3000,
          },
          {
            id: 'dashboard:new',
            kind: 'dashboard',
            label: 'New',
            to: '/graph/d/new',
            at: 2000,
          },
        ],
      })
    );

    expect(readRecentDashboards({ orgId: 1, id: 1 })).toEqual([
      { uid: 'new', visitedAt: new Date(2000).toISOString() },
      { uid: 'old', visitedAt: new Date(1000).toISOString() },
    ]);
  });
});

describe('useMongoLiveSnapshot', () => {
  const settings = {
    alertingEnabled: true,
    backupManagementEnabled: true,
  };

  const renderLive = (
    ctx: Partial<HubLiveContext> = {},
    settingsOverride: Record<string, boolean> = settings
  ) => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    return renderHook(
      () =>
        useMongoLiveSnapshot({
          enabled: true,
          environment: null,
          operations: false,
          role: 'admin',
          ...ctx,
        }),
      {
        wrapper: ({ children }: PropsWithChildren) => (
          <UserContext.Provider
            value={{ isLoading: false, user: TEST_USER_ADMIN }}
          >
            {wrapWithSettings(
              <QueryClientProvider client={client}>
                {children}
              </QueryClientProvider>,
              { settings: settingsOverride }
            )}
          </UserContext.Provider>
        ),
      }
    );
  };

  beforeEach(() => {
    vi.mocked(listManagedServicesQuietly).mockResolvedValue(RECORDED_SERVICES);
    vi.mocked(instantQuery).mockImplementation(async (query) =>
      vectorFor(query)
    );
    vi.mocked(getPrometheusAlertRules).mockResolvedValue({
      data: { groups: [] },
    });
    vi.mocked(listArtifacts).mockResolvedValue([]);
    vi.mocked(listScheduled).mockResolvedValue([]);
    vi.mocked(getReport).mockResolvedValue([]);
    vi.mocked(getRunningSessions).mockResolvedValue([]);
    vi.mocked(searchQueries).mockResolvedValue({ queries: [] });
    vi.mocked(searchDashboards).mockResolvedValue([]);
    vi.mocked(request).mockImplementation(async (path: string) =>
      path === '/topology' ? RECORDED_TOPOLOGY : { runs: [] }
    );
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('maps every source into the snapshot and makes no Operations request with it off', async () => {
    const { result } = renderLive();

    await waitFor(() =>
      expect(result.current.sources.metrics.status).toBe('ready')
    );
    await waitFor(() =>
      expect(result.current.sources.inventory.status).toBe('ready')
    );

    const { snapshot, sources } = result.current;

    expect(snapshot.services).toHaveLength(12);
    expect(Object.keys(snapshot.metrics?.up ?? {})).toHaveLength(12);
    expect(snapshot.operations).toBeNull();
    expect(sources.operations).toEqual({ status: 'disabled' });
    expect(request).not.toHaveBeenCalled();
  });

  it('reads Operations with it on', async () => {
    const { result } = renderLive({ operations: true });

    await waitFor(() =>
      expect(result.current.snapshot.operations).not.toBeNull()
    );

    expect(
      vi
        .mocked(request)
        .mock.calls.map(([path]) => path)
        .sort()
    ).toEqual([
      '/inventory/bootstrap-runs?limit=20',
      '/inventory/runs?limit=20',
      '/topology',
    ]);
    expect(result.current.snapshot.operations?.clusterIds).toHaveProperty(
      'prod/orders'
    );
  });

  it('fails only the metrics source when its queries fail', async () => {
    vi.mocked(instantQuery).mockRejectedValue(
      new Error('Request failed with 403')
    );

    const { result } = renderLive();

    await waitFor(() =>
      expect(result.current.sources.metrics.status).toBe('error')
    );
    await waitFor(() =>
      expect(result.current.sources.inventory.status).toBe('ready')
    );

    expect(result.current.sources.metrics.message).toBe(
      'Request failed with 403'
    );
    expect(result.current.snapshot.metrics).toBeNull();
    expect(result.current.snapshot.services).toHaveLength(12);
  });

  it('asks nothing for sample data', () => {
    const { result } = renderLive({ enabled: false });

    expect(
      Object.values(result.current.sources).every(
        (s) => s.status === 'disabled'
      )
    ).toBe(true);
    expect(listManagedServicesQuietly).not.toHaveBeenCalled();
    expect(instantQuery).not.toHaveBeenCalled();
    expect(request).not.toHaveBeenCalled();
  });

  it('leaves backups and alerts alone when their features are off', async () => {
    const { result } = renderLive(
      {},
      { alertingEnabled: false, backupManagementEnabled: false }
    );

    await waitFor(() =>
      expect(result.current.sources.inventory.status).toBe('ready')
    );

    expect(listArtifacts).not.toHaveBeenCalled();
    expect(getPrometheusAlertRules).not.toHaveBeenCalled();
    expect(result.current.sources.backups).toEqual({ status: 'disabled' });
    expect(result.current.snapshot.backups).toEqual({
      enabled: false,
      artifacts: [],
      scheduled: [],
    });
  });

  it('says nothing about backups or dashboards before they answer', async () => {
    vi.mocked(listArtifacts).mockReturnValue(new Promise(() => {}));
    vi.mocked(searchDashboards).mockReturnValue(new Promise(() => {}));

    const { result } = renderLive();

    await waitFor(() =>
      expect(result.current.sources.inventory.status).toBe('ready')
    );

    expect(result.current.sources.backups).toEqual({ status: 'loading' });
    expect(result.current.snapshot.backups).toBeNull();
    expect(result.current.sources.dashboards).toEqual({ status: 'loading' });
    expect(result.current.snapshot.dashboards).toBeNull();
  });

  it('flags a query RTA reports as running for 14 minutes', async () => {
    const service = RECORDED_SERVICES[5];

    vi.mocked(getRunningSessions).mockResolvedValue([
      {
        serviceId: service.serviceId,
        serviceName: service.serviceName,
        clusterName: 'orders',
        startTime: '',
        status: RealtimeSessionStatus.running,
      },
    ]);
    vi.mocked(searchQueries).mockResolvedValue({
      queries: [
        {
          serviceId: service.serviceId,
          serviceName: service.serviceName,
          queryId: 'r1',
          queryText: 'db.orders.aggregate([])',
          queryExecutionDuration: '840s',
          queryCollectTime: '',
          clientAddress: '',
          queryRawJson: '',
          mongoDbPayload: {
            dbInstanceAddress: '',
            clientAppName: '',
            databaseName: 'shop',
            collection: 'orders',
            operationStartTime: '',
            planSummary: '',
            operation: 'aggregate',
            username: '',
          },
        },
      ],
    });

    const { result } = renderLive();

    await waitFor(() =>
      expect(result.current.snapshot.realtime?.queries).toHaveLength(1)
    );

    const running = findRunningQueries(
      result.current.snapshot.realtime ?? {
        sessionServiceIds: [],
        queries: [],
      },
      toServiceInputs(RECORDED_SERVICES)
    );

    expect(running).toHaveLength(1);
    expect(runningAge(running[0].query)).toBe('14m');
  });

  it('reads running queries only for MongoDB services with a running session', async () => {
    vi.mocked(getRunningSessions).mockResolvedValue([
      {
        serviceId: RECORDED_SERVICES[5].serviceId,
        serviceName: RECORDED_SERVICES[5].serviceName,
        clusterName: 'orders',
        startTime: '',
        status: RealtimeSessionStatus.running,
      },
      {
        serviceId: 'mysql-service',
        serviceName: 'mysql-shop-1',
        clusterName: '',
        startTime: '',
        status: RealtimeSessionStatus.running,
      },
    ]);

    const { result } = renderLive();

    await waitFor(() =>
      expect(result.current.snapshot.realtime?.sessionServiceIds).toEqual([
        RECORDED_SERVICES[5].serviceId,
      ])
    );
    await waitFor(() => expect(searchQueries).toHaveBeenCalled());
    expect(vi.mocked(searchQueries).mock.calls[0][0]).toEqual({
      serviceIds: [RECORDED_SERVICES[5].serviceId],
    });
  });
});
