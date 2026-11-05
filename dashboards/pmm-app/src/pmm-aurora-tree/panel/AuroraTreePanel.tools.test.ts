import { DEFAULT_OPTIONS } from './AuroraTreePanel.constants';
import { mergedFrame, unmergedFrames } from './AuroraTreePanel.mocks';
import {
  compareSortValues,
  filterClusters,
  formatValue,
  getAccountOptions,
  getClusterOptions,
  getRegionOptions,
  getServiceUrl,
  getWritersText,
  groupClusters,
  narrowSelection,
  readFilters,
  readInstances,
  toUrlQuery,
} from './AuroraTreePanel.tools';
import { ClusterFilters } from './AuroraTreePanel.types';

jest.mock('@grafana/runtime', () => ({
  config: { appSubUrl: '/graph' },
}));

describe('AuroraTreePanel.tools::', () => {
  const clusters = groupClusters(readInstances([mergedFrame], DEFAULT_OPTIONS));

  it('groups instances by cluster and puts the unknown cluster last in its region', () => {
    expect(clusters.map((cluster) => cluster.clusterId)).toEqual([
      'database-2',
      'database-3',
      'AWS cluster unknown',
      'pmm-qa-aurora3-mysql',
    ]);
  });

  it('aggregates a cluster: counts, writers, summed queries and max storage', () => {
    const [, database3] = clusters;

    expect(database3).toMatchObject({
      region: 'eu-north-1',
      awsAccountId: '111111111111',
      monitoredInstances: 2,
      exportersConnected: 2,
      writers: 1,
      qps: 24.75,
      storageBytes: 5000,
    });
    expect(database3.instances).toEqual([
      {
        serviceName: 'database-3-instance-1', role: 'writer', up: 1, qps: 20.5,
      },
      {
        serviceName: 'database-3-instance-1-reader', role: 'reader', up: 1, qps: 4.25,
      },
    ]);
  });

  it('marks an instance without innodb_read_only as unknown role', () => {
    const unknown = clusters[2];

    expect(unknown.instances).toEqual([{ serviceName: 'aurora-stale', role: 'unknown', up: 0 }]);
    expect(unknown).toMatchObject({ exportersConnected: 0, writers: 0 });
    expect(unknown.qps).toBeUndefined();
    expect(unknown.storageBytes).toBeUndefined();
  });

  it('joins one frame per query the same way as a merged frame', () => {
    expect(groupClusters(readInstances(unmergedFrames, DEFAULT_OPTIONS))).toEqual(clusters);
  });

  it('reads the value fields named in the options', () => {
    const options = { ...DEFAULT_OPTIONS, readOnlyField: 'Value #A', upField: 'Value #B' };
    const [database2] = groupClusters(readInstances([mergedFrame], options));

    expect(database2.instances[0]).toMatchObject({ role: 'reader', up: 0 });
  });

  it('shows "1 writer" only for exactly one writer', () => {
    expect(getWritersText(1)).toBe('1 writer');
    expect(getWritersText(0)).toBe('changing');
    expect(getWritersText(2)).toBe('changing');
  });

  it('formats values with Grafana units', () => {
    expect(formatValue(24.75, 'qps', 2)).toBe('24.75 qps');
    expect(formatValue(undefined, 'qps', 2)).toBe('-');
    expect(formatValue(5 * 1024 * 1024 * 1024, 'bytes')).toBe('5 GiB');
  });

  it('links a service to its details dashboard under the Grafana sub-path without a time range', () => {
    expect(getServiceUrl('database-3-instance-1-reader')).toBe(
      '/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details?var-service_name=database-3-instance-1-reader',
    );
    expect(getServiceUrl('db 1')).toBe('/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details?var-service_name=db%201');
  });
});

