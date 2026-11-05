import { pickListenWorkloadId } from './listenMapHelpers';

describe('pickListenWorkloadId', () => {
  it('prefers container_id over app_id', () => {
    expect(
      pickListenWorkloadId({
        app_id: '/k8s/ns/cluster1-pxc',
        container_id: '/k8s/ns/cluster1-pxc-0/pxc',
      })
    ).toBe('/k8s/ns/cluster1-pxc-0/pxc');
  });

  it('falls back to app_id', () => {
    expect(pickListenWorkloadId({ app_id: '/k8s/ns/x' })).toBe('/k8s/ns/x');
  });
});
