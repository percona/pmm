import { parseAppId } from './parseAppId';
import { computeHealth } from './mapHealth';
import {
  HealthStatus,
  ServiceEdge,
  ServiceMapData,
  ServiceMapOptions,
  ServiceNode,
} from '../types';

/** Coroot-style /k8s/ns/pod/container → /k8s/ns/pod when a container segment exists. */
export function podId(appId: string): string {
  const m = appId.match(/^\/k8s\/([^/]+)\/(.+)$/);
  if (!m) {
    return appId;
  }
  const rest = m[2];
  if (!rest.includes('/')) {
    return appId;
  }
  const podSegment = rest.split('/')[0];
  return `/k8s/${m[1]}/${podSegment}`;
}

/** Hide low-traffic healthy HTTP edges; keep TCP-only (rps===0) and unhealthy edges. */
export function shouldHideWeakGreenEdge(edge: ServiceEdge, opts: ServiceMapOptions): boolean {
  if (opts.hideWeakEdges === false) {
    return false;
  }
  if (edge.health !== 'green') {
    return false;
  }
  if (edge.tcpFailed > 0) {
    return false;
  }
  if (edge.rps <= 0) {
    return false;
  }
  const maxRps = opts.weakEdgeMaxRps ?? 1;
  if (edge.rps >= maxRps) {
    return false;
  }
  return true;
}

function mergeEdgeMetricsWithOpts(a: ServiceEdge, b: ServiceEdge, opts: ServiceMapOptions): ServiceEdge {
  const rps = a.rps + b.rps;
  const errRpsA = a.rps > 0 ? (a.errPct / 100) * a.rps : 0;
  const errRpsB = b.rps > 0 ? (b.errPct / 100) * b.rps : 0;
  const errRps = errRpsA + errRpsB;
  const errPct = rps > 0 ? (errRps / rps) * 100 : 0;
  const tcpFailed = a.tcpFailed + b.tcpFailed;
  const p95Ms = Math.max(a.p95Ms, b.p95Ms);
  const bytesIn = a.bytesIn + b.bytesIn;
  const bytesOut = a.bytesOut + b.bytesOut;
  const id = `${a.source}→${a.target}`;
  const health = computeHealth(errPct, tcpFailed, rps, opts);
  return {
    id,
    source: a.source,
    target: a.target,
    rps,
    errPct,
    p95Ms,
    bytesIn,
    bytesOut,
    tcpFailed,
    health,
  };
}

/**
 * Remap container-level edges to pod-level, merge, drop same-pod internal edges.
 * @param rawContainerData same topology as `data` (container-level), used only to count containers per pod.
 */
export function aggregateByPod(
  data: ServiceMapData,
  opts: ServiceMapOptions,
  rawContainerData: ServiceMapData
): ServiceMapData {
  const merged = new Map<string, ServiceEdge>();

  for (const e of data.edges) {
    const s = podId(e.source);
    const t = podId(e.target);
    if (s === t) {
      continue;
    }
    const key = `${s}→${t}`;
    const existing = merged.get(key);
    if (!existing) {
      merged.set(key, {
        ...e,
        id: key,
        source: s,
        target: t,
      });
    } else {
      merged.set(key, mergeEdgeMetricsWithOpts(existing, e, opts));
    }
  }

  const edges = Array.from(merged.values());
  const built = buildServiceMapDataFromEdges(edges, opts);
  const presentPodIds = new Set(built.nodes.map((n) => n.id));

  /** After strict name filter, a pod may have matching containers but no edge whose *both* ends match — edges-only layout dropped it. */
  const byPod = new Map<string, ServiceNode[]>();
  for (const n of data.nodes) {
    const pid = podId(n.id);
    let list = byPod.get(pid);
    if (!list) {
      list = [];
      byPod.set(pid, list);
    }
    list.push(n);
  }

  const orphanPodNodes: ServiceNode[] = [];
  for (const [pid, containers] of byPod) {
    if (!presentPodIds.has(pid)) {
      orphanPodNodes.push(buildPodAggregateNodeFromContainers(pid, containers, opts));
    }
  }

  const namespaceSet = new Set(built.namespaces);
  for (const n of orphanPodNodes) {
    namespaceSet.add(n.parsed.namespace);
  }

  const nodes = [...built.nodes, ...orphanPodNodes]
    .sort((a, b) => a.id.localeCompare(b.id))
    .map((n) => ({
      ...n,
      podChildContainerCount: mergedContainerAppIdsForPod(n.id, rawContainerData).length,
    }));

  return {
    ...built,
    nodes,
    namespaces: Array.from(namespaceSet).sort(),
    podToContainerAppIds: rawContainerData.podToContainerAppIds,
  };
}

/** Merge container-level nodes into one pod-level node (metrics only; used for orphan pods). */
function buildPodAggregateNodeFromContainers(
  podAggregateId: string,
  containers: ServiceNode[],
  opts: ServiceMapOptions
): ServiceNode {
  const totalRps = containers.reduce((s, c) => s + c.rps, 0);
  const errRps = containers.reduce((s, c) => s + c.rps * (c.errPct / 100), 0);
  const errPct = totalRps > 0 ? (errRps / totalRps) * 100 : 0;
  const p95Ms = containers.reduce((m, c) => Math.max(m, c.p95Ms), 0);
  const bytesIn = containers.reduce((s, c) => s + c.bytesIn, 0);
  const bytesOut = containers.reduce((s, c) => s + c.bytesOut, 0);
  const parsed = parseAppId(podAggregateId);
  return {
    id: podAggregateId,
    parsed,
    rps: totalRps,
    errPct,
    p95Ms,
    bytesIn,
    bytesOut,
    health: computeHealth(errPct, 0, totalRps, opts),
  };
}

