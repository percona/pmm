import { DataFrame, FieldType } from '@grafana/data';
import { parseAppId } from './parseAppId';
import { HealthStatus, ServiceEdge, ServiceMapData, ServiceNode, ServiceMapOptions } from '../types';

interface RawEdgeAcc {
  rps: number;
  errRps: number;
  latency: number;
  bytesIn: number;
  bytesOut: number;
  tcpFailed: number;
}

function computeHealth(errPct: number, tcpFailed: number, totalRps: number, opts: ServiceMapOptions): HealthStatus {
  if (totalRps <= 0 && tcpFailed <= 0) {
    return 'unknown';
  }
  if (errPct >= opts.errorRedThreshold || tcpFailed > 0) {
    return 'red';
  }
  if (errPct >= opts.errorAmberThreshold) {
    return 'amber';
  }
  return 'green';
}

interface LabeledRow {
  value: number;
  app: string;
  dest: string;
  actualDest: string;
  proto: string;
  status: string;
}

function extractLabeledSeries(frames: DataFrame[]): LabeledRow[] {
  const rows: LabeledRow[] = [];
  for (const frame of frames) {
    const valueField = frame.fields.find((f) => f.type === FieldType.number);
    if (!valueField || frame.length === 0) {
      continue;
    }
    const app = valueField.labels?.['app_id'] ?? valueField.labels?.['app'] ?? '';
    const dest = valueField.labels?.['destination'] ?? valueField.labels?.['dest'] ?? '';
    const actualDest = valueField.labels?.['actual_destination'] ?? '';
    const proto = valueField.labels?.['proto'] ?? '';
    const status = valueField.labels?.['status'] ?? '';
    for (let i = 0; i < frame.length; i++) {
      rows.push({ app, dest, actualDest, proto, status, value: valueField.values[i] as number });
    }
  }
  return rows;
}

/**
 * Build a listen_addr → app_id lookup from container_net_tcp_listen_info frames.
 * Multiple listen_addrs can map to the same app_id. We pick the first seen.
 */
export function buildIpToAppIdMap(frames: DataFrame[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const frame of frames) {
    const valueField = frame.fields.find((f) => f.type === FieldType.number);
    if (!valueField) {
      continue;
    }
    const listenAddr = valueField.labels?.['listen_addr'] ?? '';
    const appId = valueField.labels?.['app_id'] ?? '';
    if (listenAddr && appId && !map.has(listenAddr)) {
      map.set(listenAddr, appId);
    }
  }
  return map;
}

function resolveIp(ip: string, ipMap: Map<string, string>): string | null {
  const exact = ipMap.get(ip);
  if (exact) {
    return exact;
  }
  // Try matching just the IP part (without port) against listen_addrs
  const colonIdx = ip.lastIndexOf(':');
  if (colonIdx > 0) {
    const ipOnly = ip.substring(0, colonIdx);
    for (const [addr, appId] of ipMap) {
      if (addr.startsWith(ipOnly + ':')) {
        return appId;
      }
    }
  }
  return null;
}

/**
 * Resolve a destination to a named app_id.
 * Tries: dest directly, then actual_destination (the real pod IP behind a ClusterIP).
 */
function resolveDestination(dest: string, actualDest: string, ipMap: Map<string, string>): string {
  if (!dest) {
    return dest;
  }
  // Already a named app_id
  if (dest.startsWith('/') || (dest.includes(':') && !dest.match(/^\d/))) {
    return dest;
  }
  // Try resolving the destination IP directly
  const fromDest = resolveIp(dest, ipMap);
  if (fromDest) {
    return fromDest;
  }
  // For ClusterIP destinations, try the actual_destination (real pod IP)
  if (actualDest) {
    const fromActual = resolveIp(actualDest, ipMap);
    if (fromActual) {
      return fromActual;
    }
  }
  return dest;
}

