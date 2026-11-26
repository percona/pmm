import { describe, expect, it } from 'vitest';
import type { ManagedService } from 'types/services.types';
import { TEST_MANAGED_SERVICE } from 'utils/testStubs';
import { MonitoringStatus } from '../Inventory.types';
import { toClusterRows, toServiceRows } from './Services.utils';

const service = (
  serviceName: string,
  cluster: string,
  customLabels: Record<string, string> = {}
): ManagedService => ({
  ...TEST_MANAGED_SERVICE,
  serviceId: serviceName,
  serviceName,
  cluster,
  customLabels,
});

describe('toServiceRows', () => {
  it('sorts the services by name and computes their monitoring status', () => {
    const rows = toServiceRows([service('b', ''), service('a', '')]);

    expect(rows.map(({ serviceName }) => serviceName)).toEqual(['a', 'b']);
    expect(rows[0].agentsStatus).toBe(MonitoringStatus.ok);
  });

  it('falls back to the cluster custom label', () => {
    expect(
      toServiceRows([service('a', '', { cluster: 'labelled' })])[0].cluster
    ).toBe('labelled');
  });
});

describe('toClusterRows', () => {
  it('groups non-clustered services first, then clusters by name', () => {
    const clusters = toClusterRows(
      toServiceRows([
        service('c', 'cluster-b'),
        service('a', 'cluster-a'),
        service('b', ''),
        service('d', 'cluster-a'),
      ])
    );

    expect(
      clusters.map(({ clusterId, name, services }) => ({
        clusterId,
        name,
        services: services.map(({ serviceName }) => serviceName),
      }))
    ).toEqual([
      {
        clusterId: 'cluster:',
        name: 'Non-clustered services',
        services: ['b'],
      },
      {
        clusterId: 'cluster:cluster-a',
        name: 'cluster-a',
        services: ['a', 'd'],
      },
      { clusterId: 'cluster:cluster-b', name: 'cluster-b', services: ['c'] },
    ]);
  });
});