function buildServiceMapDataFromEdges(edges: ServiceEdge[], opts: ServiceMapOptions): ServiceMapData {
  const nodeIds = new Set<string>();
  for (const e of edges) {
    if (e.source) {
      nodeIds.add(e.source);
    }
    if (e.target) {
      nodeIds.add(e.target);
    }
  }

  const nodeMetrics = new Map<
    string,
    {
      outRps: number;
      inRps: number;
      outErrRps: number;
      inErrRps: number;
      bytesIn: number;
      bytesOut: number;
      latency: number;
      tcpFailed: number;
    }
  >();

  function getOrInitNode(nid: string) {
    let nm = nodeMetrics.get(nid);
    if (!nm) {
      nm = {
        outRps: 0,
        inRps: 0,
        outErrRps: 0,
        inErrRps: 0,
        bytesIn: 0,
        bytesOut: 0,
        latency: 0,
        tcpFailed: 0,
      };
      nodeMetrics.set(nid, nm);
    }
    return nm;
  }

  for (const e of edges) {
    const edgeErrRps = e.rps > 0 ? (e.errPct / 100) * e.rps : 0;
    const sm = getOrInitNode(e.source);
    sm.outRps += e.rps;
    sm.outErrRps += edgeErrRps;
    sm.bytesOut += e.bytesOut;
    sm.latency = Math.max(sm.latency, e.p95Ms);
    sm.tcpFailed += e.tcpFailed;

    const tm = getOrInitNode(e.target);
    tm.inRps += e.rps;
    tm.inErrRps += edgeErrRps;
    tm.bytesIn += e.bytesIn;
  }

  const namespaceSet = new Set<string>();
  const nodes: ServiceNode[] = [];
  for (const id of nodeIds) {
    const parsed = parseAppId(id);
    namespaceSet.add(parsed.namespace);
    const nm = nodeMetrics.get(id) ?? {
      outRps: 0,
      inRps: 0,
      outErrRps: 0,
      inErrRps: 0,
      bytesIn: 0,
      bytesOut: 0,
      latency: 0,
      tcpFailed: 0,
    };
    const nodeRps = nm.outRps > 0 ? nm.outRps : nm.inRps;
    const nodeErrRps = nm.outErrRps + nm.inErrRps;
    const errPct = nodeRps > 0 ? (nodeErrRps / nodeRps) * 100 : 0;
    nodes.push({
      id,
      parsed,
      rps: nodeRps,
      errPct,
      p95Ms: nm.latency,
      bytesIn: nm.bytesIn,
      bytesOut: nm.bytesOut,
      health: computeHealth(errPct, nm.tcpFailed, nodeRps, opts),
    });
  }

  return {
    nodes,
    edges,
    namespaces: Array.from(namespaceSet).sort(),
  };
}

/**
 * pod id → container app_ids under that pod (one pass over raw.nodes).
 * Skips nodes that are already pod-level ids (no container segment).
 */
function buildContainersByPodMap(raw: ServiceMapData): Map<string, string[]> {
  const m = new Map<string, string[]>();
  for (const n of raw.nodes) {
    const p = podId(n.id);
    if (n.id === p) {
      continue;
    }
    let list = m.get(p);
    if (!list) {
      list = [];
      m.set(p, list);
    }
    list.push(n.id);
  }
  for (const list of m.values()) {
    list.sort();
  }
  return m;
}

/** Union of label-derived inventory and graph nodes (sorted, deduped). */
export function mergedContainerAppIdsForPod(podAggregateId: string, raw: ServiceMapData): string[] {
  const fromNodes = buildContainersByPodMap(raw).get(podAggregateId) ?? [];
  const fromLabels = raw.podToContainerAppIds?.[podAggregateId] ?? [];
  const merged = new Set<string>([...fromNodes, ...fromLabels]);
  return Array.from(merged).sort();
}

/** Containers whose podId maps to the given pod id. */
export function listContainerAppIdsForPod(podAggregateId: string, raw: ServiceMapData): string[] {
  return mergedContainerAppIdsForPod(podAggregateId, raw);
}

/** One row per container under the pod; synthetic zeros when a container has no graph node. */
export function getChildContainerNodesForPod(podAggregateId: string, raw: ServiceMapData): ServiceNode[] {
  const ids = mergedContainerAppIdsForPod(podAggregateId, raw);
  const byId = new Map(raw.nodes.map((n) => [n.id, n]));
  return ids.map((id) => {
    const existing = byId.get(id);
    if (existing) {
      return existing;
    }
    return {
      id,
      parsed: parseAppId(id),
      rps: 0,
      errPct: 0,
      p95Ms: 0,
      bytesIn: 0,
      bytesOut: 0,
      health: 'unknown' as HealthStatus,
    };
  });
}