export function transformToServiceMap(
  requestFrames: DataFrame[],
  latencyFrames: DataFrame[],
  bytesSentFrames: DataFrame[],
  bytesRecvFrames: DataFrame[],
  tcpFailedFrames: DataFrame[],
  ipMap: Map<string, string>,
  opts: ServiceMapOptions
): ServiceMapData {
  const edgeMap = new Map<string, RawEdgeAcc>();
  const nodeIds = new Set<string>();

  function addToEdge(app: string, dest: string, actualDest: string): string {
    const resolvedDest = resolveDestination(dest, actualDest, ipMap);
    const key = `${app}→${resolvedDest}`;
    if (app) {
      nodeIds.add(app);
    }
    if (resolvedDest) {
      nodeIds.add(resolvedDest);
    }
    return key;
  }

  function getOrCreateEdge(key: string): RawEdgeAcc {
    let acc = edgeMap.get(key);
    if (!acc) {
      acc = { rps: 0, errRps: 0, latency: 0, bytesIn: 0, bytesOut: 0, tcpFailed: 0 };
      edgeMap.set(key, acc);
    }
    return acc;
  }

  // L7 requests
  const reqRows = extractLabeledSeries(requestFrames);
  for (const row of reqRows) {
    const key = addToEdge(row.app, row.dest, row.actualDest);
    const acc = getOrCreateEdge(key);
    acc.rps += row.value;
    if (row.status) {
      const isOk = row.status === '2xx' || row.status === '200' || row.status === 'ok'
        || row.status === '1xx' || row.status === '3xx';
      if (!isOk) {
        acc.errRps += row.value;
      }
    }
  }

  // Latency
  const latRows = extractLabeledSeries(latencyFrames);
  for (const row of latRows) {
    const key = addToEdge(row.app, row.dest, row.actualDest);
    const acc = edgeMap.get(key);
    if (acc) {
      acc.latency = Math.max(acc.latency, row.value);
    }
  }

  // TCP bytes sent
  const sentRows = extractLabeledSeries(bytesSentFrames);
  for (const row of sentRows) {
    const key = addToEdge(row.app, row.dest, row.actualDest);
    const acc = edgeMap.get(key);
    if (acc) {
      acc.bytesOut += row.value;
    }
  }

  // TCP bytes received
  const recvRows = extractLabeledSeries(bytesRecvFrames);
  for (const row of recvRows) {
    const key = addToEdge(row.app, row.dest, row.actualDest);
    const acc = edgeMap.get(key);
    if (acc) {
      acc.bytesIn += row.value;
    }
  }

  // TCP failed
  const failRows = extractLabeledSeries(tcpFailedFrames);
  for (const row of failRows) {
    const key = addToEdge(row.app, row.dest, row.actualDest);
    const acc = edgeMap.get(key);
    if (acc) {
      acc.tcpFailed += row.value;
    }
  }

  // Build edges — skip self-loops and below-threshold edges
  const edges: ServiceEdge[] = [];
  for (const [key, acc] of edgeMap.entries()) {
    if (acc.rps < opts.minEdgeWeight && acc.bytesOut === 0 && acc.bytesIn === 0) {
      continue;
    }
    const [source, target] = key.split('→');
    if (source === target) {
      continue;
    }
    const errPct = acc.rps > 0 ? (acc.errRps / acc.rps) * 100 : 0;
    edges.push({
      id: key,
      source,
      target,
      rps: acc.rps,
      errPct,
      p95Ms: acc.latency,
      bytesIn: acc.bytesIn,
      bytesOut: acc.bytesOut,
      tcpFailed: acc.tcpFailed,
      health: computeHealth(errPct, acc.tcpFailed, acc.rps, opts),
    });
  }

  // Aggregate node-level metrics from edges
  const nodeMetrics = new Map<string, {
    outRps: number; inRps: number;
    outErrRps: number; inErrRps: number;
    bytesIn: number; bytesOut: number;
    latency: number; tcpFailed: number;
  }>();

  function getOrInitNode(nid: string) {
    let nm = nodeMetrics.get(nid);
    if (!nm) {
      nm = { outRps: 0, inRps: 0, outErrRps: 0, inErrRps: 0, bytesIn: 0, bytesOut: 0, latency: 0, tcpFailed: 0 };
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

  // Build nodes
  const nodes: ServiceNode[] = [];
  const namespaceSet = new Set<string>();
  for (const id of nodeIds) {
    const parsed = parseAppId(id);
    namespaceSet.add(parsed.namespace);
    const nm = nodeMetrics.get(id) ?? {
      outRps: 0, inRps: 0, outErrRps: 0, inErrRps: 0,
      bytesIn: 0, bytesOut: 0, latency: 0, tcpFailed: 0,
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
