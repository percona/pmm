import { parseAppId } from './parseAppId';
import { ServiceEdge, ServiceMapData, ServiceMapOptions } from '../types';
import {
  aggregateByPod,
  listContainerAppIdsForPod,
  mergedContainerAppIdsForPod,
  podId,
  shouldHideWeakGreenEdge,
} from './podAggregate';

const baseOpts: ServiceMapOptions = {
  promDatasource: '',
  clickhouseDatasource: '',
  errorAmberThreshold: 1,
  errorRedThreshold: 5,
  minEdgeWeight: 0,
  groupByPod: true,
  hideWeakEdges: true,
  weakEdgeMaxRps: 1,
  labelMode: 'name',
  namespaceRenameMap: {},
  tracesDashboardUid: '',
  tracesViewPanel: 20,
  kubernetesApiClusterIPs: '',
  kubernetesApiserverEndpointIPs: '',
  destinationLabelOverrides: '',
  clusterTcpPorts: '',
};

describe('podId', () => {
  it('collapses container path', () => {
    expect(podId('/k8s/demo/vmagent-abc/vmagent')).toBe('/k8s/demo/vmagent-abc');
  });
  it('leaves single-segment workload unchanged', () => {
    expect(podId('/k8s/demo/something')).toBe('/k8s/demo/something');
  });
  it('leaves external ids', () => {
    expect(podId('10.0.0.1:443')).toBe('10.0.0.1:443');
  });
});

describe('shouldHideWeakGreenEdge', () => {
  const green: ServiceEdge = {
    id: 'a',
    source: 'a',
    target: 'b',
    rps: 0.2,
    errPct: 0,
    p95Ms: 1,
    bytesIn: 0,
    bytesOut: 0,
    tcpFailed: 0,
    health: 'green',
  };

  it('hides low green HTTP', () => {
    expect(shouldHideWeakGreenEdge(green, baseOpts)).toBe(true);
  });
  it('keeps TCP-only (rps 0)', () => {
    expect(
      shouldHideWeakGreenEdge(
        { ...green, rps: 0, bytesOut: 100, health: 'green' },
        baseOpts
      )
    ).toBe(false);
  });
  it('keeps amber', () => {
    expect(shouldHideWeakGreenEdge({ ...green, health: 'amber' }, baseOpts)).toBe(false);
  });
  it('respects hideWeakEdges off', () => {
    expect(shouldHideWeakGreenEdge(green, { ...baseOpts, hideWeakEdges: false })).toBe(false);
  });
});

describe('aggregateByPod', () => {
  const raw: ServiceMapData = {
    namespaces: ['demo'],
    nodes: [],
    edges: [
      {
        id: 'e1',
        source: '/k8s/demo/p1/c1',
        target: '/k8s/other/x/y',
        rps: 10,
        errPct: 0,
        p95Ms: 5,
        bytesIn: 0,
        bytesOut: 0,
        tcpFailed: 0,
        health: 'green',
      },
      {
        id: 'e2',
        source: '/k8s/demo/p1/c2',
        target: '/k8s/other/x/y',
        rps: 5,
        errPct: 0,
        p95Ms: 3,
        bytesIn: 0,
        bytesOut: 0,
        tcpFailed: 0,
        health: 'green',
      },
      {
        id: 'e3',
        source: '/k8s/demo/p1/c1',
        target: '/k8s/demo/p1/c2',
        rps: 1,
        errPct: 0,
        p95Ms: 1,
        bytesIn: 0,
        bytesOut: 0,
        tcpFailed: 0,
        health: 'green',
      },
    ],
  };

  raw.nodes = [
    {
      id: '/k8s/demo/p1/c1',
      parsed: { raw: '', namespace: 'demo', name: 'p1/c1', kind: 'k8s' },
      rps: 1,
      errPct: 0,
      p95Ms: 1,
      bytesIn: 0,
      bytesOut: 0,
      health: 'green',
    },
    {
      id: '/k8s/demo/p1/c2',
      parsed: { raw: '', namespace: 'demo', name: 'p1/c2', kind: 'k8s' },
      rps: 1,
      errPct: 0,
      p95Ms: 1,
      bytesIn: 0,
      bytesOut: 0,
      health: 'green',
    },
  ];

  it('merges parallel edges and drops same-pod', () => {
    const agg = aggregateByPod(raw, baseOpts, raw);
    const toOther = agg.edges.find((e) => e.target === '/k8s/other/x');
    expect(toOther?.rps).toBe(15);
    expect(agg.edges.some((e) => e.source === '/k8s/demo/p1' && e.target === '/k8s/demo/p1')).toBe(
      false
    );
  });

  it('lists containers for pod', () => {
    const ids = listContainerAppIdsForPod('/k8s/demo/p1', raw);
    expect(ids).toEqual(['/k8s/demo/p1/c1', '/k8s/demo/p1/c2']);
  });

  it('merges label-only containers with graph nodes', () => {
    const withLabels: ServiceMapData = {
      ...raw,
      podToContainerAppIds: {
        '/k8s/demo/p1': ['/k8s/demo/p1/c1', '/k8s/demo/p1/c2', '/k8s/demo/p1/sidecar'],
      },
    };
    expect(mergedContainerAppIdsForPod('/k8s/demo/p1', withLabels)).toEqual([
      '/k8s/demo/p1/c1',
      '/k8s/demo/p1/c2',
      '/k8s/demo/p1/sidecar',
    ]);
  });

  it('keeps a pod when strict subgraph leaves no edges but containers still match', () => {
    const containerId = '/k8s/demo/pmm-ha-2/pmm-ha';
    const orphan: ServiceMapData = {
      namespaces: ['demo'],
      nodes: [
        {
          id: containerId,
          parsed: parseAppId(containerId),
          rps: 1.68,
          errPct: 0,
          p95Ms: 0,
          bytesIn: 0,
          bytesOut: 0,
          health: 'green',
        },
      ],
      edges: [],
    };
    const agg = aggregateByPod(orphan, baseOpts, orphan);
    const pod = agg.nodes.find((n) => n.id === '/k8s/demo/pmm-ha-2');
    expect(pod).toBeDefined();
    expect(pod!.rps).toBeCloseTo(1.68);
  });
});
