import { ServiceMapData } from '../types';
import { filterServiceMapByPodSubstring } from './filterByPodName';
import { parseAppId } from './parseAppId';

const mkNode = (id: string) => ({
  id,
  parsed: parseAppId(id),
  rps: 1,
  errPct: 0,
  p95Ms: 0,
  bytesIn: 0,
  bytesOut: 0,
  health: 'green' as const,
});

describe('filterServiceMapByPodSubstring', () => {
  it('returns data unchanged when query empty', () => {
    const data: ServiceMapData = {
      namespaces: ['demo'],
      nodes: [mkNode('/k8s/demo/a/x'), mkNode('/k8s/demo/b/y')],
      edges: [
        {
          id: 'e',
          source: '/k8s/demo/a/x',
          target: '/k8s/demo/b/y',
          rps: 1,
          errPct: 0,
          p95Ms: 0,
          bytesIn: 0,
          bytesOut: 0,
          tcpFailed: 0,
          health: 'green',
        },
      ],
    };
    expect(filterServiceMapByPodSubstring(data, '   ')).toBe(data);
  });

  it('keeps matching nodes and neighbors on any incident edge', () => {
    const a = '/k8s/demo/foo-pod/c1';
    const b = '/k8s/demo/bar-pod/c2';
    const c = '/k8s/demo/alone/c3';
    const data: ServiceMapData = {
      namespaces: ['demo'],
      nodes: [mkNode(a), mkNode(b), mkNode(c)],
      edges: [
        {
          id: 'ab',
          source: a,
          target: b,
          rps: 1,
          errPct: 0,
          p95Ms: 0,
          bytesIn: 0,
          bytesOut: 0,
          tcpFailed: 0,
          health: 'green',
        },
      ],
    };
    const out = filterServiceMapByPodSubstring(data, 'foo');
    expect(out.nodes.map((n) => n.id).sort()).toEqual([a, b].sort());
    expect(out.edges).toHaveLength(1);
  });

  it('returns empty when nothing matches', () => {
    const data: ServiceMapData = {
      namespaces: ['demo'],
      nodes: [mkNode('/k8s/demo/a/x')],
      edges: [],
    };
    const out = filterServiceMapByPodSubstring(data, 'zzz');
    expect(out.nodes).toHaveLength(0);
    expect(out.edges).toHaveLength(0);
  });

  it('matches displayName when raw id differs', () => {
    const n = mkNode('10.0.0.1:443');
    n.parsed.displayName = 'pmm-ha-2';
    const data: ServiceMapData = {
      namespaces: ['external'],
      nodes: [n],
      edges: [],
    };
    const out = filterServiceMapByPodSubstring(data, 'pmm-ha-2');
    expect(out.nodes).toHaveLength(1);
  });
});
