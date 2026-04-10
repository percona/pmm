import { mergeContainerIdsIntoPodMap } from './containerInventory';

describe('mergeContainerIdsIntoPodMap', () => {
  it('adds container_info paths to existing pod buckets', () => {
    const merged = mergeContainerIdsIntoPodMap(
      {
        '/k8s/ns/cluster1-pxc-0': ['/k8s/ns/cluster1-pxc-0/pmm-client'],
      },
      [
        '/k8s/ns/cluster1-pxc-0/pmm-client',
        '/k8s/ns/cluster1-pxc-0/pxc',
        '/k8s/ns/cluster1-pxc-0/logs',
      ]
    );
    expect(merged['/k8s/ns/cluster1-pxc-0']).toEqual([
      '/k8s/ns/cluster1-pxc-0/logs',
      '/k8s/ns/cluster1-pxc-0/pmm-client',
      '/k8s/ns/cluster1-pxc-0/pxc',
    ]);
  });
});