describe('AuroraTreePanel.tools:: filters', () => {
  const clusters = groupClusters(readInstances([mergedFrame], DEFAULT_OPTIONS));
  const noFilters: ClusterFilters = {
    query: '', regions: [], accounts: [], clusters: [],
  };
  const values = (options: Array<{ value: string }>) => options.map(({ value }) => value);
  const filter = (filters: Partial<ClusterFilters>) => filterClusters(clusters, { ...noFilters, ...filters });
  const clusterIds = (filters: Partial<ClusterFilters>) => filter(filters).clusters
    .map(({ clusterId }) => clusterId);

  it('keeps every cluster without filters', () => {
    expect(filter({})).toEqual({ clusters, autoExpanded: new Set() });
  });

  it('searches cluster ID, account and region case-insensitively, keeping all instances', () => {
    expect(clusterIds({ query: 'DATABASE-3' })).toEqual(['database-3']);
    expect(clusterIds({ query: '2222' })).toEqual(['pmm-qa-aurora3-mysql']);
    expect(clusterIds({ query: 'north' })).toEqual(['database-2', 'database-3', 'AWS cluster unknown']);
    expect(filter({ query: 'database-3' }).clusters[0].instances).toHaveLength(2);
    expect(filter({ query: 'database-3' }).autoExpanded.size).toBe(0);
  });

  it('narrows a cluster matched only by service name to those instances and expands it', () => {
    const { clusters: matching, autoExpanded } = filter({ query: ' Reader ' });

    expect(matching).toHaveLength(1);
    expect(matching[0]).toMatchObject({ clusterId: 'database-3', monitoredInstances: 2 });
    expect(matching[0].instances.map(({ serviceName }) => serviceName)).toEqual(['database-3-instance-1-reader']);
    expect(autoExpanded).toEqual(new Set([matching[0].key]));
  });

  it('combines account and cluster filters with the search', () => {
    expect(clusterIds({ accounts: ['111111111111'] })).toEqual(['database-2', 'database-3']);
    expect(clusterIds({ accounts: ['111111111111'], clusters: ['database-3'] })).toEqual(['database-3']);
    expect(clusterIds({ accounts: ['222222222222'], clusters: ['database-3'] })).toEqual([]);
    expect(clusterIds({ accounts: ['111111111111', '222222222222'], query: 'instance-1' })).toEqual([
      'database-2',
      'database-3',
      'pmm-qa-aurora3-mysql',
    ]);
  });

  it('finds the unknown cluster group by search and by cluster filter', () => {
    expect(clusterIds({ query: 'unknown' })).toEqual(['AWS cluster unknown']);
    expect(clusterIds({ clusters: ['AWS cluster unknown'] })).toEqual(['AWS cluster unknown']);
    expect(clusterIds({ query: 'stale' })).toEqual(['AWS cluster unknown']);
  });

  it('returns nothing when no cluster matches', () => {
    expect(filter({ query: 'no-such-cluster' })).toEqual({ clusters: [], autoExpanded: new Set() });
  });

  it('filters by region, alone and combined with the other filters', () => {
    expect(clusterIds({ regions: ['us-east-2'] })).toEqual(['pmm-qa-aurora3-mysql']);
    expect(clusterIds({ regions: ['eu-north-1'] })).toEqual(['database-2', 'database-3', 'AWS cluster unknown']);
    expect(clusterIds({ regions: ['eu-north-1', 'us-east-2'], accounts: ['222222222222'] })).toEqual([
      'pmm-qa-aurora3-mysql',
    ]);
    expect(clusterIds({ regions: ['us-east-2'], clusters: ['database-3'] })).toEqual([]);
    expect(clusterIds({ regions: ['eu-north-1'], query: 'reader' })).toEqual(['database-3']);
  });

  it('builds region options from the data', () => {
    expect(getRegionOptions(clusters)).toEqual([
      { label: 'eu-north-1', value: 'eu-north-1' },
      { label: 'us-east-2', value: 'us-east-2' },
    ]);
  });

  it('builds account options narrowed to the selected regions, without the empty account', () => {
    expect(getAccountOptions(clusters, [])).toEqual([
      { label: '111111111111', value: '111111111111', description: 'eu-north-1' },
      { label: '222222222222', value: '222222222222', description: 'us-east-2' },
    ]);
    expect(values(getAccountOptions(clusters, ['us-east-2']))).toEqual(['222222222222']);
  });

  it('builds cluster options narrowed to the selected regions and accounts, unknown group last', () => {
    expect(values(getClusterOptions(clusters, { regions: [], accounts: [] }))).toEqual([
      'database-2',
      'database-3',
      'pmm-qa-aurora3-mysql',
      'AWS cluster unknown',
    ]);
    expect(values(getClusterOptions(clusters, { regions: [], accounts: ['222222222222'] }))).toEqual([
      'pmm-qa-aurora3-mysql',
    ]);
    expect(values(getClusterOptions(clusters, { regions: ['eu-north-1'], accounts: [] }))).toEqual([
      'database-2',
      'database-3',
      'AWS cluster unknown',
    ]);
  });

  it('drops selected accounts and clusters the selected regions and accounts no longer offer', () => {
    const selection = {
      regions: [],
      accounts: ['111111111111', '222222222222'],
      clusters: ['database-3', 'pmm-qa-aurora3-mysql'],
    };

    expect(narrowSelection(clusters, { ...selection, regions: ['us-east-2'] })).toEqual({
      regions: ['us-east-2'],
      accounts: ['222222222222'],
      clusters: ['pmm-qa-aurora3-mysql'],
    });
    expect(narrowSelection(clusters, { ...selection, accounts: ['111111111111'] })).toEqual({
      regions: [],
      accounts: ['111111111111'],
      clusters: ['database-3'],
    });
    expect(narrowSelection(clusters, selection)).toEqual(selection);
  });

  it('reads filters from URL parameters', () => {
    expect(readFilters({})).toEqual(noFilters);
    expect(readFilters({
      'aurora-q': 'reader',
      'aurora-region': ['eu-north-1', 'us-east-2'],
      'aurora-account': '111111111111',
      'aurora-cluster': ['database-2', 'database-3'],
    })).toEqual({
      query: 'reader',
      regions: ['eu-north-1', 'us-east-2'],
      accounts: ['111111111111'],
      clusters: ['database-2', 'database-3'],
    });
  });

  it('writes filters to URL parameters, removing empty ones', () => {
    expect(toUrlQuery(noFilters)).toEqual({
      'aurora-q': null,
      'aurora-region': null,
      'aurora-account': null,
      'aurora-cluster': null,
    });
    expect(toUrlQuery({
      query: 'db', regions: ['r'], accounts: ['1'], clusters: ['c'],
    })).toEqual({
      'aurora-q': 'db',
      'aurora-region': ['r'],
      'aurora-account': ['1'],
      'aurora-cluster': ['c'],
    });
  });

  it('sorts missing values last in both directions', () => {
    expect(compareSortValues(undefined, 1, false)).toBe(1);
    expect(compareSortValues(undefined, 1, true)).toBe(1);
    expect(compareSortValues(1, undefined, true)).toBe(-1);
    expect(compareSortValues(1, 3, false)).toBeLessThan(0);
    expect(compareSortValues(1, 3, true)).toBeGreaterThan(0);
    expect(compareSortValues('b', 'a', false)).toBeGreaterThan(0);
  });
});
