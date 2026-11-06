import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { RDS_QUERIES } from 'api/rds';
import { SettingsContext } from 'contexts/settings';
import type { DataFrameJSON } from 'types/metrics.types';
import type { GrafanaDatasource } from 'types/settings.types';
import { wrapWithQueryProvider, wrapWithSettings } from 'utils/testUtils';
import { useRdsInstances } from './useRds';

const { post } = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock('api/api', () => ({ grafanaApi: { post } }));

const DATASOURCES: Record<string, GrafanaDatasource> = {
  Metrics: { uid: 'metrics-uid', type: 'prometheus' },
  ClickHouse: { uid: 'clickhouse-uid', type: 'grafana-clickhouse-datasource' },
};

// One series of an instant query, as Grafana encodes it: a time and a value field.
const series = (
  labels: Record<string, string>,
  value: number | null
): DataFrameJSON => ({
  schema: {
    fields: [
      { name: 'Time', type: 'time' },
      { name: 'Value', type: 'number', labels },
    ],
  },
  data: { values: [[1793921502528], [value]] },
});

const writer = {
  service_name: 'database-3-instance-1',
  node_name: 'database-3-instance-1',
  node_id: 'node-1',
  service_type: 'mysql',
  region: 'eu-north-1',
};

const results = {
  up: {
    frames: [
      series(writer, 1),
      series(
        {
          service_name: 'legacy-pg',
          node_id: 'node-2',
          service_type: 'postgresql',
          region: 'eu-north-1',
        },
        0
      ),
    ],
  },
  info: {
    frames: [
      series(
        {
          node_id: 'node-1',
          region: 'eu-north-1',
          aws_account_id: '111111111111',
          aws_cluster_id: 'database-3',
          aws_engine: 'aurora-mysql',
          instance: 'database-3-instance-1',
        },
        1
      ),
    ],
  },
  innodbReadOnly: {
    frames: [series({ service_name: 'database-3-instance-1' }, 0)],
  },
  readOnly: { frames: [] },
  // NaN reaches the browser as null.
  inRecovery: { frames: [series({ service_name: 'legacy-pg' }, null)] },
  throughput: {
    frames: [series({ service_name: 'database-3-instance-1' }, 20.5)],
  },
  volumeUsedBytes: { frames: [series({ node_id: 'node-1' }, 5000)] },
  freeStorageBytes: {},
};

const renderRdsHook = (
  datasources: Record<string, GrafanaDatasource> | null = DATASOURCES
) =>
  renderHook(() => useRdsInstances(), {
    wrapper: ({ children }: { children: ReactNode }) =>
      wrapWithQueryProvider(
        datasources ? (
          wrapWithSettings(<>{children}</>, { frontend: { datasources } })
        ) : (
          <SettingsContext.Provider value={{ isLoading: true, settings: null }}>
            {children}
          </SettingsContext.Provider>
        )
      ),
  });

describe('useRdsInstances', () => {
  beforeEach(() => {
    post.mockReset();
  });

  it('runs every query as an instant query against the Metrics data source', async () => {
    post.mockResolvedValue({ data: { results } });

    const { result } = renderRdsHook();

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(post).toHaveBeenCalledTimes(1);

    const [url, body, config] = post.mock.calls[0];

    expect(url).toBe('/ds/query');
    expect(config).toBeUndefined();
    expect(body).toMatchObject({ from: 'now-1h', to: 'now' });
    expect(body.queries.map(({ refId }: { refId: string }) => refId)).toEqual(
      Object.keys(RDS_QUERIES)
    );
    body.queries.forEach((query: Record<string, unknown>) => {
      expect(query).toMatchObject({
        datasource: { type: 'prometheus', uid: 'metrics-uid' },
        instant: true,
        range: false,
        expr: RDS_QUERIES[query.refId as keyof typeof RDS_QUERIES],
      });
    });
  });

  it('joins service metrics by service and AWS labels and storage by node', async () => {
    post.mockResolvedValue({ data: { results } });

    const { result } = renderRdsHook();

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual([
      {
        serviceName: 'database-3-instance-1',
        nodeName: 'database-3-instance-1',
        nodeId: 'node-1',
        serviceType: 'mysql',
        region: 'eu-north-1',
        awsAccountId: '111111111111',
        clusterId: 'database-3',
        awsEngine: 'aurora-mysql',
        instanceId: 'database-3-instance-1',
        sourceInstanceId: '',
        sourceRegion: '',
        up: 1,
        innodbReadOnly: 0,
        throughput: 20.5,
        volumeUsedBytes: 5000,
      },
      {
        serviceName: 'legacy-pg',
        nodeName: '',
        nodeId: 'node-2',
        serviceType: 'postgresql',
        region: 'eu-north-1',
        awsAccountId: '',
        clusterId: '',
        awsEngine: '',
        instanceId: '',
        sourceInstanceId: '',
        sourceRegion: '',
        up: 0,
      },
    ]);
  });

  // Grafana answers 400 for the whole request when one query fails, with every result in the body.
  const rejectWith = (failed: Record<string, unknown>) =>
    post.mockRejectedValue({
      isAxiosError: true,
      response: { status: 400, data: { results: { ...results, ...failed } } },
    });

  it('leaves out the values of an optional query that fails', async () => {
    rejectWith({ throughput: { status: 400, error: 'bad query' } });

    const { result } = renderRdsHook();

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.[0]).toMatchObject({
      serviceName: 'database-3-instance-1',
      clusterId: 'database-3',
      up: 1,
      volumeUsedBytes: 5000,
    });
    expect(result.current.data?.[0].throughput).toBeUndefined();
  });

  it('fails when a required query fails', async () => {
    rejectWith({ info: { status: 400, error: 'bad query' } });

    const { result } = renderRdsHook();

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error?.message).toBe('bad query');
  });

  it('fails when the request fails otherwise', async () => {
    post.mockRejectedValue(new Error('Network Error'));

    const { result } = renderRdsHook();

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error?.message).toBe('Network Error');
  });

  it('fails without the Metrics data source', async () => {
    const { result } = renderRdsHook({ ClickHouse: DATASOURCES.ClickHouse });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error?.message).toBe(
      "Data source 'Metrics' not found"
    );
    expect(post).not.toHaveBeenCalled();
  });

  it('waits for the settings', () => {
    const { result } = renderRdsHook(null);

    expect(result.current.isPending).toBe(true);
    expect(result.current.fetchStatus).toBe('idle');
    expect(post).not.toHaveBeenCalled();
  });
});
