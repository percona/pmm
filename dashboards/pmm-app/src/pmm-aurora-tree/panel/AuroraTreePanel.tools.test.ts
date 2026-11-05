import { dateTime } from '@grafana/data';
import { DEFAULT_OPTIONS } from './AuroraTreePanel.constants';
import { mergedFrame, timeRange, unmergedFrames } from './AuroraTreePanel.mocks';
import {
  formatValue,
  getServiceUrl,
  getWritersText,
  groupClusters,
  readInstances,
} from './AuroraTreePanel.tools';

jest.mock('@grafana/runtime', () => ({
  config: { appSubUrl: '/graph' },
}));

describe('AuroraTreePanel.tools::', () => {
  const clusters = groupClusters(readInstances([mergedFrame], DEFAULT_OPTIONS));

  it('groups instances by cluster and puts the unknown cluster last', () => {
    expect(clusters.map((cluster) => cluster.clusterId)).toEqual(['database-2', 'database-3', 'AWS cluster unknown']);
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

  it('links a service to its details dashboard under the Grafana sub-path with the time range', () => {
    expect(getServiceUrl('database-3-instance-1-reader', timeRange)).toBe(
      '/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details'
      + '?var-service_name=database-3-instance-1-reader&from=now-12h&to=now',
    );
  });

  it('uses epoch milliseconds for an absolute time range', () => {
    const from = dateTime(1700000000000);
    const to = dateTime(1700003600000);

    expect(getServiceUrl('db 1', { from, to, raw: { from, to } })).toBe(
      '/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details?var-service_name=db%201&from=1700000000000&to=1700003600000',
    );
  });
});
