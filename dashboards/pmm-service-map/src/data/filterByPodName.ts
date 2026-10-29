import { ServiceEdge, ServiceMapData, ServiceNode } from '../types';
import { formatNodeLabel, parseAppId } from './parseAppId';

function k8sPathAfterNamespace(appId: string): string | null {
  const m = appId.match(/^\/k8s\/[^/]+\/(.+)$/);
  return m ? m[1] : null;
}

function matchesServiceNode(n: ServiceNode, qLower: string): boolean {
  if (!qLower) {
    return true;
  }
  const hay = (s: string) => s.toLowerCase().includes(qLower);

  if (hay(n.id)) {
    return true;
  }

  const p = n.parsed;
  if (hay(formatNodeLabel(p, 'name'))) {
    return true;
  }
  if (hay(formatNodeLabel(p, 'namespace-name'))) {
    return true;
  }
  if (p.displayName && hay(p.displayName)) {
    return true;
  }

  const rest = k8sPathAfterNamespace(n.id);
  if (rest) {
    const podSegment = rest.split('/')[0];
    if (hay(podSegment) || hay(rest)) {
      return true;
    }
  }

  return false;
}

/**
 * Keep namespaces' pod→container map entries when namespace chips restrict the graph.
 */
export function filterPodToContainerAppIdsByNamespaces(
  podMap: Record<string, string[]> | undefined,
  nsPick: Set<string>
): Record<string, string[]> | undefined {
  if (!podMap || nsPick.size === 0) {
    return podMap;
  }
  const out: Record<string, string[]> = {};
  for (const [pod, ids] of Object.entries(podMap)) {
    const ns = parseAppId(pod).namespace;
    if (nsPick.has(ns)) {
      out[pod] = ids;
    }
  }
  return out;
}

/**
 * Case-insensitive substring on id / labels (and k8s path segments).
 * Keeps every node that matches, plus **all neighbors** on any edge touching a match
 * (incoming/outgoing), so the subgraph stays connected to peers even when peers don't match.
 */
export function filterServiceMapByPodSubstring(data: ServiceMapData, query: string): ServiceMapData {
  const qLower = query.trim().toLowerCase();
  if (!qLower) {
    return data;
  }

  const matchingIds = new Set(data.nodes.filter((n) => matchesServiceNode(n, qLower)).map((n) => n.id));
  if (matchingIds.size === 0) {
    return {
      ...data,
      nodes: [],
      edges: [],
      namespaces: [],
    };
  }

  const keptNodeIds = new Set<string>(matchingIds);
  const keptEdges: ServiceEdge[] = [];
  for (const e of data.edges) {
    if (matchingIds.has(e.source) || matchingIds.has(e.target)) {
      keptEdges.push(e);
      keptNodeIds.add(e.source);
      keptNodeIds.add(e.target);
    }
  }

  const nodes = data.nodes.filter((n) => keptNodeIds.has(n.id));
  const namespaces = [...new Set(nodes.map((n) => n.parsed.namespace))].sort();
  return {
    ...data,
    nodes,
    edges: keptEdges,
    namespaces,
  };
}
